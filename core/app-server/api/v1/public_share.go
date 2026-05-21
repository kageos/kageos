package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/publicshare"
)

type PublicShareAPI struct {
	publicShareService *service.PublicShareService
	appService         *service.AppService
	teamAccessService  *service.TeamAccessService
}

func NewPublicShareAPI(
	publicShareService *service.PublicShareService,
	appService *service.AppService,
	teamAccessService *service.TeamAccessService,
) *PublicShareAPI {
	return &PublicShareAPI{
		publicShareService: publicShareService,
		appService:         appService,
		teamAccessService:  teamAccessService,
	}
}

func (a *PublicShareAPI) Create(c *gin.Context) {
	var req dto.CreatePublicShareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	req.FullCodePath = access.NormalizeResourcePath(req.FullCodePath)
	if err := requireAccess(c, a.teamAccessService, req.FullCodePath, access.ActionWrite); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	resp, err := a.publicShareService.Create(contextx.ToContext(c), &req, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	resp.PublicURL = publicShareURL(c, resp.ShareID)
	response.OkWithData(c, resp)
}

func (a *PublicShareAPI) List(c *gin.Context) {
	fullCodePath := access.NormalizeResourcePath(c.Query("full_code_path"))
	if fullCodePath == "" {
		response.FailWithMessage(c, "full_code_path 参数不能为空")
		return
	}
	if err := requireAccess(c, a.teamAccessService, fullCodePath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	tenantUser, app, _, err := parseFullCodePath(fullCodePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	resp, err := a.publicShareService.List(contextx.ToContext(c), tenantUser, app, repository.PublicShareListFilter{
		FullCodePath: fullCodePath,
		Keyword:      strings.TrimSpace(c.Query("keyword")),
		CreatedBy:    strings.TrimSpace(c.Query("created_by")),
		Status:       strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	for _, item := range resp.Items {
		item.PublicURL = publicShareURL(c, item.ShareID)
	}
	response.OkWithData(c, resp)
}

func (a *PublicShareAPI) Disable(c *gin.Context) {
	shareID := strings.TrimSpace(c.Param("share_id"))
	share, err := a.publicShareService.GetShare(contextx.ToContext(c), shareID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := requireAccess(c, a.teamAccessService, share.FullCodePath, access.ActionWrite); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := a.publicShareService.Disable(contextx.ToContext(c), shareID, contextx.GetRequestUser(c)); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

func (a *PublicShareAPI) AnonymousToken(c *gin.Context) {
	token, claims, err := publicshare.ValidateOrIssueAnonymousToken(c.GetHeader(publicshare.AnonymousTokenHeader))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	c.Header(publicshare.AnonymousTokenHeader, token)
	response.OkWithData(c, &dto.PublicAnonymousTokenResp{
		AnonymousToken: token,
		ExpiresAt:      time.Unix(claims.ExpiresAt, 0),
	})
}

func (a *PublicShareAPI) View(c *gin.Context) {
	shareID := strings.TrimSpace(c.Param("share_id"))
	if _, err := publicshare.ValidateAnonymousToken(c.GetHeader(publicshare.AnonymousTokenHeader)); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	share, err := a.publicShareService.GetActiveShare(contextx.ToContext(c), shareID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	resp, err := a.publicShareService.BuildView(contextx.ToContext(c), share)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

func (a *PublicShareAPI) Submit(c *gin.Context) {
	shareID := strings.TrimSpace(c.Param("share_id"))
	token := c.GetHeader(publicshare.AnonymousTokenHeader)
	claims, err := publicshare.ValidateAnonymousToken(token)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	share, err := a.publicShareService.GetActiveShare(contextx.ToContext(c), shareID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	actorID := publicshare.DeriveActorID(share.TenantUser, share.App, share.ShareID, claims.SessionID)
	req, err := a.buildPublicRequestAppReq(c, share, actorID, token)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, callErr := a.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	eventStatus := "success"
	eventError := ""
	if callErr != nil {
		eventStatus = "failed"
		eventError = callErr.Error()
	} else if resp != nil && resp.Error != "" {
		eventStatus = "failed"
		eventError = resp.Error
	}

	a.recordPublicSubmitLog(ctx, c, req, resp, callErr, mill)
	a.publicShareService.RecordEvent(context.Background(), &model.PublicShareEvent{
		ShareID:       share.ShareID,
		TenantUser:    share.TenantUser,
		App:           share.App,
		FullCodePath:  share.FullCodePath,
		AnonActorID:   actorID,
		Action:        model.PublicShareActionFormSubmit,
		Status:        eventStatus,
		TraceID:       req.TraceId,
		ErrorMessage:  eventError,
		IPAddressHash: publicshare.HashValue(c.ClientIP()),
		UserAgentHash: publicshare.HashValue(c.GetHeader("User-Agent")),
	})

	metadata := map[string]interface{}{
		"trace_id":        req.TraceId,
		"app":             req.App,
		"total_cost_mill": mill,
		"anonymous_user":  actorID,
	}
	if resp != nil {
		metadata["version"] = resp.Version
	}
	if callErr != nil {
		response.FailWithMessage(c, callErr.Error(), metadata)
		return
	}
	if resp == nil {
		response.FailWithMessage(c, "公开表单提交失败: 空响应", metadata)
		return
	}
	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}
	if err := a.publicShareService.IncrementUseCount(ctx, share.ShareID); err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}
	a.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(share.FullCodePath, "/"))
	c.Header(publicshare.AnonymousTokenHeader, token)
	response.OkWithData(c, resp.Result, metadata)
}

func (a *PublicShareAPI) CallbackOnSelectFuzzy(c *gin.Context) {
	shareID := strings.TrimSpace(c.Param("share_id"))
	token := c.GetHeader(publicshare.AnonymousTokenHeader)
	claims, err := publicshare.ValidateAnonymousToken(token)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	share, err := a.publicShareService.GetActiveShare(contextx.ToContext(c), shareID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	actorID := publicshare.DeriveActorID(share.TenantUser, share.App, share.ShareID, claims.SessionID)
	req, err := a.buildPublicCallbackAppReq(c, share, actorID, token, "OnSelectFuzzy")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := a.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	metadata := map[string]interface{}{
		"trace_id":        req.TraceId,
		"app":             req.App,
		"total_cost_mill": mill,
		"anonymous_user":  actorID,
	}
	if resp != nil {
		metadata["version"] = resp.Version
	}
	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}
	if resp == nil {
		response.FailWithMessage(c, "公开表单回调失败: 空响应", metadata)
		return
	}
	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}
	response.OkWithData(c, resp.Result, metadata)
}

func (a *PublicShareAPI) buildPublicRequestAppReq(c *gin.Context, share *model.PublicShare, actorID, token string) (*dto.RequestAppReq, error) {
	tenantUser, app, router, err := parseFullCodePath(share.FullCodePath)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %w", err)
	}
	defer c.Request.Body.Close()
	traceID := contextx.GetTraceId(c)
	if traceID == "" {
		traceID = fmt.Sprintf("public_%s_%d", share.ShareID, time.Now().UnixNano())
	}
	c.Request.Header.Set(contextx.TraceIdHeader, traceID)
	c.Request.Header.Set(contextx.RequestUserHeader, actorID)
	c.Request.Header.Set(contextx.ClientSourceHeader, "public_share")
	c.Request.Header.Set(contextx.SourceTypeHeader, "public_share")
	c.Request.Header.Set(contextx.SourceRefHeader, share.ShareID)
	c.Request.Header.Set(publicshare.AnonymousTokenHeader, token)

	return &dto.RequestAppReq{
		User:           tenantUser,
		App:            app,
		Router:         router,
		Method:         http.MethodPost,
		TraceId:        traceID,
		RequestUser:    actorID,
		AnonymousToken: token,
		ClientSource:   "public_share",
		SourceType:     "public_share",
		SourceRef:      share.ShareID,
		Body:           body,
		UrlQuery:       c.Request.URL.RawQuery,
	}, nil
}

func (a *PublicShareAPI) buildPublicCallbackAppReq(c *gin.Context, share *model.PublicShare, actorID, token, callbackType string) (*dto.RequestAppReq, error) {
	tenantUser, app, router, err := parseFullCodePath(share.FullCodePath)
	if err != nil {
		return nil, err
	}
	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	defer c.Request.Body.Close()

	traceID := contextx.GetTraceId(c)
	if traceID == "" {
		traceID = fmt.Sprintf("public_%s_%d", share.ShareID, time.Now().UnixNano())
	}
	c.Request.Header.Set(contextx.TraceIdHeader, traceID)
	c.Request.Header.Set(contextx.RequestUserHeader, actorID)
	c.Request.Header.Set(contextx.ClientSourceHeader, "public_share")
	c.Request.Header.Set(contextx.SourceTypeHeader, "public_share")
	c.Request.Header.Set(contextx.SourceRefHeader, share.ShareID)
	c.Request.Header.Set(publicshare.AnonymousTokenHeader, token)

	envelope := callbackRequestEnvelope{
		Method: http.MethodPost,
		Router: router,
		Body:   all,
		Type:   callbackType,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}

	return &dto.RequestAppReq{
		User:           tenantUser,
		App:            app,
		Router:         "/_callback",
		Method:         http.MethodPost,
		TraceId:        traceID,
		RequestUser:    actorID,
		AnonymousToken: token,
		ClientSource:   "public_share",
		SourceType:     "public_share",
		SourceRef:      share.ShareID,
		Body:           body,
		UrlQuery:       c.Request.URL.RawQuery,
	}, nil
}

func (a *PublicShareAPI) recordPublicSubmitLog(ctx context.Context, c *gin.Context, req *dto.RequestAppReq, resp *dto.RequestAppResp, err error, durationMillis int64) {
	formLogReq := &dto.RecordFormOperateLogReq{
		TenantUser:     req.User,
		RequestUser:    req.RequestUser,
		App:            req.App,
		Router:         req.Router,
		Action:         "public_form_submit",
		FunctionMethod: req.Method,
		RequestBody:    req.Body,
		ResponseBody:   buildFormOperateLogResponseBody(resp, err, durationMillis),
		IPAddress:      c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
		TraceID:        req.TraceId,
		DurationMillis: durationMillis,
		Status:         "success",
		Summary:        "公开表单提交成功",
	}
	if resp != nil {
		formLogReq.Version = resp.Version
	}
	if err != nil || (resp != nil && resp.Error != "") {
		formLogReq.Status = "failed"
		formLogReq.Summary = "公开表单提交失败"
	}
	if logErr := a.appService.RecordFormOperateLog(ctx, formLogReq); logErr != nil {
		logger.Warnf(ctx, "[PublicShareSubmit] 记录公开 Form 操作日志失败: %v", logErr)
	}
}

func publicShareURL(c *gin.Context, shareID string) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/public/s/%s", scheme, host, shareID)
}
