package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
	"gorm.io/gorm"
)

type PublicShareService struct {
	publicShareRepo *repository.PublicShareRepository
	functionRepo    *repository.FunctionRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	operateLogRepo  *repository.OperateLogRepository
}

const maxPublicSharePresetValuesBytes = 64 * 1024
const maxPublicShareSubmissionPageSize = 100

func NewPublicShareService(
	publicShareRepo *repository.PublicShareRepository,
	functionRepo *repository.FunctionRepository,
	serviceTreeRepo *repository.ServiceTreeRepository,
	operateLogRepo *repository.OperateLogRepository,
) *PublicShareService {
	return &PublicShareService{
		publicShareRepo: publicShareRepo,
		functionRepo:    functionRepo,
		serviceTreeRepo: serviceTreeRepo,
		operateLogRepo:  operateLogRepo,
	}
}

func (s *PublicShareService) Create(ctx context.Context, req *dto.CreatePublicShareReq, actor string) (*dto.PublicShareResp, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	fullCodePath := access.NormalizeResourcePath(req.FullCodePath)
	if fullCodePath == "" {
		return nil, fmt.Errorf("full_code_path 不能为空")
	}
	tenantUser, app, _, err := parsePublicFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}
	function, err := s.functionRepo.GetFunctionByFullCodePath(fullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}
	if functionschema.Type(function.Schema) != functionschema.TypeForm {
		return nil, fmt.Errorf("公开匿名提交 MVP 仅支持 form 节点")
	}
	presetValues, err := normalizePublicSharePresetValues(req.PresetValues)
	if err != nil {
		return nil, err
	}

	shareID, err := newShareID()
	if err != nil {
		return nil, err
	}
	share := &model.PublicShare{
		ShareID:      shareID,
		TenantUser:   tenantUser,
		App:          app,
		FullCodePath: fullCodePath,
		ResourceType: model.PublicShareResourceTypeForm,
		Action:       model.PublicShareActionFormSubmit,
		Title:        strings.TrimSpace(req.Title),
		Description:  strings.TrimSpace(req.Description),
		Enabled:      true,
		ExpiresAt:    req.ExpiresAt,
		MaxUses:      max(req.MaxUses, 0),
		PresetValues: presetValues,
	}
	share.CreatedBy = actor
	share.UpdatedBy = actor
	if err := s.publicShareRepo.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("创建公开分享失败: %w", err)
	}
	return publicShareToResp(share), nil
}

func (s *PublicShareService) List(ctx context.Context, tenantUser, app string, filter repository.PublicShareListFilter) (*dto.PublicShareListResp, error) {
	filter.FullCodePath = access.NormalizeResourcePath(filter.FullCodePath)
	shares, err := s.publicShareRepo.List(ctx, tenantUser, app, filter)
	if err != nil {
		return nil, fmt.Errorf("查询公开分享失败: %w", err)
	}
	resp := &dto.PublicShareListResp{Items: make([]*dto.PublicShareResp, 0, len(shares))}
	for _, share := range shares {
		resp.Items = append(resp.Items, publicShareToResp(share))
	}
	return resp, nil
}

func (s *PublicShareService) Disable(ctx context.Context, shareID, actor string) error {
	shareID = strings.TrimSpace(shareID)
	if shareID == "" {
		return fmt.Errorf("share_id 不能为空")
	}
	if err := s.publicShareRepo.Disable(ctx, shareID, actor); err != nil {
		return fmt.Errorf("禁用公开分享失败: %w", err)
	}
	return nil
}

func (s *PublicShareService) GetShare(ctx context.Context, shareID string) (*model.PublicShare, error) {
	share, err := s.publicShareRepo.GetByShareID(ctx, strings.TrimSpace(shareID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("公开分享不存在")
		}
		return nil, fmt.Errorf("获取公开分享失败: %w", err)
	}
	return share, nil
}

func (s *PublicShareService) GetActiveShare(ctx context.Context, shareID string) (*model.PublicShare, error) {
	share, err := s.GetShare(ctx, shareID)
	if err != nil {
		return nil, err
	}
	if err := validatePublicShareUsable(share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *PublicShareService) BuildView(ctx context.Context, share *model.PublicShare) (*dto.PublicShareViewResp, error) {
	function, err := s.functionRepo.GetFunctionByFullCodePath(share.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取表单信息失败: %w", err)
	}
	schema, err := functionschema.Parse(function.Schema)
	if err != nil {
		return nil, fmt.Errorf("表单 schema 无效: %w", err)
	}
	resp := &dto.PublicShareViewResp{
		ShareID:      share.ShareID,
		Title:        fallbackPublicShareTitle(share.Title, function),
		Description:  share.Description,
		FullCodePath: share.FullCodePath,
		Schema:       schema,
		ExpiresAt:    share.ExpiresAt,
		PresetValues: share.PresetValues,
	}
	if share.MaxUses > 0 {
		remaining := share.MaxUses - share.UseCount
		if remaining < 0 {
			remaining = 0
		}
		resp.RemainingUses = &remaining
	}
	return resp, nil
}

// ListSubmissions 返回当前匿名访客在当前公开表单上的提交日志。
// 查询范围完全由服务端固定，客户端不能传 actor、路由或来源条件。
func (s *PublicShareService) ListSubmissions(
	ctx context.Context,
	share *model.PublicShare,
	actorID string,
	page int,
	pageSize int,
) (*dto.PublicShareSubmissionListResp, error) {
	if share == nil {
		return nil, fmt.Errorf("公开分享不存在")
	}
	if s.operateLogRepo == nil {
		return nil, fmt.Errorf("操作日志服务不可用")
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("匿名访客标识不能为空")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > maxPublicShareSubmissionPageSize {
		pageSize = maxPublicShareSubmissionPageSize
	}

	resourcePath := "/" + strings.TrimPrefix(access.NormalizeResourcePath(share.FullCodePath), "/")
	logs, total, err := s.operateLogRepo.GetOperateLogs(ctx, &dto.GetOperateLogsReq{
		TenantUser:   share.TenantUser,
		ActorUser:    actorID,
		App:          share.App,
		ResourceType: "form",
		ResourcePath: resourcePath,
		Action:       "form_submit",
		SourceType:   contextx.SourceTypePublicShare,
		SourceRef:    share.ShareID,
		Page:         page,
		PageSize:     pageSize,
		OrderBy:      "created_at desc",
	})
	if err != nil {
		return nil, fmt.Errorf("查询公开提交记录失败: %w", err)
	}

	items := make([]*dto.PublicShareSubmissionItem, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		var details dto.FormOperateLogDetails
		if len(log.DetailsJSON) > 0 {
			_ = json.Unmarshal(log.DetailsJSON, &details)
		}
		items = append(items, &dto.PublicShareSubmissionItem{
			Status:         log.Status,
			Summary:        log.Summary,
			RequestBody:    details.RequestBody,
			ResponseBody:   details.ResponseBody,
			DurationMillis: details.DurationMillis,
			TraceID:        log.TraceID,
			CreatedAt:      time.Time(log.CreatedAt).Format("2006-01-02 15:04:05"),
		})
	}

	return &dto.PublicShareSubmissionListResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func fallbackPublicShareTitle(title string, function *model.Function) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	if function == nil {
		return "公开表单"
	}
	if last := strings.TrimSpace(function.GetLastRouterSegment()); last != "" {
		return last
	}
	return "公开表单"
}

func (s *PublicShareService) ReserveUse(ctx context.Context, shareID string) error {
	if err := s.publicShareRepo.ReserveUse(ctx, shareID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("公开分享已达到提交上限或已关闭")
		}
		return fmt.Errorf("预留公开分享提交次数失败: %w", err)
	}
	return nil
}

func (s *PublicShareService) ReleaseUse(ctx context.Context, shareID string) {
	_ = s.publicShareRepo.ReleaseUse(ctx, shareID)
}

func (s *PublicShareService) RecordEvent(ctx context.Context, event *model.PublicShareEvent) {
	if event == nil {
		return
	}
	_ = s.publicShareRepo.CreateEvent(ctx, event)
}

func validatePublicShareUsable(share *model.PublicShare) error {
	if share == nil {
		return fmt.Errorf("公开分享不存在")
	}
	if !share.Enabled {
		return fmt.Errorf("公开分享已关闭")
	}
	if share.ResourceType != model.PublicShareResourceTypeForm || share.Action != model.PublicShareActionFormSubmit {
		return fmt.Errorf("公开分享类型暂不支持")
	}
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return fmt.Errorf("公开分享已过期")
	}
	if share.MaxUses > 0 && share.UseCount >= share.MaxUses {
		return fmt.Errorf("公开分享提交次数已达上限")
	}
	return nil
}

func publicShareToResp(share *model.PublicShare) *dto.PublicShareResp {
	if share == nil {
		return nil
	}
	return &dto.PublicShareResp{
		ShareID:      share.ShareID,
		TenantUser:   share.TenantUser,
		App:          share.App,
		FullCodePath: share.FullCodePath,
		ResourceType: share.ResourceType,
		Action:       share.Action,
		Title:        share.Title,
		Description:  share.Description,
		Enabled:      share.Enabled,
		ExpiresAt:    share.ExpiresAt,
		MaxUses:      share.MaxUses,
		UseCount:     share.UseCount,
		LastUsedAt:   share.LastUsedAt,
		CreatedAt:    time.Time(share.CreatedAt),
		CreatedBy:    share.CreatedBy,
		PresetValues: share.PresetValues,
	}
}

func normalizePublicSharePresetValues(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return nil, nil
	}
	if len(trimmed) > maxPublicSharePresetValuesBytes {
		return nil, fmt.Errorf("preset_values 过大，最多允许 %d 字节", maxPublicSharePresetValuesBytes)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &values); err != nil || values == nil {
		return nil, fmt.Errorf("preset_values 必须是 JSON 对象")
	}
	normalized := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = json.RawMessage(strings.TrimSpace(string(value)))
		if len(value) == 0 {
			value = json.RawMessage("null")
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	out, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("preset_values 序列化失败: %w", err)
	}
	return out, nil
}

func parsePublicFullCodePath(fullCodePath string) (user, app, router string, err error) {
	fullCodePath = strings.TrimPrefix(fullCodePath, "/")
	parts := strings.Split(fullCodePath, "/")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("full_code_path 格式错误，至少需要包含 user/app/function")
	}
	return parts[0], parts[1], strings.Join(parts[2:], "/"), nil
}

func newShareID() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "ps_" + base64.RawURLEncoding.EncodeToString(buf), nil
}
