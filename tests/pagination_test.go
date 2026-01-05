package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/username/fms-api/internal/pagination"
)

func TestParsePagination_Defaults(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?", nil)
	c.Request = req

	p := pagination.ParsePagination(c)
	if p.Limit != 10 {
		t.Fatalf("expected default limit 10, got %d", p.Limit)
	}
	if p.Page != 1 {
		t.Fatalf("expected default page 1, got %d", p.Page)
	}
}

func TestParsePagination_InvalidParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?limit=abc&page=-1", nil)
	c.Request = req

	p := pagination.ParsePagination(c)
	if !c.IsAborted() {
		// Parser aborts on invalid params
		t.Fatalf("expected context to be aborted for invalid params, pagination=%+v", p)
	}
}
