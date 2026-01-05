package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	orgModel "github.com/username/fms-api/internal/organization"
	"github.com/username/fms-api/internal/vehicle"
)

func TestCreateVehicle_WithoutDevice_Success(t *testing.T) {
	db := setupTestDB(t)
	vh := vehicle.NewHandler(db)

	// create organization row in DB so vehicle creation checks succeed
	org := orgModel.Organization{Name: "O", Active: true}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("failed to create org: %v", err)
	}
	orgID := org.ID

	// create a super admin current user
	cu := auth.CurrentUser{ID: 1, UserType: auth.UserTypeSuperAdmin}

	// create router and register routes so unexported handlers are reachable
	router := gin.New()
	router.POST("/vehicles", func(c *gin.Context) { c.Set(auth.ContextUserKey, cu); vh.CreateVehicle(c) })

	req := vehicle.VehicleCreateRequest{OrganizationID: orgID, PlateNumber: "B 1234", VIN: "VIN1", Name: "V1", VehicleType: "TRUCK"}
	b, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/vehicles", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 when creating vehicle, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestGetCurrentPosition_NotFound(t *testing.T) {
	db := setupTestDB(t)
	vh := vehicle.NewHandler(db)
	// register routes on a router and call the endpoint via HTTP
	router := gin.New()
	vh.RegisterRoutes(router)

	// 1) no position -> 404
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/vehicles/1/current-position", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when no position, got %d", w.Code)
	}

	// insert a position and try again
	now := time.Now().UTC()
	db.Exec("INSERT INTO vehicle_current_position (vehicle_id, ts, lat, lon, updated_at) VALUES (?, ?, ?, ?, ?)", 1, now, -6.2, 106.8, now)

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/vehicles/1/current-position", nil)
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 after inserting position, got %d", w2.Code)
	}
}
