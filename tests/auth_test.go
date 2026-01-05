package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/user"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)

	// create an active user
	password := "secret123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	u := user.User{
		Email:        "alice@example.com",
		PasswordHash: string(hash),
		FullName:     "Alice",
		UserType:     user.UserTypeOrgUser,
		Active:       true,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	h := auth.NewHandler(db)

	// prepare gin
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]string{"email": "alice@example.com", "password": password}
	b, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(b))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)

	if w.Code != 200 {
		t.Fatalf("expected 200 OK, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if tok, ok := resp["token"].(string); !ok || tok == "" {
		t.Fatalf("expected token in response")
	}
	if userObj, ok := resp["user"].(map[string]interface{}); !ok {
		t.Fatalf("expected user object in response")
	} else {
		if userObj["email"] != "alice@example.com" {
			t.Fatalf("expected user email alice@example.com, got %v", userObj["email"])
		}
	}
}

func TestLogin_BadJSON(t *testing.T) {
	db := setupTestDB(t)
	h := auth.NewHandler(db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("notjson")))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Login(c)
	if w.Code != 400 {
		t.Fatalf("expected 400 for bad json, got %d", w.Code)
	}
}
