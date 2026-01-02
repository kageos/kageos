package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// StandardAPI 标准接口处理器
// 提供标准化的 RESTful 接口，使用 full-code-path 作为路径参数
type StandardAPI struct {
	appService *service.AppService
}

// NewStandardAPI 创建标准接口处理器
func NewStandardAPI(appService *service.AppService) *StandardAPI {
	return &StandardAPI{
		appService: appService,
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

// buildRequestAppReq 构建 RequestAppReq 请求对象
func (s *StandardAPI) buildRequestAppReq(c *gin.Context, fullCodePath string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}

	req := &dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      router,
		Method:      c.Request.Method,
		TraceId:     contextx.GetTraceId(c),
		RequestUser: contextx.GetRequestUser(c),
		Token:       contextx.GetToken(c),
	}

	// 绑定请求体（POST、PUT、PATCH、DELETE 等方法通常有请求体）
	if c.Request.ContentLength > 0 && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" || c.Request.Method == "DELETE") {
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

// buildCallbackAppReq 构建 CallbackApp 请求对象
func (s *StandardAPI) buildCallbackAppReq(c *gin.Context, fullCodePath string, callbackType string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}

	req := &dto.RequestAppReq{
		User:        user,
		App:         app,
		Router:      "/_callback",
		Method:      c.Request.Method,
		TraceId:     contextx.GetTraceId(c),
		RequestUser: contextx.GetRequestUser(c),
		Token:       contextx.GetToken(c),
	}

	// 读取请求体
	all, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	defer c.Request.Body.Close()

	// 构建回调请求体
	mp := make(map[string]interface{})
	mp["method"] = c.Request.Method
	mp["router"] = router
	mp["body"] = all
	mp["type"] = callbackType

	// 绑定查询参数
	req.UrlQuery = c.Request.URL.RawQuery

	// 将回调信息序列化为 JSON
	marshal, err := json.Marshal(mp)
	if err != nil {
		return nil, err
	}
	req.Body = marshal

	return req, nil
}

// ============================================
// Table 函数接口
// ============================================

// TableSearch Table 查询接口
// @Summary Table 查询
// @Description 查询表格数据（列表），支持分页、排序、搜索
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param page query int false "页码（可选，默认 1）"
// @Param page_size query int false "每页数量（可选，默认 20）"
// @Param sorts query string false "排序（可选，格式：id:desc,name:asc）"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/search/{full-code-path} [get]
func (s *StandardAPI) TableSearch(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "解析路径参数失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableCreate Table 新增接口
// @Summary Table 新增
// @Description 新增表格记录
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "新增记录的字段数据"
// @Success 200 {object} dto.RequestAppResp "新增成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/create/{full-code-path} [post]
func (s *StandardAPI) TableCreate(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建回调请求对象（调用 OnTableAddRow）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableAddRow")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableBatchCreate Table 批量导入接口
// @Summary Table 批量导入
// @Description 批量导入表格记录（直接批量插入数据库，不触发 OnTableAddRow 回调）
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "批量导入数据，格式：{\"data\": [{\"field1\": \"value1\"}, ...]}"
// @Success 200 {object} dto.RequestAppResp "导入成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/batch-create/{full-code-path} [post]
func (s *StandardAPI) TableBatchCreate(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建回调请求对象（调用 OnTableCreateInBatches）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableCreateInBatches")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableTemplate Table 下载导入模板接口
// @Summary Table 下载导入模板
// @Description 根据函数详情生成 Excel 导入模板
// @Tags 标准接口
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Success 200 {file} file "Excel 模板文件"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/template/{full-code-path} [get]
func (s *StandardAPI) TableTemplate(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	ctx := contextx.ToContext(c)

	// 获取函数信息（直接使用 full-code-path）
	function, err := s.appService.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "获取函数信息失败: "+err.Error())
		return
	}

	// 解析 response 字段为 widget.Field（已经解析好的结构）
	var responseFields []*widget.Field
	if len(function.Response) > 0 {
		if err := json.Unmarshal(function.Response, &responseFields); err != nil {
			response.FailWithMessage(c, "解析函数配置失败: "+err.Error())
			return
		}
	}

	// 获取当前用户信息（用于创建用户字段的默认值）
	username := contextx.GetRequestUser(c)
	
	// 过滤可编辑字段（table_permission 为空或 update，排除 ID 字段）
	// 同时包含系统字段（created_at, create_by 等，即使 permission="read" 也导出）
	editableFields := make([]*widget.Field, 0)
	for _, field := range responseFields {
		// 检查是否是 ID 字段
		if field.Widget.Type == widget.TypeID {
			continue // 跳过 ID 字段
		}

		// 检查是否是系统字段（created_at, create_by 等）
		isSystemField := false
		if field.Code == "created_at" || field.Code == "create_by" || 
		   field.Code == "updated_at" || field.Code == "updated_by" {
			isSystemField = true
		}

		// 包含可编辑字段或系统字段
		if isSystemField || field.TablePermission == "" || field.TablePermission == "update" {
			editableFields = append(editableFields, field)
		}
	}

	if len(editableFields) == 0 {
		response.FailWithMessage(c, "没有可编辑的字段")
		return
	}

	// 生成 Excel 模板
	excelFile := excelize.NewFile()
	defer excelFile.Close()

	sheetName := "Sheet1"
	excelFile.DeleteSheet("Sheet1")
	excelFile.NewSheet(sheetName)

	// 第一行：字段名称（中文）
	// 第二行开始：示例数据行
	// 对于 select/multiselect：每个选项作为一行
	// 对于 bool/switch：显示"是"和"否"两行
	// 其他类型：使用默认值

	// 设置第一行（字段名称）
	for i, field := range editableFields {
		fieldName := field.Name
		if fieldName == "" {
			fieldName = field.FieldName
		}
		cellName, _ := excelize.CoordinatesToCellName(i+1, 1)
		excelFile.SetCellValue(sheetName, cellName, fieldName)
	}

	// 生成示例数据行（传入当前用户和时间，用于系统字段的默认值）
	exampleRows := generateExampleRows(editableFields, username)
	
	// 写入示例数据行（从第二行开始）
	for rowIndex, row := range exampleRows {
		for colIndex, value := range row {
			if value != nil {
				cellName, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
				excelFile.SetCellValue(sheetName, cellName, value)
			}
		}
	}

	// 设置列宽
	for i := range editableFields {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		excelFile.SetColWidth(sheetName, colName, colName, 20)
	}

	// 设置响应头
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	
	// 从 full-code-path 中提取函数名（最后一段）
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	funcName := "template"
	if len(pathParts) > 0 {
		funcName = pathParts[len(pathParts)-1]
	}
	
	// 文件名编码处理（支持中文文件名）
	fileName := fmt.Sprintf("%s_导入模板.xlsx", funcName)
	// 使用 RFC 5987 格式支持中文文件名（兼容性更好）
	encodedFileName := url.QueryEscape(fileName)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fileName, encodedFileName))
	
	// 设置状态码
	c.Status(200)
	
	// 写入响应（直接写入，不使用 JSON 包装器）
	if err := excelFile.Write(c.Writer); err != nil {
		logger.Errorf(ctx, "[TableTemplate] 生成 Excel 模板失败: %v", err)
		// 如果已经写入部分数据，不能再用 response.FailWithMessage
		// 这里只能记录错误，无法返回错误响应
		return
	}
}

// generateExampleRows 根据字段类型生成示例数据行
// 返回：多行示例数据，每行是一个字段值的数组
// username: 当前用户名，用于创建用户字段的默认值
func generateExampleRows(fields []*widget.Field, username string) [][]interface{} {
	// 找出需要生成多行的字段（select/multiselect 和 bool/switch）
	var maxRows int = 1 // 最大行数
	
	for _, field := range fields {
		rowCount := 1
		switch field.Widget.Type {
		case widget.TypeSelect, widget.TypeMultiSelect:
			// 从 widget config 中获取选项
			if configMap, ok := field.Widget.Config.(map[string]interface{}); ok {
				// Options 是 []string 类型
				if options, ok := configMap["options"].([]interface{}); ok && len(options) > 0 {
					rowCount = len(options)
				} else if optionsStr, ok := configMap["options"].([]string); ok && len(optionsStr) > 0 {
					rowCount = len(optionsStr)
				}
			}
		case widget.TypeSwitch:
			// bool 类型：显示"是"和"否"两行
			rowCount = 2
		}
		
		if rowCount > maxRows {
			maxRows = rowCount
		}
	}
	
	// 生成示例数据行
	rows := make([][]interface{}, maxRows)
	for rowIndex := 0; rowIndex < maxRows; rowIndex++ {
		row := make([]interface{}, len(fields))
		for colIndex, field := range fields {
			row[colIndex] = generateExampleValueForRow(field, rowIndex, maxRows, username)
		}
		rows[rowIndex] = row
	}
	
	return rows
}

// generateExampleValueForRow 为指定行生成字段的示例值
// username: 当前用户名，用于创建用户字段的默认值
func generateExampleValueForRow(field *widget.Field, rowIndex int, maxRows int, username string) interface{} {
	dataType := "string"
	if field.Data != nil {
		dataType = field.Data.Type
	}
	
	widgetType := field.Widget.Type
	config, ok := field.Widget.Config.(map[string]interface{})
	
	// 处理系统字段
	switch field.Code {
	case "created_at", "updated_at":
		// 创建时间/更新时间：使用当前时间，格式为 2006-01-02 15:04:05
		now := time.Now()
		return now.Format("2006-01-02 15:04:05")
	case "create_by", "updated_by":
		// 创建用户/更新用户：使用当前用户名（必须获取到，否则返回空字符串）
		if username != "" {
			return username
		}
		return "" // 如果获取不到用户名，返回空字符串（前端会处理）
	}
	
	switch widgetType {
	case widget.TypeSelect, widget.TypeMultiSelect:
		// 获取所有选项，按行索引返回对应选项
		// Options 是 []string 类型，直接返回字符串
		if ok {
			// 尝试 []interface{} 类型（JSON 反序列化后）
			if options, ok := config["options"].([]interface{}); ok && len(options) > 0 {
				if rowIndex < len(options) {
					// 每个选项是字符串
					if optionStr, ok := options[rowIndex].(string); ok {
						return optionStr
					}
				}
				// 如果行数超过选项数，返回最后一个选项
				if len(options) > 0 {
					if optionStr, ok := options[len(options)-1].(string); ok {
						return optionStr
					}
				}
			} else if optionsStr, ok := config["options"].([]string); ok && len(optionsStr) > 0 {
				// 尝试 []string 类型（直接类型断言）
				if rowIndex < len(optionsStr) {
					return optionsStr[rowIndex]
				}
				// 如果行数超过选项数，返回最后一个选项
				return optionsStr[len(optionsStr)-1]
			}
		}
		// 没有选项，返回默认值
		return fmt.Sprintf("选项%d", rowIndex+1)
		
	case widget.TypeSwitch:
		// bool 类型：第一行显示"是"，第二行显示"否"
		if rowIndex == 0 {
			return "是"
		}
		return "否"
		
	case widget.TypeNumber, widget.TypeFloat:
		// 数字类型：使用默认值或示例数字
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return 123
		
	case widget.TypeTimestamp:
		// 日期类型：如果是创建时间/更新时间字段，使用当前时间；否则使用默认值或示例日期
		if field.Code == "created_at" || field.Code == "updated_at" {
			now := time.Now()
			return now.Format("2006-01-02 15:04:05")
		}
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return "2024-01-01"
		
	case widget.TypeTextArea:
		// 多行文本：使用默认值或示例文本
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return fmt.Sprintf("示例文本%d", rowIndex+1)
		
	case widget.TypeFiles:
		// 文件类型：显示为空
		return ""
		
	case widget.TypeUser:
		// 用户类型：如果是创建用户/更新用户字段，使用当前用户名；否则使用默认值
		if field.Code == "create_by" || field.Code == "updated_by" {
			if username != "" {
				return username
			}
			return "" // 如果获取不到用户名，返回空字符串（前端会处理）
		}
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return "" // 默认值为空字符串
		
	default:
		// 其他类型：根据 dataType 判断
		if dataType == widget.DataTypeInt || dataType == widget.DataTypeFloat {
			if ok {
				if defaultVal, ok := config["default"]; ok {
					return defaultVal
				}
			}
			return 123
		}
		// 文本类型：使用默认值或示例文本
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return fmt.Sprintf("示例文本%d", rowIndex+1)
	}
}


// TableUpdate Table 更新接口
// @Summary Table 更新
// @Description 更新表格记录
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "更新记录的字段数据（必须包含 id 字段）"
// @Success 200 {object} dto.RequestAppResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/update/{full-code-path} [put]
func (s *StandardAPI) TableUpdate(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 读取请求体，用于记录操作日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.FailWithMessage(c, "读取请求体失败: "+err.Error())
		return
	}
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes))) // 重新设置请求体，供后续使用

	// 构建回调请求对象（调用 OnTableUpdateRow）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableUpdateRow")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 解析请求体，用于记录操作日志
	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
		user, app, router, _ := parseFullCodePath(fullCodePath)
		logReq := &dto.RecordTableOperateLogReq{
			TenantUser:  user,
			RequestUser: req.RequestUser,
			App:         app,
			Router:      router,
			Action:      "OnTableUpdateRow",
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			TraceID:     req.TraceId,
		}

		// 获取 row_id
		if rowIDStr := c.Query("_row_id"); rowIDStr != "" {
			if id, err := strconv.ParseInt(rowIDStr, 10, 64); err == nil {
				logReq.RowID = id
			}
		} else if id, ok := bodyData["id"].(float64); ok {
			logReq.RowID = int64(id)
		}

		// 获取 updates 和 old_values
		if updatesData, ok := bodyData["updates"].(map[string]interface{}); ok {
			logReq.Updates, _ = json.Marshal(updatesData)
		}
		if oldValuesData, ok := bodyData["old_values"].(map[string]interface{}); ok {
			logReq.OldValues, _ = json.Marshal(oldValuesData)
		}

		// 异步记录操作日志
		ctx := contextx.ToContext(c)
		go func() {
			if err := s.appService.RecordTableOperateLog(ctx, logReq); err != nil {
				logger.Warnf(ctx, "[TableUpdate] 记录 Table 更新操作日志失败: %v", err)
			}
		}()
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// TableDelete Table 删除接口
// @Summary Table 删除
// @Description 删除表格记录（支持批量删除）
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "删除记录的ID列表，格式：{\"ids\": [1, 2, 3]}"
// @Success 200 {object} dto.RequestAppResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/table/delete/{full-code-path} [delete]
func (s *StandardAPI) TableDelete(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 读取请求体，用于记录操作日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.FailWithMessage(c, "读取请求体失败: "+err.Error())
		return
	}
	// 重新设置请求体，供后续使用
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// 构建回调请求对象（调用 OnTableDeleteRows）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableDeleteRows")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 解析请求体，用于记录操作日志
	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
		user, app, router, _ := parseFullCodePath(fullCodePath)
		logReq := &dto.RecordTableOperateLogReq{
			TenantUser:  user,
			RequestUser: req.RequestUser,
			App:         app,
			Router:      router,
			Action:      "OnTableDeleteRows",
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			TraceID:     req.TraceId,
		}

		// 获取 ids 列表
		if ids, ok := bodyData["ids"].([]interface{}); ok {
			rowIDs := make([]int64, 0, len(ids))
			for _, id := range ids {
				if idFloat, ok := id.(float64); ok {
					rowIDs = append(rowIDs, int64(idFloat))
				}
			}
			logReq.RowIDs = rowIDs
		}

		// 异步记录操作日志
		ctx := contextx.ToContext(c)
		go func() {
			if err := s.appService.RecordTableOperateLog(ctx, logReq); err != nil {
				logger.Warnf(ctx, "[TableDelete] 记录 Table 删除操作日志失败: %v", err)
			}
		}()
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// ============================================
// Form 函数接口
// ============================================

// FormSubmit Form 提交接口
// @Summary Form 提交
// @Description 提交表单数据
// @Tags 标准接口
// @Accept json
// @Accept application/x-www-form-urlencoded
// @Accept multipart/form-data
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "表单字段数据"
// @Success 200 {object} dto.RequestAppResp "提交成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/form/submit/{full-code-path} [post]
func (s *StandardAPI) FormSubmit(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "解析路径参数失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// ============================================
// Chart 函数接口
// ============================================

// ChartQuery Chart 查询接口
// @Summary Chart 查询
// @Description 查询图表数据
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param query query object false "图表查询条件"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/chart/query/{full-code-path} [get]
func (s *StandardAPI) ChartQuery(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, "解析路径参数失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

// ============================================
// Callback 接口
// ============================================

// CallbackOnSelectFuzzy 模糊搜索回调接口
// @Summary 模糊搜索回调
// @Description Select 组件的模糊搜索回调
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "搜索条件，格式：{\"code\": \"field_code\", \"type\": \"by_values\", \"value\": [1, 2, 3], \"request\": {...}}"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/callback/on_select_fuzzy/{full-code-path} [post]
func (s *StandardAPI) CallbackOnSelectFuzzy(c *gin.Context) {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "full-code-path 参数不能为空")
		return
	}

	// 构建回调请求对象（调用 OnSelectFuzzy）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnSelectFuzzy")
	if err != nil {
		response.FailWithMessage(c, "构建请求失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.FailWithMessage(c, err.Error(), metadata)
		return
	}

	if resp.Error != "" {
		response.Result(resp.ErrCode, nil, resp.Error, c, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}

