package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/user"
)

func TestCreateUserInOrg_ForbiddenWhenNotOrgAdmin(t *testing.T) {
	db := setupTestDB(t)
	h := user.NewHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// no current user in context -> should be forbidden
	body := map[string]string{"email": "bob@example.com", "password": "pw", "fullName": "Bob"}
	b, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateUserInOrg(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden when no current user, got %d", w.Code)
	}
}

func TestListUsers_PaginationAndAccess(t *testing.T) {
	db := setupTestDB(t)

	// create an organization and a user in it
	org := int64(1)
	u1 := user.User{Email: "u1@example.com", FullName: "U1", UserType: user.UserTypeOrgUser, OrganizationID: &org, Active: true}
	u2 := user.User{Email: "u2@example.com", FullName: "U2", UserType: user.UserTypeOrgUser, OrganizationID: &org, Active: true}
	if err := db.Create(&u1).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if err := db.Create(&u2).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	h := user.NewHandler(db)

	// as org admin
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	cu := auth.CurrentUser{ID: 10, UserType: auth.UserTypeOrgUser, OrganizationID: &org}
	// set role to admin
	role := auth.OrgRole("ADMIN")
	cu.OrgRole = &role
	c.Set(auth.ContextUserKey, cu)

	// request with pagination params
	c.Request = httptest.NewRequest(http.MethodGet, "/users?limit=1&page=1", nil)

	h.ListUsers(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("data missing in response")
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 user in paginated data, got %d", len(data))
	}

	// test GetUserByID success
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Set(auth.ContextUserKey, cu)
	c2.Request = httptest.NewRequest(http.MethodGet, "/users/"+strconv.FormatInt(u1.ID, 10), nil)
	// set param manually
	c2.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(u1.ID, 10)}}
	h.GetUserByID(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for GetUserByID, got %d", w2.Code)
	}
}
