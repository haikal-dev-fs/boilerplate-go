package device

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	// ganti module sesuai go.mod kamu
)

// Handler untuk device & data source
type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// ========= REGISTER ROUTES (HANYA UNTUK /admin GROUP) =========

func (h *Handler) RegisterAdminRoutes(r gin.IRoutes) {
	r.POST("/data-sources", h.CreateDataSource)
	r.GET("/data-sources", h.ListDataSources)
	r.GET("/data-sources/:id", h.GetDataSource)
	r.PUT("/data-sources/:id", h.UpdateDataSource)
	r.DELETE("/data-sources/:id", h.DeleteDataSource)

	r.POST("/devices", h.CreateDevice)
	r.GET("/devices", h.ListDevices)
	r.GET("/devices/unassigned", h.ListUnassignedDevices)
	r.PUT("/devices/:deviceId/rebind", h.RebindDevice)
	r.GET("/devices/:deviceId", h.GetDevice)
	r.PUT("/devices/:deviceId", h.UpdateDevice)
	r.DELETE("/devices/:deviceId", h.DeleteDevice)
}
