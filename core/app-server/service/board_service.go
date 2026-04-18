package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/gorm"
)

// parseCover 将 DB 存的逗号分隔封面转为 []string；空串返回 nil
func parseCover(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	urls := make([]string, 0, len(parts))
	for _, p := range parts {
		if u := strings.TrimSpace(p); u != "" {
			urls = append(urls, u)
		}
	}
	return urls
}

// formatCover 将 []string 用逗号拼接存 DB
func formatCover(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return strings.Join(urls, ",")
}

// summaryFromContent 从正文截取纯文本摘要，最多 maxRunes 个字符
func summaryFromContent(content string, maxRunes int) string {
	s := content
	// 简单去除 markdown 常见符号：标题 #、列表 - *、代码块 ```、链接 [text](url)、图片 ![]()
	re := regexp.MustCompile(`(?m)^#+\s*|^[-*]\s*|^\d+\.\s*|\[([^\]]*)\]\([^)]*\)|!\[[^\]]*\]\([^)]*\)|` + "`[^`]*`" + `|[*_~]+`)
	s = re.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}

// BoardService 版块/帖子服务
type BoardService struct {
	boardPostRepo   *repository.BoardPostRepository
	serviceTreeRepo *repository.ServiceTreeRepository
}

// NewBoardService 创建版块服务
func NewBoardService(boardPostRepo *repository.BoardPostRepository, serviceTreeRepo *repository.ServiceTreeRepository) *BoardService {
	return &BoardService{
		boardPostRepo:   boardPostRepo,
		serviceTreeRepo: serviceTreeRepo,
	}
}

// GetPostPath 根据帖子 ID 返回所属版块的 full_code_path（供鉴权中间件使用）
func (s *BoardService) GetPostPath(ctx context.Context, id int64) (string, error) {
	post, err := s.boardPostRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("帖子不存在")
		}
		return "", fmt.Errorf("获取帖子失败: %w", err)
	}
	return post.FullCodePath, nil
}

// ListPosts 按版块路径分页列表
func (s *BoardService) ListPosts(ctx context.Context, fullCodePath string, page, pageSize int) (*dto.ListPostsResp, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("版块不存在: %s", fullCodePath)
		}
		return nil, fmt.Errorf("获取版块失败: %w", err)
	}
	if !tree.IsBoard() {
		return nil, fmt.Errorf("节点不是版块类型: %s", tree.Type)
	}
	offset := (page - 1) * pageSize
	list, total, err := s.boardPostRepo.ListByFullCodePath(fullCodePath, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("查询帖子列表失败: %w", err)
	}
	items := make([]dto.PostItem, 0, len(list))
	for _, p := range list {
		summary := strings.TrimSpace(p.Summary)
		if summary == "" && p.Content != "" {
			summary = summaryFromContent(p.Content, 150)
		}
		items = append(items, dto.PostItem{
			ID:        p.ID,
			TreeID:    p.TreeID,
			Title:     p.Title,
			Summary:   summary,
			Cover:     parseCover(p.Cover),
			Author:    p.Author,
			Status:    p.Status,
			CreatedAt: time.Time(p.CreatedAt).Format(time.RFC3339),
			UpdatedAt: time.Time(p.UpdatedAt).Format(time.RFC3339),
		})
	}
	return &dto.ListPostsResp{List: items, Total: total}, nil
}

// GetPost 帖子详情
func (s *BoardService) GetPost(ctx context.Context, id int64) (*dto.GetPostResp, error) {
	post, err := s.boardPostRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("帖子不存在")
		}
		return nil, fmt.Errorf("获取帖子失败: %w", err)
	}
	tree, err := s.serviceTreeRepo.GetByID(post.TreeID)
	if err != nil || tree == nil || !tree.IsBoard() {
		return nil, fmt.Errorf("版块不存在或类型错误")
	}
	summary := strings.TrimSpace(post.Summary)
	if summary == "" && post.Content != "" {
		summary = summaryFromContent(post.Content, 150)
	}
	return &dto.GetPostResp{
		ID:            post.ID,
		TreeID:        post.TreeID,
		FullCodePath:  post.FullCodePath,
		Title:         post.Title,
		Summary:       summary,
		Cover:         parseCover(post.Cover),
		Content:       post.Content,
		ContentFormat: post.ContentFormat,
		Author:        post.Author,
		Status:        post.Status,
		CreatedAt:     time.Time(post.CreatedAt).Format(time.RFC3339),
		UpdatedAt:     time.Time(post.UpdatedAt).Format(time.RFC3339),
	}, nil
}

// CreatePost 发帖
func (s *BoardService) CreatePost(ctx context.Context, req *dto.CreatePostReq) (*dto.GetPostResp, error) {
	author := contextx.GetRequestUser(ctx)
	if author == "" {
		return nil, fmt.Errorf("请先登录")
	}
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("版块不存在: %s", req.FullCodePath)
		}
		return nil, fmt.Errorf("获取版块失败: %w", err)
	}
	if !tree.IsBoard() {
		return nil, fmt.Errorf("节点不是版块类型: %s", tree.Type)
	}
	status := req.Status
	if status == "" {
		status = "published"
	}
	contentFormat := req.ContentFormat
	if contentFormat == "" {
		contentFormat = "markdown"
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" && strings.TrimSpace(req.Content) != "" {
		summary = summaryFromContent(req.Content, 150)
	}
	post := &model.BoardPost{
		TreeID:        tree.ID,
		FullCodePath:  req.FullCodePath,
		Title:         req.Title,
		Summary:       summary,
		Cover:         formatCover(req.Cover),
		Content:       req.Content,
		ContentFormat: contentFormat,
		Author:        author,
		Status:        status,
	}
	if err := s.boardPostRepo.Create(post); err != nil {
		return nil, fmt.Errorf("发帖失败: %w", err)
	}
	return s.GetPost(ctx, post.ID)
}

// UpdatePost 更新帖子
func (s *BoardService) UpdatePost(ctx context.Context, req *dto.UpdatePostReq) (*dto.GetPostResp, error) {
	if req == nil || req.ID <= 0 {
		return nil, fmt.Errorf("帖子ID不能为空")
	}

	post, err := s.boardPostRepo.GetByID(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("帖子不存在")
		}
		return nil, fmt.Errorf("获取帖子失败: %w", err)
	}
	if req.Title != "" {
		post.Title = req.Title
	}
	// 摘要：有传则用，否则若改了正文则从正文重新截取
	if req.Summary != "" {
		post.Summary = strings.TrimSpace(req.Summary)
	} else if req.Content != "" {
		post.Summary = summaryFromContent(req.Content, 150)
	}
	if len(req.Cover) > 0 {
		post.Cover = formatCover(req.Cover)
	}
	if req.Content != "" {
		post.Content = req.Content
	}
	if req.ContentFormat != "" {
		post.ContentFormat = req.ContentFormat
	}
	if req.Status != "" {
		post.Status = req.Status
	}
	if err := s.boardPostRepo.Update(post); err != nil {
		return nil, fmt.Errorf("更新帖子失败: %w", err)
	}
	return s.GetPost(ctx, post.ID)
}

// DeletePost 删除帖子
func (s *BoardService) DeletePost(ctx context.Context, id int64) error {
	_, err := s.boardPostRepo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("帖子不存在")
		}
		return fmt.Errorf("获取帖子失败: %w", err)
	}
	return s.boardPostRepo.Delete(id)
}
