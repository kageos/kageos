package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
)

// StandardAPI 标准接口处理器
// 提供标准化的 RESTful 接口，使用 full-code-path 作为路径参数
type StandardAPI struct {
	appService        *service.AppService
	teamAccessService *service.TeamAccessService
}

type callbackRequestEnvelope struct {
	Method string `json:"method"`
	Router string `json:"router"`
	Body   []byte `json:"body"`
	Type   string `json:"type"`
}

const (
	privateRuntimePythonRouter     = "/_runtime/python"
	internalTableGetRowsCallback   = "__table_get_rows"
	tableGetRowsCallbackHTTPMethod = http.MethodPost
)

type tableGetRowsCallbackReq struct {
	IDs []int64 `json:"ids"`
}

// NewStandardAPI 创建标准接口处理器
func NewStandardAPI(appService *service.AppService, teamAccessService *service.TeamAccessService) *StandardAPI {
	return &StandardAPI{
		appService:        appService,
		teamAccessService: teamAccessService,
	}
}

// parseFullCodePath 从路径参数解析 full-code-path
// 格式：/{user}/{app}/{...}
func parseFullCodePath(fullCodePath string) (user, app string, router string, err error) {
	// 移除开头的斜杠
	fullCodePath = strings.TrimPrefix(fullCodePath, "/")
	parts := strings.Split(fullCodePath, "/")

	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("full-code-path 格式错误，至少需要包含 user/app/function")
	}

	user = parts[0]
	app = parts[1]
	router = strings.Join(parts[2:], "/")

	return user, app, router, nil
}

func parseWorkspaceRootPath(fullCodePath string) (user, app, root string, err error) {
	root = access.NormalizeResourcePath(fullCodePath)
	user, app, err = access.ParseUserApp(root)
	if err != nil {
		return "", "", "", err
	}
	return user, app, access.AppRootPath(user, app), nil
}

func parsePositiveInt64Value(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

// needFillOldValues 判断 table/update 请求体是否缺少 old_values，需要内部自动查表填充
func needFillOldValues(bodyData map[string]interface{}) bool {
	if bodyData == nil {
		return false
	}
	if _, hasID := bodyData["id"]; !hasID {
		return false
	}
	oldValues, ok := bodyData["old_values"].(map[string]interface{})
	return !ok || len(oldValues) == 0
}

// getBodyIDInt64 从 body 中解析 id（支持 float64/int）
func getBodyIDInt64(bodyData map[string]interface{}) (int64, bool) {
	if bodyData == nil {
		return 0, false
	}
	switch v := bodyData["id"].(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	default:
		return 0, false
	}
}

func tableRowID(row map[string]interface{}) (int64, bool) {
	if row == nil {
		return 0, false
	}
	switch v := row["id"].(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		id, err := v.Int64()
		return id, err == nil
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return id, err == nil
	default:
		return 0, false
	}
}

func findTableRowByID(rows []map[string]interface{}, id int64) map[string]interface{} {
	for _, row := range rows {
		rowID, ok := tableRowID(row)
		if ok && rowID == id {
			return row
		}
	}
	if len(rows) == 1 {
		return rows[0]
	}
	return nil
}

func extractTableGetRowsCallbackRows(result interface{}) ([]map[string]interface{}, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("解析 %s 结果失败: %w", internalTableGetRowsCallback, err)
	}
	var payload struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("解析 %s rows 失败: %w", internalTableGetRowsCallback, err)
	}
	if payload.Rows == nil {
		return nil, fmt.Errorf("%s 未返回 rows", internalTableGetRowsCallback)
	}
	return payload.Rows, nil
}

// buildRequestAppReq 构建 RequestAppReq 请求对象
func (s *StandardAPI) buildRequestAppReq(c *gin.Context, fullCodePath string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}

	req := &dto.RequestAppReq{
		User:                  user,
		App:                   app,
		Router:                router,
		Method:                c.Request.Method,
		TraceId:               contextx.GetTraceId(c),
		RequestUser:           contextx.GetRequestUser(c),
		RequestUserDept:       contextx.GetRequestDepartmentFullPath(c),
		Token:                 contextx.GetToken(c),
		ClientSource:          contextx.GetClientSource(c),
		SourceType:            contextx.GetSourceType(c),
		SourceRef:             contextx.GetSourceRef(c),
		SourcePath:            fullCodePath,
		SourceTitle:           contextx.GetSourceTitle(c),
		SourceParentPath:      contextx.GetSourceParentPath(c),
		SourceParentTitle:     contextx.GetSourceParentTitle(c),
		SourceTemplateType:    contextx.GetSourceTemplateType(c),
		WorkspaceSessionID:    contextx.GetWorkspaceSessionID(c),
		WorkspaceSessionTitle: contextx.GetWorkspaceSessionTitle(c),
		WorkspaceRole:         contextx.GetWorkspaceRole(c),
		InitiatorUser:         contextx.GetInitiatorUser(c),
		WorkspaceMessageID:    parsePositiveInt64Value(contextx.GetWorkspaceMessageID(c)),
		ToolCallID:            contextx.GetToolCallID(c),
		ToolName:              contextx.GetToolName(c),
	}

	// 绑定请求体（POST、PUT、PATCH、DELETE 等方法通常有请求体）
	if c.Request.ContentLength > 0 && (c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodDelete) {
		all, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		defer c.Request.Body.Close()
		req.Body = all
	}

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	return req, nil
}

func (s *StandardAPI) buildRuntimePythonRequestAppReq(c *gin.Context, fullCodePath string) (*dto.RequestAppReq, error) {
	user, app, _, err := parseWorkspaceRootPath(fullCodePath)
	if err != nil {
		return nil, err
	}

	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	defer c.Request.Body.Close()
	if len(bytes.TrimSpace(all)) == 0 {
		return nil, fmt.Errorf("请求体不能为空")
	}

	var runtimeReq dto.RunPythonRuntimeReq
	if err := json.Unmarshal(all, &runtimeReq); err != nil {
		return nil, fmt.Errorf("解析 run_python runtime 请求失败: %w", err)
	}
	if strings.TrimSpace(runtimeReq.PythonCode) == "" {
		return nil, fmt.Errorf("python_code 不能为空")
	}
	normalizedBody, err := json.Marshal(runtimeReq)
	if err != nil {
		return nil, err
	}

	return &dto.RequestAppReq{
		User:            user,
		App:             app,
		Router:          privateRuntimePythonRouter,
		Method:          http.MethodPost,
		TraceId:         contextx.GetTraceId(c),
		RequestUser:     contextx.GetRequestUser(c),
		RequestUserDept: contextx.GetRequestDepartmentFullPath(c),
		Token:           contextx.GetToken(c),
		ClientSource:    contextx.GetClientSource(c),
		SourceType:      contextx.GetSourceType(c),
		SourceRef:       contextx.GetSourceRef(c),
		Body:            normalizedBody,
	}, nil
}

func requireAgentToolRuntimeSource(c *gin.Context) error {
	if contextx.GetSourceType(c) == contextx.SourceTypeAgentTool && contextx.GetClientSource(c) == contextx.ClientSourceAgent {
		return nil
	}
	return fmt.Errorf("runtime python 仅允许 agent tool 内部调用")
}

// buildCallbackAppReq 构建 CallbackApp 请求对象
func (s *StandardAPI) buildCallbackAppReq(c *gin.Context, fullCodePath string, callbackType string) (*dto.RequestAppReq, error) {
	// 读取请求体
	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	defer c.Request.Body.Close()
	return s.buildCallbackAppReqWithBody(c, fullCodePath, callbackType, c.Request.Method, all, c.Request.URL.RawQuery)
}

func (s *StandardAPI) buildCallbackAppReqWithBody(c *gin.Context, fullCodePath, callbackType, method string, body []byte, rawQuery string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}

	req := &dto.RequestAppReq{
		User:                  user,
		App:                   app,
		Router:                "/_callback",
		TargetRouter:          router,
		Method:                method,
		TraceId:               contextx.GetTraceId(c),
		RequestUser:           contextx.GetRequestUser(c),
		RequestUserDept:       contextx.GetRequestDepartmentFullPath(c),
		Token:                 contextx.GetToken(c),
		ClientSource:          contextx.GetClientSource(c),
		SourceType:            contextx.GetSourceType(c),
		SourceRef:             contextx.GetSourceRef(c),
		SourcePath:            fullCodePath,
		SourceTitle:           contextx.GetSourceTitle(c),
		SourceParentPath:      contextx.GetSourceParentPath(c),
		SourceParentTitle:     contextx.GetSourceParentTitle(c),
		SourceTemplateType:    contextx.GetSourceTemplateType(c),
		WorkspaceSessionID:    contextx.GetWorkspaceSessionID(c),
		WorkspaceSessionTitle: contextx.GetWorkspaceSessionTitle(c),
		WorkspaceRole:         contextx.GetWorkspaceRole(c),
		InitiatorUser:         contextx.GetInitiatorUser(c),
		WorkspaceMessageID:    parsePositiveInt64Value(contextx.GetWorkspaceMessageID(c)),
		ToolCallID:            contextx.GetToolCallID(c),
		ToolName:              contextx.GetToolName(c),
	}

	// 构建回调请求体
	envelope := callbackRequestEnvelope{
		Method: method,
		Router: router,
		Body:   body,
		Type:   callbackType,
	}

	// 绑定查询参数
	req.UrlQuery = rawQuery

	// 将回调信息序列化为 JSON
	marshal, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	req.Body = marshal

	return req, nil
}

func (s *StandardAPI) fetchTableRowsByIDs(c *gin.Context, fullCodePath string, ids []int64) ([]map[string]interface{}, error) {
	body, err := json.Marshal(tableGetRowsCallbackReq{IDs: ids})
	if err != nil {
		return nil, fmt.Errorf("构造 %s 请求失败: %w", internalTableGetRowsCallback, err)
	}
	req, err := s.buildCallbackAppReqWithBody(c, fullCodePath, internalTableGetRowsCallback, tableGetRowsCallbackHTTPMethod, body, "")
	if err != nil {
		return nil, err
	}
	resp, err := s.appService.RequestApp(contextx.ToContext(c), req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%s 返回空响应", internalTableGetRowsCallback)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s 失败: %s", internalTableGetRowsCallback, resp.Error)
	}
	return extractTableGetRowsCallbackRows(resp.Result)
}

func (s *StandardAPI) ensureTableCallbackEnabled(c *gin.Context, fullCodePath, callbackType, denyMessage string) error {
	function, err := s.appService.GetFunctionByFullCodePath(contextx.ToContext(c), fullCodePath)
	if err != nil {
		return err
	}
	if functionschema.Type(function.Schema) != functionschema.TypeTable {
		return fmt.Errorf("目标函数不是 Table 类型，不支持该操作")
	}
	if !function.HasCallback(callbackType) {
		return fmt.Errorf("%s", denyMessage)
	}
	return nil
}
