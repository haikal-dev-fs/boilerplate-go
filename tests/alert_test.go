package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/alert"
	"github.com/username/fms-api/internal/auth"
)

func TestListVehicleAlerts_InvalidStatusAndRange(t *testing.T) {
	db := setupTestDB(t)
	h := alert.NewHandler(db)

	router := gin.New()
	h.RegisterRoutes(router)

	// invalid status
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/vehicles/1/alerts?status=INVALID", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", w.Code)
	}

	// invalid date range to > max days (use MAX_RANGE_DAYS=1 via env not set here), craft from/to
	from := time.Now().AddDate(0, 0, -10).Format(time.RFC3339)
	to := time.Now().Format(time.RFC3339)
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/vehicles/1/alerts?from="+from+"&to="+to, nil)
	router.ServeHTTP(w2, r2)
	// default maxDays is 7 so range 10 days should be rejected
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for too large range, got %d", w2.Code)
	}
}

func TestListAlerts_UnauthorizedWithoutUser(t *testing.T) {
	db := setupTestDB(t)
	h := alert.NewHandler(db)
	router := gin.New()
	h.RegisterRoutes(router)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for alerts when no user, got %d", w.Code)
	}

	// now set a current user and try
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	// use handler directly via context injection
	c, _ := gin.CreateTestContext(w2)
	cu := auth.CurrentUser{ID: 1, UserType: auth.UserTypeSuperAdmin}
	c.Set(auth.ContextUserKey, cu)
	c.Request = r2
	h.ListAlerts(c)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 when super admin lists alerts, got %d", w2.Code)
	}
}
