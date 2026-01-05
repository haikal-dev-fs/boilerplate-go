package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var jwtSecret = []byte("super-secret-dev-key") // TODO: pindah ke env

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string           `json:"token"`
	User  LoginUserPayload `json:"user"`
}

type LoginUserPayload struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	FullName       string  `json:"fullName"`
	UserType       string  `json:"userType"`
	OrganizationID *int64  `json:"organizationId,omitempty"`
	OrgRole        *string `json:"orgRole,omitempty"`
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.POST("/auth/login", h.Login)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "invalid JSON body",
		})
		return
	}

	// 1. Cari user by email
	// gunakan struct lokal agar tidak mengimpor package `user` (menghindari import cycle)
	type authUser struct {
		ID             int64
		Email          string
		PasswordHash   string
		FullName       string
		UserType       string
		OrganizationID *int64
		OrgRole        *string
		Active         bool
	}

	var u authUser
	// Use explicit table name so GORM queries the existing `users` table
	if err := h.DB.Table("users").Where("email = ? AND active = TRUE", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_credentials",
			"message": "email atau password salah",
		})
		return
	}

	// 2. Cek password
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_credentials",
			"message": "email atau password salah",
		})
		return
	}

	// 3. Siapkan claims JWT
	var orgID *int64
	if u.OrganizationID != nil {
		orgID = u.OrganizationID
	}

	var orgRoleStr *string
	if u.OrgRole != nil {
		orgRoleStr = u.OrgRole
	}

	claims := UserClaims{
		UserID:         u.ID,
		UserType:       u.UserType,
		OrganizationID: orgID,
		OrgRole:        orgRoleStr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_error",
			"message": "failed to generate token",
		})
		return
	}

	resp := LoginResponse{
		Token: tokenString,
		User: LoginUserPayload{
			ID:             u.ID,
			Email:          u.Email,
			FullName:       u.FullName,
			UserType:       u.UserType,
			OrganizationID: orgID,
			OrgRole:        orgRoleStr,
		},
	}

	c.JSON(http.StatusOK, resp)
}
