package vehicle

import "time"

// Model untuk tabel vehicles
type Vehicle struct {
	ID             int64  `json:"id"             gorm:"column:id;primaryKey"`
	OrganizationID int64  `json:"organizationId" gorm:"column:organization_id"`
	PlateNumber    string `json:"plateNumber"    gorm:"column:plate_number"` // FIXED
	VIN            string `json:"vin"            gorm:"column:vin"`          // FIXED
	Name           string `json:"name"           gorm:"column:name"`         // editable by ADMIN
	VehicleType    string `json:"vehicleType"    gorm:"column:vehicle_type"`
	Active         bool   `json:"active"         gorm:"column:active"`
}

func (Vehicle) TableName() string {
	return "vehicles"
}

// dipakai SUPER_ADMIN saat create vehicle
type VehicleCreateRequest struct {
	OrganizationID int64  `json:"organizationId"`
	PlateNumber    string `json:"plateNumber"`
	VIN            string `json:"vin"`
	Name           string `json:"name"`
	VehicleType    string `json:"vehicleType"`
	DeviceID       *int64 `json:"deviceId,omitempty"` // optional
}

// dipakai Org Admin saat update simple data
type VehicleUpdateRequest struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
	// ❌ tidak ada VIN, PlateNumber, DeviceID → supaya tidak bisa diubah
}

// Model GORM untuk tabel vehicle_current_position
type VehicleCurrentPositionDB struct {
	VehicleID  int64     `gorm:"column:vehicle_id"`
	DeviceID   *int64    `gorm:"column:device_id"`
	TS         time.Time `gorm:"column:ts"`
	Lat        float64   `gorm:"column:lat"`
	Lon        float64   `gorm:"column:lon"`
	SpeedKph   *float64  `gorm:"column:speed_kph"`
	HeadingDeg *float64  `gorm:"column:heading_deg"`
	IgnitionOn *bool     `gorm:"column:ignition_on"`
	OdometerKm *float64  `gorm:"column:odometer_km"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (VehicleCurrentPositionDB) TableName() string {
	return "vehicle_current_position"
}

// Struct untuk response JSON posisi terkini
type VehicleCurrentPosition struct {
	VehicleID  int64    `json:"vehicleId"`
	DeviceID   *int64   `json:"deviceId,omitempty"`
	TS         string   `json:"ts"`
	Lat        float64  `json:"lat"`
	Lon        float64  `json:"lon"`
	SpeedKph   *float64 `json:"speedKph,omitempty"`
	HeadingDeg *float64 `json:"headingDeg,omitempty"`
	IgnitionOn *bool    `json:"ignitionOn,omitempty"`
	OdometerKm *float64 `json:"odometerKm,omitempty"`
	UpdatedAt  string   `json:"updatedAt"`
}
