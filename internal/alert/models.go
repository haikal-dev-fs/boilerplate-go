package alert

import (
	"time"

	"gorm.io/datatypes"
)

// Model untuk tabel alerts
type Alert struct {
	ID             int64             `json:"id"            gorm:"column:id;primaryKey"`
	VehicleID      int64             `json:"vehicleId"     gorm:"column:vehicle_id"`
	DeviceID       *int64            `json:"deviceId"      gorm:"column:device_id"`
	AlertTypeID    int64             `json:"alertTypeId"   gorm:"column:alert_type_id"`
	StartedAt      time.Time         `json:"startedAt"     gorm:"column:started_at"`
	EndedAt        *time.Time        `json:"endedAt"       gorm:"column:ended_at"`
	Status         string            `json:"status"        gorm:"column:status"` // ACTIVE / CLEARED / ACK
	Message        *string           `json:"message"       gorm:"column:message"`
	Payload        datatypes.JSONMap `json:"payload"       gorm:"column:payload"`
	CreatedAt      time.Time         `json:"createdAt"     gorm:"column:created_at"`
	AcknowledgedAt *time.Time        `json:"acknowledgedAt" gorm:"column:acknowledged_at"`
}

func (Alert) TableName() string {
	return "alerts"
}
