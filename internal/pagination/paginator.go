package pagination

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Pagination holds pagination parameters and metadata
type Pagination struct {
	Limit    int   `json:"limit"`
	Offset   int   `json:"offset"`
	Page     int   `json:"page"`
	MaxLimit int   `json:"maxLimit"`
	Total    int64 `json:"total,omitempty"`
}

// ParsePagination reads query params `limit` and `page` and enforces max limit from env `MAX_LIMIT`.
// Defaults: limit=10, maxLimit=1000 (if env absent)
func ParsePagination(c *gin.Context) Pagination {
	// defaults
	defaultLimit := 10
	maxLimit := 1000

	if ml := os.Getenv("MAX_LIMIT"); ml != "" {
		if v, err := strconv.Atoi(ml); err == nil && v > 0 {
			maxLimit = v
		}
	}

	// read limit
	limit := defaultLimit
	if ls := c.Query("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 {
			limit = v
		} else if ls != "" {
			// invalid param, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid limit parameter"})
			c.Abort()
			return Pagination{}
		}
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	// read page
	page := 1
	if ps := c.Query("page"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			page = v
		} else if ps != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "invalid page parameter"})
			c.Abort()
			return Pagination{}
		}
	}

	offset := (page - 1) * limit

	return Pagination{Limit: limit, Offset: offset, Page: page, MaxLimit: maxLimit}
}
