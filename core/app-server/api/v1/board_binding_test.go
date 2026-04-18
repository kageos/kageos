package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/gin-gonic/gin"
)

func TestUpdatePostReqBindDoesNotRequireBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPut, "/workspace/api/v1/posts/12", strings.NewReader(`{"title":"新的标题"}`))
	request.Header.Set("Content-Type", "application/json")
	ctx.Request = request

	var req dto.UpdatePostReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		t.Fatalf("ShouldBindJSON returned error: %v", err)
	}

	if req.Title != "新的标题" {
		t.Fatalf("req.Title = %q, want %q", req.Title, "新的标题")
	}
	if req.ID != 0 {
		t.Fatalf("req.ID = %d, want 0 before path assignment", req.ID)
	}
}

func TestUpdateBoardReqBindDoesNotRequireBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPut, "/workspace/api/v1/boards/crud/34", strings.NewReader(`{"name":"新版块"}`))
	request.Header.Set("Content-Type", "application/json")
	ctx.Request = request

	var req dto.UpdateBoardReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		t.Fatalf("ShouldBindJSON returned error: %v", err)
	}

	if req.Name != "新版块" {
		t.Fatalf("req.Name = %q, want %q", req.Name, "新版块")
	}
	if req.ID != 0 {
		t.Fatalf("req.ID = %d, want 0 before path assignment", req.ID)
	}
}
