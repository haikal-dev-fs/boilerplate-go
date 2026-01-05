package trip

import (
	"time"

	"gorm.io/datatypes"
)

type Trip struct {
	ID              int64             `json:"id"               gorm:"column:id;primaryKey"`
	VehicleID       int64             `json:"vehicleId"        gorm:"column:vehicle_id"`
	StartTs         time.Time         `json:"startTs"          gorm:"column:start_ts"`
	EndTs           time.Time         `json:"endTs"            gorm:"column:end_ts"`
	StartLat        *float64          `json:"startLat"         gorm:"column:start_lat"`
	StartLon        *float64          `json:"startLon"         gorm:"column:start_lon"`
	EndLat          *float64          `json:"endLat"           gorm:"column:end_lat"`
	EndLon          *float64          `json:"endLon"           gorm:"column:end_lon"`
	DistanceKm      *float64          `json:"distanceKm"       gorm:"column:distance_km"`
	DurationSeconds *int64            `json:"durationSeconds"  gorm:"column:duration_seconds"`
	MaxSpeedKph     *float64          `json:"maxSpeedKph"      gorm:"column:max_speed_kph"`
	AvgSpeedKph     *float64          `json:"avgSpeedKph"      gorm:"column:avg_speed_kph"`
	Metadata        datatypes.JSONMap `json:"metadata"         gorm:"column:metadata"` // mapping JSONB
	CreatedAt       time.Time         `json:"createdAt"        gorm:"column:created_at"`
}

// nama tabel di DB
func (Trip) TableName() string {
	return "trips"
}
