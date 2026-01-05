package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/pagination"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
	Role     string `json:"role"` // "ADMIN" atau "USER"
}

type UserDTO struct {
	ID             int64  `json:"id"`
	Email          string `json:"email"`
	FullName       string `json:"fullName"`
	UserType       string `json:"userType"`
	OrganizationID int64  `json:"organizationId"`
	OrgRole        string `json:"orgRole"`
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.POST("/users", h.CreateUserInOrg)
	r.GET("/users", h.ListUsers)
	r.GET("/users/:id", h.GetUserByID)
	r.PUT("/users/:id", h.UpdateUser)
	r.DELETE("/users/:id", h.DeleteUser)
}

type UpdateUserRequest struct {
	FullName *string `json:"fullName,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"` // "ADMIN" atau "USER"
	Active   *bool   `json:"active,omitempty"`
}

// ListUsers returns list of users. SUPER_ADMIN sees all, org admins see their org users.
func (h *Handler) ListUsers(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// parse pagination params
	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var users []User
	var total int64

	query := h.DB.Model(&User{})
	if !cu.IsSuperAdmin() {
		if cu.OrganizationID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "no_org_access"})
			return
		}
		query = query.Where("organization_id = ?", *cu.OrganizationID)
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if err := query.Order("id").Limit(p.Limit).Offset(p.Offset).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	// map to DTOs
	var out []UserDTO
	for _, u := range users {
		var orgID int64
		if u.OrganizationID != nil {
			orgID = *u.OrganizationID
		}
		var orgRole string
		if u.OrgRole != nil {
			orgRole = string(*u.OrgRole)
		}
		out = append(out, UserDTO{
			ID:             u.ID,
			Email:          u.Email,
			FullName:       u.FullName,
			UserType:       string(u.UserType),
			OrganizationID: orgID,
			OrgRole:        orgRole,
		})
	}

	// return data + pagination metadata
	resp := gin.H{
		"data": out,
		"pagination": gin.H{
			"total":     total,
			"limit":     p.Limit,
			"page":      p.Page,
			"max_limit": p.MaxLimit,
		},
	}
	c.JSON(http.StatusOK, resp)
}

// GetUserByID returns a single user with access checks
func (h *Handler) GetUserByID(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := c.Param("id")
	var u User
	if err := h.DB.First(&u, idStr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if !cu.IsSuperAdmin() {
		if cu.OrganizationID == nil || u.OrganizationID == nil || *cu.OrganizationID != *u.OrganizationID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
	}

	var orgID int64
	if u.OrganizationID != nil {
		orgID = *u.OrganizationID
	}
	var orgRole string
	if u.OrgRole != nil {
		orgRole = string(*u.OrgRole)
	}

	resp := UserDTO{
		ID:             u.ID,
		Email:          u.Email,
		FullName:       u.FullName,
		UserType:       string(u.UserType),
		OrganizationID: orgID,
		OrgRole:        orgRole,
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateUser allows org admin to update user in their org (change name, role, password, active)
func (h *Handler) UpdateUser(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsOrgAdmin() || cu.OrganizationID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	idStr := c.Param("id")
	var u User
	if err := h.DB.First(&u, idStr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	// only allow editing users within same org
	if u.OrganizationID == nil || *u.OrganizationID != *cu.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON body"})
		return
	}

	if req.FullName != nil {
		u.FullName = *req.FullName
	}
	if req.Active != nil {
		u.Active = *req.Active
	}
	if req.Role != nil {
		var r OrgRole
		if *req.Role == "ADMIN" {
			r = OrgRoleAdmin
		} else {
			r = OrgRoleUser
		}
		u.OrgRole = &r
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "hash_error", "message": err.Error()})
			return
		}
		u.PasswordHash = string(hash)
	}

	if err := h.DB.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	var orgID int64
	if u.OrganizationID != nil {
		orgID = *u.OrganizationID
	}
	var orgRole string
	if u.OrgRole != nil {
		orgRole = string(*u.OrgRole)
	}

	resp := UserDTO{
		ID:             u.ID,
		Email:          u.Email,
		FullName:       u.FullName,
		UserType:       string(u.UserType),
		OrganizationID: orgID,
		OrgRole:        orgRole,
	}
	c.JSON(http.StatusOK, resp)
}

// DeleteUser performs soft-delete by setting active = false (org admin only)
func (h *Handler) DeleteUser(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsOrgAdmin() || cu.OrganizationID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	idStr := c.Param("id")
	var u User
	if err := h.DB.First(&u, idStr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if u.OrganizationID == nil || *u.OrganizationID != *cu.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	u.Active = false
	if err := h.DB.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateUserInOrg(c *gin.Context) {
	// 1. Ambil current user (harus org admin)
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsOrgAdmin() || cu.OrganizationID == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya ORG ADMIN yang boleh membuat user baru",
		})
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "invalid JSON body",
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "email dan password wajib diisi",
		})
		return
	}

	var role OrgRole
	if req.Role == "ADMIN" {
		role = OrgRoleAdmin
	} else {
		role = OrgRoleUser
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "hash_error",
			"message": "gagal meng-hash password",
		})
		return
	}

	orgID := *cu.OrganizationID

	u := User{
		Email:          req.Email,
		PasswordHash:   string(hash),
		FullName:       req.FullName,
		UserType:       UserTypeOrgUser,
		OrganizationID: &orgID,
		OrgRole:        &role,
		Active:         true,
	}

	if err := h.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	resp := UserDTO{
		ID:             u.ID,
		Email:          u.Email,
		FullName:       u.FullName,
		UserType:       string(u.UserType),
		OrganizationID: orgID,
		OrgRole:        string(*u.OrgRole),
	}

	c.JSON(http.StatusCreated, resp)
}
