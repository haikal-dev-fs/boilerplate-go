package tests

import (
    "testing"

    "github.com/username/fms-api/internal/alert"
    "github.com/username/fms-api/internal/device"
    "github.com/username/fms-api/internal/organization"
    "github.com/username/fms-api/internal/user"
    "github.com/username/fms-api/internal/vehicle"

    gsqlite "github.com/glebarez/sqlite"
    "gorm.io/gorm"
)

// setupTestDB opens an in-memory sqlite DB and auto-migrates all models
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(gsqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open sqlite memory DB: %v", err)
    }

    // Auto migrate tables used by tests
    if err := db.AutoMigrate(
        &user.User{},
        &organization.Organization{},
        &vehicle.Vehicle{},
        &vehicle.VehicleCurrentPositionDB{},
        &device.DataSource{},
        &device.Device{},
        &device.VehicleDevice{},
        &alert.Alert{},
    ); err != nil {
        t.Fatalf("automigrate failed: %v", err)
    }

    return db
}
