package device

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/username/fms-api/internal/auth"
	// ambil model VehicleDevice (cek kendaraan akan dilakukan tanpa import package vehicle untuk menghindari import cycle)
)

func (h *Handler) RebindDevice(c *gin.Context) {
	// cek super admin
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh melakukan rebind device",
		})
		return
	}

	// ambil deviceId dari path
	deviceIdStr := c.Param("deviceId")
	deviceID, err := strconv.ParseInt(deviceIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_device_id"})
		return
	}

	// ambil body
	var req struct {
		VehicleID int64 `json:"vehicleId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.VehicleID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}

	// cek device ada
	var dev Device
	if err := h.DB.First(&dev, deviceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device_not_found"})
		return
	}

	// cek kendaraan tujuan ada (gunakan struct minimal dan Table("vehicles") supaya tidak perlu import package vehicle)
	var sv struct {
		ID int64 `gorm:"column:id"`
	}
	if err := h.DB.Table("vehicles").First(&sv, req.VehicleID).Error; err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "vehicle_not_found"})
		return
	}

	now := time.Now()

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 1) matikan mapping lama
		if err := tx.Model(&VehicleDevice{}).
			Where("device_id = ? AND active = TRUE", deviceID).
			Updates(map[string]interface{}{
				"active":        false,
				"unassigned_at": now,
			}).Error; err != nil {
			return err
		}

		// 2) buat mapping baru
		newMapping := VehicleDevice{
			DeviceID:   deviceID,
			VehicleID:  req.VehicleID,
			Active:     true,
			AssignedAt: now,
		}

		if err := tx.Create(&newMapping).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "device berhasil direbind",
		"deviceId":  deviceID,
		"vehicleId": req.VehicleID,
	})
}
