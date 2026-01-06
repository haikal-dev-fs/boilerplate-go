package device

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/pagination"
	"gorm.io/gorm"
)

// ========= USECASE: DATA SOURCES =========

func (h *Handler) CreateDataSource(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh membuat data source",
		})
		return
	}

	var req CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "body bukan JSON valid",
		})
		return
	}

	if req.Name == "" || req.Code == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "name, code, dan type wajib diisi",
		})
		return
	}

	ds := DataSource{
		Name: req.Name,
		Code: req.Code,
		Type: req.Type,
	}

	if err := h.DB.Create(&ds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, ds)
}

func (h *Handler) ListDataSources(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh melihat data sources",
		})
		return
	}

	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var sources []DataSource
	var total int64
	query := h.DB.Model(&DataSource{})
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if err := query.Order("id").Limit(p.Limit).Offset(p.Offset).Find(&sources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sources, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}

// GetDataSource returns a single data source by id
func (h *Handler) GetDataSource(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "hanya SUPER_ADMIN yang boleh melihat data source"})
		return
	}
	id := c.Param("id")
	var ds DataSource
	if err := h.DB.First(&ds, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ds)
}

// UpdateDataSource updates a data source. Only SUPER_ADMIN.
func (h *Handler) UpdateDataSource(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	id := c.Param("id")
	var ds DataSource
	if err := h.DB.First(&ds, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	var req UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON body"})
		return
	}
	if req.Name != nil {
		ds.Name = *req.Name
	}
	if req.Code != nil {
		ds.Code = *req.Code
	}
	if req.Type != nil {
		ds.Type = *req.Type
	}

	if err := h.DB.Save(&ds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ds)
}

// DeleteDataSource deletes a data source if no devices reference it
func (h *Handler) DeleteDataSource(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	id := c.Param("id")
	var ds DataSource
	if err := h.DB.First(&ds, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	// check devices
	var cnt int64
	if err := h.DB.Model(&Device{}).Where("data_source_id = ?", ds.ID).Count(&cnt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	if cnt > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "has_devices", "message": "data source masih memiliki device"})
		return
	}

	if err := h.DB.Delete(&ds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ========= USECASE: DEVICES =========

func (h *Handler) CreateDevice(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh membuat device",
		})
		return
	}

	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "body bukan JSON valid",
		})
		return
	}

	if req.DataSourceID == 0 || req.ExternalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "dataSourceId dan externalId wajib diisi",
		})
		return
	}

	// 1. Pastikan data_source ada
	var ds DataSource
	if err := h.DB.First(&ds, req.DataSourceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "invalid_data_source",
				"message": "data_source tidak ditemukan",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	// 2. Buat device
	dev := Device{
		DataSourceID: req.DataSourceID,
		ExternalID:   req.ExternalID,
		SimNumber:    req.SimNumber,
		Model:        req.Model,
		Protocol:     req.Protocol,
		Active:       true,
	}

	if req.Metadata != nil {
		dev.Metadata = req.Metadata
	}

	if err := h.DB.Create(&dev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dev)
}

func (h *Handler) ListDevices(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh melihat devices",
		})
		return
	}

	activeStr := c.Query("active")
	dataSourceIDStr := c.Query("dataSourceId")

	query := h.DB.Model(&Device{})

	if activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err == nil {
			query = query.Where("active = ?", active)
		}
	}
	if dataSourceIDStr != "" {
		if dsID, err := strconv.ParseInt(dataSourceIDStr, 10, 64); err == nil {
			query = query.Where("data_source_id = ?", dsID)
		}
	}

	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var devices []Device
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if err := query.Order("id").Limit(p.Limit).Offset(p.Offset).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": devices, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}

// Device aktif yang belum terikat ke kendaraan mana pun
func (h *Handler) ListUnassignedDevices(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh melihat devices unassigned",
		})
		return
	}

	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var devices []Device
	var total int64

	base := h.DB.Table("devices AS d").
		Select("d.*").
		Joins("LEFT JOIN vehicle_devices vd ON vd.device_id = d.id AND vd.active = TRUE").
		Where("d.active = TRUE AND vd.id IS NULL")

	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if err := base.Limit(p.Limit).Offset(p.Offset).Scan(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": devices, "pagination": gin.H{"total": total, "limit": p.Limit, "page": p.Page, "max_limit": p.MaxLimit}})
}

// GetDevice returns device by id
func (h *Handler) GetDevice(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "hanya SUPER_ADMIN"})
		return
	}
	idStr := c.Param("deviceId")
	deviceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_device_id"})
		return
	}

	var dev Device
	if err := h.DB.First(&dev, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	// fetch vehicles mapped to this device (active mappings)
	var vehicles []struct {
		ID             int64   `json:"id" gorm:"column:id"`
		OrganizationID *int64  `json:"organizationId,omitempty" gorm:"column:organization_id"`
		PlateNumber    string  `json:"plateNumber" gorm:"column:plate_number"`
		VIN            string  `json:"vin" gorm:"column:vin"`
		Name           *string `json:"name,omitempty" gorm:"column:name"`
		VehicleType    *string `json:"vehicleType,omitempty" gorm:"column:vehicle_type"`
		Active         bool    `json:"active" gorm:"column:active"`
	}

	if err := h.DB.Table("vehicles v").
		Select("v.id, v.organization_id, v.plate_number, v.vin, v.name, v.vehicle_type, v.active").
		Joins("JOIN vehicle_devices vd ON vd.vehicle_id = v.id AND vd.active = TRUE").
		Where("vd.device_id = ?", deviceID).
		Scan(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device":   dev,
		"vehicles": vehicles,
	})
}

// UpdateDevice updates device fields (SUPER_ADMIN)
func (h *Handler) UpdateDevice(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	id := c.Param("deviceId")
	var dev Device
	if err := h.DB.First(&dev, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON body"})
		return
	}

	if req.DataSourceID != nil {
		// check datasource exists
		var ds DataSource
		if err := h.DB.First(&ds, *req.DataSourceID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid_data_source"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
			return
		}
		dev.DataSourceID = *req.DataSourceID
	}
	if req.SimNumber != nil {
		dev.SimNumber = *req.SimNumber
	}
	if req.Model != nil {
		dev.Model = *req.Model
	}
	if req.Protocol != nil {
		dev.Protocol = *req.Protocol
	}
	if req.Metadata != nil {
		dev.Metadata = req.Metadata
	}
	if req.Active != nil {
		dev.Active = *req.Active
	}

	if err := h.DB.Save(&dev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dev)
}

// DeleteDevice performs soft-delete by setting active = false
func (h *Handler) DeleteDevice(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	id := c.Param("deviceId")
	var dev Device
	if err := h.DB.First(&dev, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	dev.Active = false
	if err := h.DB.Save(&dev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
