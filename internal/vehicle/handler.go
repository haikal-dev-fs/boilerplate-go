package vehicle

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/device"
	"github.com/username/fms-api/internal/organization"
	"github.com/username/fms-api/internal/pagination"
)

type UserRole string

const (
	RoleAdmin UserRole = "ADMIN"
	RoleUser  UserRole = "USER"
)

type CurrentUser struct {
	ID             int64
	OrganizationID int64
	Role           UserRole
}

const ctxUserKey = "currentUser"

// Helper ambil user dari gin.Context
func getCurrentUser(c *gin.Context) (*CurrentUser, bool) {
	val, ok := c.Get(ctxUserKey)
	if !ok {
		return nil, false
	}
	user, ok := val.(*CurrentUser)
	return user, ok
}

// Helper pastikan user admin
func requireAdmin(c *gin.Context) (*CurrentUser, bool) {
	user, ok := getCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "user belum login",
		})
		return nil, false
	}
	if user.Role != RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya admin yang boleh melakukan operasi ini",
		})
		return nil, false
	}
	return user, true
}

// Handler menampung dependency untuk handler kendaraan
type Handler struct {
	DB *gorm.DB
}

// NewHandler membuat handler baru
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// RegisterRoutes mendaftarkan semua route kendaraan ke mux
func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	router.GET("/vehicles", h.listVehicles)
	router.POST("/vehicles", h.CreateVehicle)
	router.GET("/vehicles/:id", h.getVehicleByID)
	router.PUT("/vehicles/:id", h.updateVehicle)
	router.DELETE("/vehicles/:id", h.deleteVehicle)

	router.GET("/vehicles/:id/current-position", h.getCurrentPosition)
}

// -------------------------------------
// Implementasi logika bisnis
// -------------------------------------

func parseIDParam(c *gin.Context) (int64, bool) {
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

func (h *Handler) listVehicles(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var vehicles []Vehicle
	var total int64

	query := h.DB.Model(&Vehicle{})
	// SUPER_ADMIN boleh lihat semua kendaraan
	if !cu.IsSuperAdmin() {
		// Org Admin & Org User → hanya lihat kendaraan org sendiri
		if cu.OrganizationID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "no_org_access"})
			return
		}
		query = query.Where("organization_id = ?", *cu.OrganizationID)
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if err := query.Order("id").Limit(p.Limit).Offset(p.Offset).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	// For each vehicle, try to fetch current position (if any) and include in response
	type VehicleWithPosition struct {
		Vehicle
		Lat       *float64 `json:"lat,omitempty" gorm:"-"`
		Lon       *float64 `json:"lon,omitempty" gorm:"-"`
		TS        *string  `json:"ts,omitempty" gorm:"-"`
		UpdatedAt *string  `json:"updatedAt,omitempty" gorm:"-"`
	}

	var resp []VehicleWithPosition
	for _, v := range vehicles {
		vp := VehicleWithPosition{Vehicle: v}

		var rec VehicleCurrentPositionDB
		if err := h.DB.Where("vehicle_id = ?", v.ID).First(&rec).Error; err == nil {
			lat := rec.Lat
			lon := rec.Lon
			ts := rec.TS.Format(time.RFC3339)
			updated := rec.UpdatedAt.Format(time.RFC3339)
			vp.Lat = &lat
			vp.Lon = &lon
			vp.TS = &ts
			vp.UpdatedAt = &updated
		}
		resp = append(resp, vp)
	}

	c.JSON(http.StatusOK, gin.H{"data": resp, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}

func (h *Handler) CreateVehicle(c *gin.Context) {
	// 1. Ambil current user
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "user not in context",
		})
		return
	}

	// 2. Hanya SUPER_ADMIN yang boleh insert kendaraan
	if !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh menambah kendaraan",
		})
		return
	}

	// 3. Ambil body request
	var req VehicleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "body bukan JSON valid",
		})
		return
	}

	if strings.TrimSpace(req.PlateNumber) == "" || strings.TrimSpace(req.VIN) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "plateNumber dan vin wajib diisi",
		})
		return
	}

	// 4. Cek organization ada & aktif
	var org organization.Organization
	if err := h.DB.Where("id = ? AND active = TRUE", req.OrganizationID).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "invalid_organization",
				"message": "organization tidak ditemukan atau tidak aktif",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// 5. Kalau TIDAK ada deviceId → buat vehicle saja, tanpa mapping device
	if req.DeviceID == nil {
		v := Vehicle{
			OrganizationID: req.OrganizationID,
			PlateNumber:    req.PlateNumber,
			VIN:            req.VIN,
			Name:           req.Name,
			VehicleType:    req.VehicleType,
			Active:         true,
		}

		if err := h.DB.Create(&v).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "db_error",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, v)
		return
	}

	// 6. Kalau ADA deviceId → jalankan logic validasi device + mapping seperti sebelumnya

	deviceID := *req.DeviceID

	// 6a. Cek device ada & aktif
	var dev device.Device
	if err := h.DB.Where("id = ? AND active = TRUE", deviceID).First(&dev).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "invalid_device",
				"message": "device tidak ditemukan atau tidak aktif",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// 6b. Pastikan device belum terikat ke kendaraan lain (mapping aktif)
	var existingMapping device.VehicleDevice
	err := h.DB.Where("device_id = ? AND active = TRUE", deviceID).First(&existingMapping).Error
	if err == nil {
		// artinya mapping aktif sudah ada
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "device_already_assigned",
			"message": "device sudah terikat ke kendaraan lain",
		})
		return
	} else if err != nil && err != gorm.ErrRecordNotFound {
		// error lain
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// 6c. Jalankan dalam transaksi: create vehicle + mapping
	var createdVehicle Vehicle
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		v := Vehicle{
			OrganizationID: req.OrganizationID,
			PlateNumber:    req.PlateNumber,
			VIN:            req.VIN,
			Name:           req.Name,
			VehicleType:    req.VehicleType,
			Active:         true,
		}

		if err := tx.Create(&v).Error; err != nil {
			return err
		}

		// buat mapping vehicle <-> device
		mapping := device.VehicleDevice{
			VehicleID:  v.ID,
			DeviceID:   deviceID,
			Active:     true,
			AssignedAt: time.Now(),
		}

		if err := tx.Create(&mapping).Error; err != nil {
			return err
		}

		createdVehicle = v
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, createdVehicle)
}

func (h *Handler) getVehicleByID(c *gin.Context) {
	// 1. Ambil current user dari context (di-set oleh AuthMiddleware)
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "user not in context",
		})
		return
	}

	// 2. Ambil ID kendaraan dari URL /api/vehicles/:id
	id, ok := parseIDParam(c)
	if !ok {
		return // response sudah dikirim di parseIDParam
	}

	// 3. Ambil data kendaraan dari database
	var v Vehicle
	if err := h.DB.First(&v, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "kendaraan tidak ditemukan",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// 4. Aturan akses:
	//    - SUPER_ADMIN boleh lihat semua kendaraan
	//    - Org Admin / Org User: hanya boleh lihat kendaraan org-nya sendiri

	if cu.IsSuperAdmin() {
		c.JSON(http.StatusOK, v)
		return
	}

	// buat org user / admin
	if cu.OrganizationID == nil || v.OrganizationID != *cu.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "tidak boleh melihat kendaraan organisasi lain",
		})
		return
	}

	// attach current position if available
	var rec VehicleCurrentPositionDB
	if err := h.DB.Where("vehicle_id = ?", v.ID).First(&rec).Error; err == nil {
		resp := gin.H{
			"id":             v.ID,
			"organizationId": v.OrganizationID,
			"plateNumber":    v.PlateNumber,
			"vin":            v.VIN,
			"name":           v.Name,
			"vehicleType":    v.VehicleType,
			"active":         v.Active,
			"currentPosition": gin.H{
				"lat":       rec.Lat,
				"lon":       rec.Lon,
				"ts":        rec.TS.Format(time.RFC3339),
				"updatedAt": rec.UpdatedAt.Format(time.RFC3339),
			},
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	c.JSON(http.StatusOK, v)
}

func (h *Handler) updateVehicle(c *gin.Context) {
	// 1. Ambil current user
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "user not in context",
		})
		return
	}

	// 2. Hanya ORG_ADMIN yang boleh edit
	if !cu.IsOrgAdmin() || cu.OrganizationID == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya ORG ADMIN yang boleh mengedit kendaraan",
		})
		return
	}

	// 3. Ambil ID kendaraan dari path
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	// 4. Bind request
	var req VehicleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "body bukan JSON valid",
		})
		return
	}

	// 5. Ambil kendaraan dari DB
	var v Vehicle
	if err := h.DB.First(&v, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "kendaraan tidak ditemukan",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// 6. Pastikan kendaraan milik organisasi user
	if v.OrganizationID != *cu.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "tidak boleh mengedit kendaraan organisasi lain",
		})
		return
	}

	// 7. Terapkan hanya field yang diperbolehkan
	if req.Name != nil {
		v.Name = *req.Name
	}
	if req.Active != nil {
		v.Active = *req.Active
	}

	// VIN, PlateNumber, DeviceID -> TIDAK TERSENTUH di sini

	if err := h.DB.Save(&v).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, v)
}

func (h *Handler) deleteVehicle(c *gin.Context) {

	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "user not in context",
		})
		return
	}

	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	if !cu.IsOrgAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya ADMIN yang boleh mengedit kendaraan",
		})
		return
	}

	var v Vehicle
	if err := h.DB.First(&v, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "kendaraan tidak ditemukan",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	if err := h.DB.Save(&v).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) getCurrentPosition(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var rec VehicleCurrentPositionDB
	err := h.DB.Where("vehicle_id = ?", id).First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "posisi kendaraan tidak ditemukan",
		})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	resp := VehicleCurrentPosition{
		VehicleID:  rec.VehicleID,
		DeviceID:   rec.DeviceID,
		TS:         rec.TS.Format(time.RFC3339),
		Lat:        rec.Lat,
		Lon:        rec.Lon,
		SpeedKph:   rec.SpeedKph,
		HeadingDeg: rec.HeadingDeg,
		IgnitionOn: rec.IgnitionOn,
		OdometerKm: rec.OdometerKm,
		UpdatedAt:  rec.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, resp)

}
