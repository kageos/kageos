package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/xuri/excelize/v2"
)

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
// @Param sorts query string false "排序（可选，格式：-id,name）"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/tables/{full-code-path} [get]
func (s *StandardAPI) TableSearch(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}

	// 构建请求对象
	req, err := s.buildRequestAppReq(c, fullCodePath)
	if err != nil {
		response.Error(c, err)
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
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	s.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(fullCodePath, "/"))
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
// @Router /workspace/api/v1/tables/{full-code-path} [post]
func (s *StandardAPI) TableCreate(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionWrite); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.ensureTableCallbackEnabled(c, fullCodePath, "OnTableAddRow", "该表未开启新增能力，通常是只读查询表，不支持新增"); err != nil {
		response.Error(c, err)
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "读取请求体失败: "+err.Error())
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// 构建回调请求对象（调用 OnTableAddRow）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableAddRow")
	if err != nil {
		response.Error(c, err)
		return
	}

	user, app, router, _ := parseFullCodePath(fullCodePath)
	logReq := &dto.RecordTableActionLogReq{
		TenantUser:  user,
		RequestUser: req.RequestUser,
		App:         app,
		Router:      router,
		Action:      "OnTableAddRow",
		Source:      c.GetHeader(contextx.ClientSourceHeader),
		Body:        bodyBytes,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		TraceID:     req.TraceId,
	}
	ctx := contextx.ToContext(c)

	// 调用服务层
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	fillTableActionLogResult(logReq, resp, err, mill)
	if logErr := s.appService.RecordTableActionLog(ctx, logReq); logErr != nil {
		logger.Warnf(ctx, "[TableCreate] 记录 Table 新增操作日志失败: %v", logErr)
	}

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	s.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(fullCodePath, "/"))
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
// @Router /workspace/api/v1/table-import-templates/{full-code-path} [get]
func (s *StandardAPI) TableTemplate(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)

	// 获取函数信息（直接使用 full-code-path）
	function, err := s.appService.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 获取当前用户信息（用于创建用户字段的默认值）
	username := contextx.GetRequestUser(c)

	editableFields := functionschema.TableCreateFields(function.Schema)

	if len(editableFields) == 0 {
		response.MethodNotAllowed(c, "该表没有可编辑字段，不能生成导入模板")
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
		// 如果已经写入部分数据，不能再写 JSON 错误响应。
		// 这里只能记录错误，无法返回错误响应
		return
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
// @Router /workspace/api/v1/tables/{full-code-path} [put]
func (s *StandardAPI) TableUpdate(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionUpdate); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.ensureTableCallbackEnabled(c, fullCodePath, "OnTableUpdateRow", "该表未开启编辑能力，通常是只读查询表，不支持更新"); err != nil {
		response.Error(c, err)
		return
	}

	// 读取请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "读取请求体失败: "+err.Error())
		return
	}

	var bodyData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyData); err != nil {
		response.BadRequest(c, "请求体必须是合法 JSON")
		return
	}

	// 能力下沉：若调用方未传 old_values，则内部按 id 查表取当前行并自动填充，方便上层只传 id + updates
	if needFillOldValues(bodyData) {
		id, ok := getBodyIDInt64(bodyData)
		if !ok {
			response.BadRequest(c, "请求体缺少有效 id，无法自动填充 old_values")
			return
		}
		rows, err := s.fetchTableRowsByIDs(c, fullCodePath, []int64{id})
		if err != nil {
			response.Internal(c, "自动查询当前行失败: "+err.Error())
			return
		}
		oldRow := findTableRowByID(rows, id)
		if oldRow == nil {
			response.NotFound(c, "记录不存在（id 未查到数据），无法填充 old_values")
			return
		}
		bodyData["old_values"] = oldRow
		newBodyBytes, err := json.Marshal(bodyData)
		if err != nil {
			response.Internal(c, "构造 old_values 后序列化失败: "+err.Error())
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(newBodyBytes))
	} else {
		// 调用方已传 old_values，body 已在上面被 ReadAll 消费，需恢复供 buildCallbackAppReq 再次读取
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// 构建回调请求对象（调用 OnTableUpdateRow）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableUpdateRow")
	if err != nil {
		response.Error(c, err)
		return
	}

	// 使用已解析的 bodyData（可能已自动填充 old_values）准备操作日志
	var logReq *dto.RecordTableActionLogReq
	if bodyData != nil {
		user, app, router, _ := parseFullCodePath(fullCodePath)
		logReq = &dto.RecordTableActionLogReq{
			TenantUser:  user,
			RequestUser: req.RequestUser,
			App:         app,
			Router:      router,
			Action:      "OnTableUpdateRow",
			Source:      c.GetHeader(contextx.ClientSourceHeader),
			IPAddress:   c.ClientIP(),
			UserAgent:   c.GetHeader("User-Agent"),
			TraceID:     req.TraceId,
		}

		// 获取 row_id
		if rowIDStr := c.Query("_row_id"); rowIDStr != "" {
			if id, err := strconv.ParseInt(rowIDStr, 10, 64); err == nil {
				logReq.RowID = id
			}
		} else if id, ok := getBodyIDInt64(bodyData); ok {
			logReq.RowID = id
		}

		// 获取 updates 和 old_values
		if updatesData, ok := bodyData["updates"].(map[string]interface{}); ok {
			logReq.Updates, _ = json.Marshal(updatesData)
		}
		if oldValuesData, ok := bodyData["old_values"].(map[string]interface{}); ok {
			logReq.OldValues, _ = json.Marshal(oldValuesData)
		}

	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	fillTableActionLogResult(logReq, resp, err, mill)
	if logReq != nil {
		if logErr := s.appService.RecordTableActionLog(ctx, logReq); logErr != nil {
			logger.Warnf(ctx, "[TableUpdate] 记录 Table 更新操作日志失败: %v", logErr)
		}
	}

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	s.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(fullCodePath, "/"))
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
// @Router /workspace/api/v1/tables/{full-code-path} [delete]
func (s *StandardAPI) TableDelete(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireAccess(c, s.teamAccessService, fullCodePath, access.ActionDelete); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.ensureTableCallbackEnabled(c, fullCodePath, "OnTableDeleteRows", "该表未开启删除能力，不支持删除"); err != nil {
		response.Error(c, err)
		return
	}

	// 读取请求体，用于记录操作日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "读取请求体失败: "+err.Error())
		return
	}
	// 重新设置请求体，供后续使用
	c.Request.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// 构建回调请求对象（调用 OnTableDeleteRows）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnTableDeleteRows")
	if err != nil {
		response.Error(c, err)
		return
	}

	// 解析请求体，用于记录操作日志
	var bodyData map[string]interface{}
	var logReq *dto.RecordTableActionLogReq
	if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
		user, app, router, _ := parseFullCodePath(fullCodePath)
		logReq = &dto.RecordTableActionLogReq{
			TenantUser:  user,
			RequestUser: req.RequestUser,
			App:         app,
			Router:      router,
			Action:      "OnTableDeleteRows",
			Source:      c.GetHeader(contextx.ClientSourceHeader),
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
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	fillTableActionLogResult(logReq, resp, err, mill)
	if logReq != nil {
		if logErr := s.appService.RecordTableActionLog(ctx, logReq); logErr != nil {
			logger.Warnf(ctx, "[TableDelete] 记录 Table 删除操作日志失败: %v", logErr)
		}
	}

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	s.appService.IncrementFunctionRunCount(ctx, "/"+strings.TrimPrefix(fullCodePath, "/"))
	response.OkWithData(c, resp.Result, metadata)
}
