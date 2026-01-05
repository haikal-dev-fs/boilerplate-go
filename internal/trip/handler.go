package trip

import (
	"net/http"
	"os"
	"strconv"
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

// daftarkan route trip
func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	// path : /vehicles/:id/trips
	router.GET("/vehicles/:id/trips", h.listVehicleTrips)
	// search trips across vehicles user can access
	router.GET("/trips", h.listTrips)
	// get trip detail including position logs
	router.GET("/trips/:id", h.GetTripDetail)
}

// helper ambil id vehicle dari param
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

func (h *Handler) listVehicleTrips(c *gin.Context) {
	vehicleID, ok := parseVehicleID(c)
	if !ok {
		return
	}

	// baca query param + pagination
	fromStr := c.Query("from")
	toStr := c.Query("to")
	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var fromTime, toTime time.Time
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err == nil {
			fromTime = t
		}
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err == nil {
			toTime = t
		}
	}

	// if no range provided, default to last 1 day
	if fromTime.IsZero() && toTime.IsZero() {
		toTime = time.Now().UTC()
		fromTime = toTime.Add(-24 * time.Hour)
	}

	// enforce max range via env MAX_RANGE_DAYS (default 7)
	maxDays := 7
	if v := os.Getenv("MAX_RANGE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxDays = n
		}
	}
	if !toTime.IsZero() && !fromTime.IsZero() {
		if toTime.Before(fromTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "to must be after from"})
			return
		}
		if toTime.Sub(fromTime) > time.Duration(maxDays)*24*time.Hour {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "requested range exceeds max range"})
			return
		}
	}

	// Build query GORM
	query := h.DB.Model(&Trip{}).Where("vehicle_id = ?", vehicleID)

	if !fromTime.IsZero() {
		query = query.Where("start_ts >= ?", fromTime)
	}
	if !toTime.IsZero() {
		query = query.Where("end_ts <= ?", toTime)
	}

	// Hitung total dulu (tanpa limit)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// Ambil data dengan limit/offset, urut terbaru dulu
	var trips []Trip
	if err := query.
		Order("start_ts DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&trips).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       trips,
		"pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit},
	})

}

// listTrips returns trips across vehicles the current user can access.
// Query params: from, to (RFC3339). If not provided, defaults to last 1 day.
// Enforces max range (days) via env `MAX_RANGE_DAYS` (default 7).
func (h *Handler) listTrips(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

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
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid from parameter, must be RFC3339"})
				return
			}
			fromTime = t
		}
		if toStr != "" {
			t, err := time.Parse(time.RFC3339, toStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid to parameter, must be RFC3339"})
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

	// enforce max range
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

	// build query joining vehicles to filter by organization if needed
	// Note: do not use Select("t.*") when counting because GORM may translate Count incorrectly.
	base := h.DB.Table("trips t").Joins("JOIN vehicles v ON v.id = t.vehicle_id").Where("t.start_ts >= ? AND t.start_ts <= ?", fromTime, toTime)
	if !cu.IsSuperAdmin() {
		if cu.OrganizationID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "no_org_access"})
			return
		}
		base = base.Where("v.organization_id = ?", *cu.OrganizationID)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	var trips []Trip
	// select trip columns for the final fetch
	qry := base.Select("t.*")
	if err := qry.Order("t.start_ts DESC").Limit(p.Limit).Offset(p.Offset).Scan(&trips).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": trips, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}

// GetTripDetail returns a trip by id and includes position_log entries between start and end timestamps
func (h *Handler) GetTripDetail(c *gin.Context) {
	// parse id
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "id harus berupa angka"})
		return
	}

	// fetch trip
	var tr Trip
	if err := h.DB.First(&tr, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	// auth: check access
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if !cu.IsSuperAdmin() {
		// fetch vehicle org
		var v struct{ OrganizationID *int64 }
		if err := h.DB.Table("vehicles").Select("organization_id").Where("id = ?", tr.VehicleID).First(&v).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
			return
		}
		if v.OrganizationID == nil || cu.OrganizationID == nil || *v.OrganizationID != *cu.OrganizationID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	// query position_log between start_ts and end_ts
	type positionRecord struct {
		ID         int64     `json:"id" gorm:"column:id"`
		VehicleID  int64     `json:"vehicleId" gorm:"column:vehicle_id"`
		DeviceID   *int64    `json:"deviceId" gorm:"column:device_id"`
		TS         time.Time `json:"ts" gorm:"column:ts"`
		Lat        float64   `json:"lat" gorm:"column:lat"`
		Lon        float64   `json:"lon" gorm:"column:lon"`
		SpeedKph   *float64  `json:"speedKph,omitempty" gorm:"column:speed_kph"`
		HeadingDeg *float64  `json:"headingDeg,omitempty" gorm:"column:heading_deg"`
		AltitudeM  *float64  `json:"altitudeM,omitempty" gorm:"column:altitude_m"`
		IgnitionOn *bool     `json:"ignitionOn,omitempty" gorm:"column:ignition_on"`
		OdometerKm *float64  `json:"odometerKm,omitempty" gorm:"column:odometer_km"`
		CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at"`
	}

	var positions []positionRecord
	if err := h.DB.Table("position_log").Where("vehicle_id = ? AND ts >= ? AND ts <= ?", tr.VehicleID, tr.StartTs, tr.EndTs).Order("ts ASC").Find(&positions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trip": tr, "positions": positions})
}
