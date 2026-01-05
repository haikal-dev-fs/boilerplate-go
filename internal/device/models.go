package device

import (
	"time"

	"gorm.io/datatypes"
)

// Tabel data_sources
type DataSource struct {
	ID        int64     `json:"id"        gorm:"column:id;primaryKey"`
	Name      string    `json:"name"      gorm:"column:name"`
	Code      string    `json:"code"      gorm:"column:code"`
	Type      string    `json:"type"      gorm:"column:type"` // contoh: "DEVICE", "API"
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (DataSource) TableName() string {
	return "data_sources"
}

// Tabel devices
type Device struct {
	ID           int64             `json:"id"           gorm:"column:id;primaryKey"`
	DataSourceID int64             `json:"dataSourceId" gorm:"column:data_source_id"`
	ExternalID   string            `json:"externalId"   gorm:"column:external_id"`
	SimNumber    string            `json:"simNumber"    gorm:"column:sim_number"`
	Model        string            `json:"model"        gorm:"column:model"`
	Protocol     string            `json:"protocol"     gorm:"column:protocol"`
	Active       bool              `json:"active"       gorm:"column:active"`
	Metadata     datatypes.JSONMap `json:"metadata"     gorm:"column:metadata"`
	CreatedAt    time.Time         `json:"createdAt"    gorm:"column:created_at"`
}

func (Device) TableName() string {
	return "devices"
}

// Mapping vehicle <-> device (tabel vehicle_devices yang sudah ada)
type VehicleDevice struct {
	ID           int64      `json:"id"          gorm:"column:id;primaryKey"`
	VehicleID    int64      `json:"vehicleId"   gorm:"column:vehicle_id"`
	DeviceID     int64      `json:"deviceId"    gorm:"column:device_id"`
	Active       bool       `json:"active"      gorm:"column:active"`
	AssignedAt   time.Time  `json:"assignedAt"  gorm:"column:assigned_at"`
	UnassignedAt *time.Time `json:"unassignedAt,omitempty" gorm:"column:unassigned_at"`
}

func (VehicleDevice) TableName() string {
	return "vehicle_devices"
}
