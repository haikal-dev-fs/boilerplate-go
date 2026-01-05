package organization

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/pagination"
	userModel "github.com/username/fms-api/internal/user"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

type CreateOrgRequest struct {
	Name          string  `json:"name"`
	Code          *string `json:"code"`
	AdminEmail    string  `json:"adminEmail"`
	AdminPassword string  `json:"adminPassword"`
	AdminFullName string  `json:"adminFullName"`
}

type CreateOrgResponse struct {
	Organization Organization         `json:"organization"`
	AdminUser    OrganizationAdminDTO `json:"adminUser"`
}

type OrganizationAdminDTO struct {
	ID             int64  `json:"id"`
	Email          string `json:"email"`
	FullName       string `json:"fullName"`
	UserType       string `json:"userType"`
	OrganizationID int64  `json:"organizationId"`
	OrgRole        string `json:"orgRole"`
}

func (h *Handler) RegisterAdminRoutes(r gin.IRoutes) {
	r.POST("/organizations", h.CreateOrganizationWithAdmin)
	r.GET("/organizations", h.ListOrganizations)
	r.GET("/organizations/:id", h.GetOrganizationByID)
	r.PUT("/organizations/:id", h.UpdateOrganization)
	r.DELETE("/organizations/:id", h.DeleteOrganization)
}

type UpdateOrgRequest struct {
	Name *string `json:"name,omitempty"`
	Code *string `json:"code,omitempty"`
	// Active can be toggled by SUPER_ADMIN
	Active *bool `json:"active,omitempty"`
}

// ListOrganizations lists all organizations. Only SUPER_ADMIN allowed.
func (h *Handler) ListOrganizations(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "hanya SUPER_ADMIN"})
		return
	}

	p := pagination.ParsePagination(c)
	if c.IsAborted() {
		return
	}

	var orgs []Organization
	var total int64

	query := h.DB.Model(&Organization{})
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	if err := query.Order("id").Limit(p.Limit).Offset(p.Offset).Find(&orgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": orgs,
		"pagination": gin.H{
			"total":     total,
			"limit":     p.Limit,
			"page":      p.Page,
			"max_limit": p.MaxLimit,
		},
	})
}

// GetOrganizationByID returns organization by id. Only SUPER_ADMIN allowed.
func (h *Handler) GetOrganizationByID(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "hanya SUPER_ADMIN"})
		return
	}

	id := c.Param("id")
	var org Organization
	if err := h.DB.First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, org)
}

// UpdateOrganization updates fields of an organization. Only SUPER_ADMIN allowed.
func (h *Handler) UpdateOrganization(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "hanya SUPER_ADMIN"})
		return
	}

	id := c.Param("id")
	var org Organization
	if err := h.DB.First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid JSON body"})
		return
	}

	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Code != nil {
		org.Code = req.Code
	}
	if req.Active != nil {
		org.Active = *req.Active
	}

	if err := h.DB.Save(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, org)
}

// DeleteOrganization performs soft-delete by setting active = false. Only SUPER_ADMIN allowed.
func (h *Handler) DeleteOrganization(c *gin.Context) {
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "hanya SUPER_ADMIN"})
		return
	}

	id := c.Param("id")
	var org Organization
	if err := h.DB.First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	org.Active = false
	if err := h.DB.Save(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateOrganizationWithAdmin(c *gin.Context) {
	// 1. Ambil current user dan cek super admin
	cu, ok := auth.GetCurrentUser(c)
	if !ok || !cu.IsSuperAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "hanya SUPER_ADMIN yang boleh membuat organization baru",
		})
		return
	}

	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "invalid JSON body",
		})
		return
	}

	if req.Name == "" || req.AdminEmail == "" || req.AdminPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "name, adminEmail, adminPassword wajib diisi",
		})
		return
	}

	// 2. Hash password admin
	hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "hash_error",
			"message": "gagal meng-hash password",
		})
		return
	}

	// 3. Jalankan dalam transaksi
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 3a. Buat organization
		org := Organization{
			Name:   req.Name,
			Code:   req.Code,
			Active: true,
		}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}

		// 3b. Buat admin user untuk org ini
		orgID := org.ID
		role := userModel.OrgRoleAdmin

		adminUser := userModel.User{
			Email:          req.AdminEmail,
			PasswordHash:   string(hash),
			FullName:       req.AdminFullName,
			UserType:       userModel.UserTypeOrgUser,
			OrganizationID: &orgID,
			OrgRole:        &role,
			Active:         true,
		}

		if err := tx.Create(&adminUser).Error; err != nil {
			return err
		}

		// simpan ke context untuk response
		c.Set("createdOrg", org)
		c.Set("createdAdmin", adminUser)
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "db_error",
			"message": err.Error(),
		})
		return
	}

	orgAny, _ := c.Get("createdOrg")
	adminAny, _ := c.Get("createdAdmin")
	org := orgAny.(Organization)
	admin := adminAny.(userModel.User)

	resp := CreateOrgResponse{
		Organization: org,
		AdminUser: OrganizationAdminDTO{
			ID:             admin.ID,
			Email:          admin.Email,
			FullName:       admin.FullName,
			UserType:       string(admin.UserType),
			OrganizationID: *admin.OrganizationID,
			OrgRole:        string(*admin.OrgRole),
		},
	}

	c.JSON(http.StatusCreated, resp)
}
