package alert

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/pagination"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// Daftarkan route alerts
func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	// Sesuai swagger: GET /vehicles/{id}/alerts
	router.GET("/vehicles/:id/alerts", h.ListVehicleAlerts)
	// search alerts across vehicles user can access
	router.GET("/alerts", h.ListAlerts)
}

// helper ambil vehicle id dari path param
func parseVehicleID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "id harus berupa angka",
		})
		return 0, false
	}
	return id, true
}

func (h *Handler) ListVehicleAlerts(c *gin.Context) {
	vehicleID, ok := parseVehicleID(c)
	if !ok {
		return
	}

	// Query param:
	// ?status=ACTIVE|CLEARED|ACK (opsional)
	// pagination via ?limit & ?page
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	// parse date range; default last 1 day; enforce MAX_RANGE_DAYS
	fromStr := c.Query("from")
	toStr := c.Query("to")
	var fromTime, toTime time.Time
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid from parameter"})
			return
		}
		fromTime = t
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid to parameter"})
			return
		}
		toTime = t
	}
	if fromTime.IsZero() && toTime.IsZero() {
		toTime = time.Now().UTC()
		fromTime = toTime.Add(-24 * time.Hour)
	}
	maxDays := 7
	if v := os.Getenv("MAX_RANGE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxDays = n
		}
	}
	if toTime.Before(fromTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "to must be after from"})
		return
	}
	if toTime.Sub(fromTime) > time.Duration(maxDays)*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "requested range exceeds max range"})
		return
	}

	// Build query GORM
	query := h.DB.Model(&Alert{}).Where("vehicle_id = ?", vehicleID).Where("started_at >= ? AND started_at <= ?", fromTime, toTime)

	if status != "" {
		// validasi kasar optional: kalau bukan salah satu dari 3, bisa diabaikan / tolak
		switch status {
		case "ACTIVE", "CLEARED", "ACK":
			query = query.Where("status = ?", status)
		default:
			// kalau status tidak valid, bisa balikin error atau abaikan filter
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "bad_request",
				"message": "status harus salah satu dari: ACTIVE, CLEARED, ACK",
			})
			return
		}
	}

	// Hitung total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// Ambil data alert (urutan terbaru dulu)
	// join with vehicles to include plate_number
	type AlertWithPlate struct {
		Alert
		PlateNumber *string `json:"plateNumber" gorm:"column:plate_number"`
	}
	var alerts []AlertWithPlate
	if err := h.DB.Table("alerts a").Select("a.*, v.plate_number").Joins("JOIN vehicles v ON v.id = a.vehicle_id").Where("a.vehicle_id = ?", vehicleID).
		Order("a.started_at DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Scan(&alerts).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alerts, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}

// ListAlerts returns alerts across vehicles the current user can access.
// Query params: status (optional), from, to (RFC3339). Defaults to last 1 day. Enforces MAX_RANGE_DAYS.
func (h *Handler) ListAlerts(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	fromStr := c.Query("from")
	toStr := c.Query("to")

	now := time.Now().UTC()
	var fromTime, toTime time.Time
	if fromStr == "" && toStr == "" {
		toTime = now
		fromTime = now.Add(-24 * time.Hour)
	} else {
		if fromStr != "" {
			t, err := time.Parse(time.RFC3339, fromStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid from parameter"})
				return
			}
			fromTime = t
		}
		if toStr != "" {
			t, err := time.Parse(time.RFC3339, toStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid to parameter"})
				return
			}
			toTime = t
		}
		if fromTime.IsZero() {
			fromTime = toTime.Add(-24 * time.Hour)
		}
		if toTime.IsZero() {
			toTime = fromTime.Add(24 * time.Hour)
		}
	}

	maxDays := 7
	if v := os.Getenv("MAX_RANGE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxDays = n
		}
	}
	if toTime.Before(fromTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "to must be after from"})
		return
	}
	if toTime.Sub(fromTime) > time.Duration(maxDays)*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "requested range exceeds max range"})
		return
	}

	// base query joining vehicles
	// avoid Select("a.*") before Count to prevent invalid COUNT SQL
	base := h.DB.Table("alerts a").Joins("JOIN vehicles v ON v.id = a.vehicle_id").Where("a.started_at >= ? AND a.started_at <= ?", fromTime, toTime)
	if !cu.IsSuperAdmin() {
		if cu.OrganizationID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "no_org_access"})
			return
		}
		base = base.Where("v.organization_id = ?", *cu.OrganizationID)
	}

	if status != "" {
		switch status {
		case "ACTIVE", "CLEARED", "ACK":
			base = base.Where("a.status = ?", status)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "status harus salah satu dari: ACTIVE, CLEARED, ACK"})
			return
		}
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	// Return alerts along with vehicle plate number
	type AlertWithPlate struct {
		Alert
		PlateNumber *string `json:"plateNumber" gorm:"column:plate_number"`
	}

	var alerts []AlertWithPlate
	qry := base.Select("a.*, v.plate_number")
	if err := qry.Order("a.started_at DESC").Limit(p.Limit).Offset(p.Offset).Scan(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alerts, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}
