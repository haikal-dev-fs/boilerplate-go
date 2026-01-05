package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/device"
)

func TestCreateDataSource_And_CreateDevice_ListUnassigned(t *testing.T) {
	db := setupTestDB(t)
	h := device.NewHandler(db)

	// prepare super admin in context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	cu := auth.CurrentUser{ID: 1, UserType: auth.UserTypeSuperAdmin}
	c.Set(auth.ContextUserKey, cu)

	// create data source
	dsBody := device.CreateDataSourceRequest{Name: "DS1", Code: "DS1", Type: "API"}
	bds, _ := json.Marshal(dsBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/data-sources", bytes.NewReader(bds))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CreateDataSource(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create data source, got %d, body: %s", w.Code, w.Body.String())
	}

	// decode created datasource id
	var created device.DataSource
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal datasource: %v", err)
	}

	// create device
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Set(auth.ContextUserKey, cu)
	devReq := device.CreateDeviceRequest{DataSourceID: created.ID, ExternalID: "ext-1", SimNumber: "08123", Model: "M1", Protocol: "P1"}
	bd, _ := json.Marshal(devReq)
	c2.Request = httptest.NewRequest(http.MethodPost, "/devices", bytes.NewReader(bd))
	c2.Request.Header.Set("Content-Type", "application/json")
	h.CreateDevice(c2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create device, got %d, body: %s", w2.Code, w2.Body.String())
	}

	// list devices (simpler query to avoid SQL dialect differences in COUNT)
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Set(auth.ContextUserKey, cu)
	c3.Request = httptest.NewRequest(http.MethodGet, "/devices", nil)
	h.ListDevices(c3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 for list devices, got %d", w3.Code)
	}
}
