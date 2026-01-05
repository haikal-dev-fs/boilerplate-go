package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/organization"
)

func TestCreateOrganizationWithAdmin_Success(t *testing.T) {
	db := setupTestDB(t)
	h := organization.NewHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// set current user as super admin
	cu := auth.CurrentUser{ID: 1, UserType: auth.UserTypeSuperAdmin}
	c.Set(auth.ContextUserKey, cu)

	body := organization.CreateOrgRequest{
		Name:          "Org A",
		AdminEmail:    "admin@orga.example",
		AdminPassword: "pw1234",
		AdminFullName: "Org Admin",
	}
	b, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateOrganizationWithAdmin(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["organization"]; !ok {
		t.Fatalf("expected organization in response")
	}
	if _, ok := resp["adminUser"]; !ok {
		t.Fatalf("expected adminUser in response")
	}
}
