package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT kamu
type UserClaims struct {
	UserID         int64   `json:"userId"`
	UserType       string  `json:"userType"`       // "SUPER_ADMIN" / "ORG_USER"
	OrganizationID *int64  `json:"organizationId"` // boleh nil
	OrgRole        *string `json:"orgRole"`        // "ADMIN" / "USER" (kalau ORG_USER)
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header missing or invalid",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid token",
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*UserClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid token claims",
			})
			c.Abort()
			return
		}

		// Buat CurrentUser dari claims
		cu := CurrentUser{
			ID:       claims.UserID,
			UserType: UserType(claims.UserType),
		}

		if claims.OrganizationID != nil {
			cu.OrganizationID = claims.OrganizationID
		}
		if claims.OrgRole != nil {
			role := OrgRole(*claims.OrgRole)
			cu.OrgRole = &role
		}

		// taruh di context
		c.Set(ContextUserKey, cu)

		c.Next()
	}
}

// Helper untuk ambil current user di handler
func GetCurrentUser(c *gin.Context) (CurrentUser, bool) {
	v, ok := c.Get(ContextUserKey)
	if !ok {
		return CurrentUser{}, false
	}
	cu, ok := v.(CurrentUser)
	return cu, ok
}
