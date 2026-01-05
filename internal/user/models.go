package user

import (
	"time"

	"gorm.io/gorm"
)

type UserType string
type OrgRole string

const (
	UserTypeSuperAdmin UserType = "SUPER_ADMIN"
	UserTypeOrgUser    UserType = "ORG_USER"

	OrgRoleAdmin OrgRole = "ADMIN"
	OrgRoleUser  OrgRole = "USER"
)

type User struct {
	ID             int64     `json:"id"            gorm:"column:id;primaryKey"`
	Email          string    `json:"email"         gorm:"column:email"`
	PasswordHash   string    `json:"-"             gorm:"column:password_hash"`
	FullName       string    `json:"fullName"      gorm:"column:full_name"`
	UserType       UserType  `json:"userType"      gorm:"column:user_type"`
	OrganizationID *int64    `json:"organizationId,omitempty" gorm:"column:organization_id"`
	OrgRole        *OrgRole  `json:"orgRole,omitempty"        gorm:"column:org_role"`
	Active         bool      `json:"active"        gorm:"column:active"`
	CreatedAt      time.Time `json:"createdAt"     gorm:"column:created_at"`
}

func (User) TableName() string {
	return "users"
}

// Optional: helper untuk cek apakah user adalah org admin
func (u *User) IsOrgAdmin() bool {
	return u.UserType == UserTypeOrgUser && u.OrgRole != nil && *u.OrgRole == OrgRoleAdmin
}

// Optional: helper untuk cek super admin
func (u *User) IsSuperAdmin() bool {
	return u.UserType == UserTypeSuperAdmin
}

// Untuk GORM "hook" misalnya, kita bisa tambahkan method (optional)
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// bisa taruh logic default kalau perlu
	return nil
}
