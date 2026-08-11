package serverx

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewGinLimitsRequestBody(t *testing.T) {
	engine := NewGin(WithMode(gin.TestMode))
	engine.POST("/body", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		var maxBytesErr *http.MaxBytesError
		if !errors.As(err, &maxBytesErr) {
			c.Status(http.StatusOK)
			return
		}
		c.Status(http.StatusRequestEntityTooLarge)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/body",
		strings.NewReader(strings.Repeat("x", int(DefaultMaxRequestBodyBytes)+1)),
	)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("response status = %d, want %d", resp.Code, http.StatusRequestEntityTooLarge)
	}
}
