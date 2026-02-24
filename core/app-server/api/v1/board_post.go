package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

// BoardPost 版块帖子 API
type BoardPost struct {
	boardService *service.BoardService
}

// NewBoardPost 创建版块帖子处理器
func NewBoardPost(boardService *service.BoardService) *BoardPost {
	return &BoardPost{boardService: boardService}
}

// ListPosts 帖子列表（按版块路径分页）
// @Summary 帖子列表
// @Description 按版块 full_code_path 分页获取帖子列表
// @Tags 版块帖子
// @Param full_code_path query string true "版块完整路径"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} dto.ListPostsResp
// @Router /api/v1/posts [get]
func (s *BoardPost) ListPosts(c *gin.Context) {
	var req dto.ListPostsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	ctx := contextx.ToContext(c)
	resp, err := s.boardService.ListPosts(ctx, req.FullCodePath, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, "获取帖子列表失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// GetPost 帖子详情
// @Summary 帖子详情
// @Param id path int true "帖子ID"
// @Success 200 {object} dto.GetPostResp
// @Router /api/v1/posts/{id} [get]
func (s *BoardPost) GetPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.boardService.GetPost(ctx, id)
	if err != nil {
		response.FailWithMessage(c, "获取帖子失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// CreatePost 发帖
// @Summary 发帖
// @Param request body dto.CreatePostReq true "发帖请求"
// @Success 200 {object} dto.GetPostResp
// @Router /api/v1/posts [post]
func (s *BoardPost) CreatePost(c *gin.Context) {
	var req dto.CreatePostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.boardService.CreatePost(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "发帖失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// UpdatePost 更新帖子
// @Summary 更新帖子
// @Param id path int true "帖子ID"
// @Param request body dto.UpdatePostReq true "更新请求"
// @Success 200 {object} dto.GetPostResp
// @Router /api/v1/posts/{id} [put]
func (s *BoardPost) UpdatePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}
	var req dto.UpdatePostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id
	ctx := contextx.ToContext(c)
	resp, err := s.boardService.UpdatePost(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "更新帖子失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// DeletePost 删除帖子
// @Summary 删除帖子
// @Param id path int true "帖子ID"
// @Router /api/v1/posts/{id} [delete]
func (s *BoardPost) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}
	ctx := contextx.ToContext(c)
	if err := s.boardService.DeletePost(ctx, id); err != nil {
		response.FailWithMessage(c, "删除帖子失败: "+err.Error())
		return
	}
	response.OkWithMessage(c, "删除成功")
}
