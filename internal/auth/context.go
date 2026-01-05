package auth

type UserType string
type OrgRole string

const (
	UserTypeSuperAdmin UserType = "SUPER_ADMIN"
	UserTypeOrgUser    UserType = "ORG_USER"

	OrgRoleAdmin OrgRole = "ADMIN"
	OrgRoleUser  OrgRole = "USER"
)

// CurrentUser = info singkat user yang lagi login
type CurrentUser struct {
	ID             int64
	UserType       UserType
	OrganizationID *int64 // bisa nil kalau SUPER_ADMIN
	OrgRole        *OrgRole
}

const ContextUserKey = "currentUser"

func (cu CurrentUser) IsSuperAdmin() bool {
	return cu.UserType == UserTypeSuperAdmin
}

func (cu CurrentUser) IsOrgAdmin() bool {
	return cu.UserType == UserTypeOrgUser && cu.OrgRole != nil && *cu.OrgRole == OrgRoleAdmin
}
