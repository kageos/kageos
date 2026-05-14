package apicall

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// UpdateAppResp 与 app_runtime_namespace 中定义一致，避免循环依赖时使用
var _ = (*dto.UpdateAppResp)(nil)

// GetServiceTreeDetailByFullCodePath 根据 full_code_path 获取服务目录详情（agent-server -> app-server）
// 用于从 full_code_path 解析出节点信息（含 id/tree_id），供 add_functions、权限等使用
func GetServiceTreeDetailByFullCodePath(ctx context.Context, fullCodePath string) (*dto.GetServiceTreeDetailResp, error) {
	return GetAPI[*dto.GetServiceTreeDetailResp](ctx, "/workspace/api/v1/service_tree/detail", buildQueryParams(
		withFullCodePathQuery(fullCodePath),
	))
}

// ServiceTreeAddFunctions 向服务目录添加函数（agent-server -> workspace）
// 将生成的代码写入到工作空间对应的目录下，并更新工作空间
// async: true 表示异步处理（通过回调通知），false 表示同步处理（直接返回结果）
func ServiceTreeAddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	return PostAPI[*dto.AddFunctionsReq, *dto.AddFunctionsResp](ctx, "/workspace/api/v1/service_tree/add_functions", req)
}

// SearchFunctions 搜索函数（agent-server -> app-server）
// 根据关键词、类型等条件搜索函数，支持分页
func SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	return GetAPI[*dto.SearchFunctionsResp](ctx, "/workspace/api/v1/service_tree/search_functions", buildQueryParams(
		withPaginationQuery(req.Page, req.PageSize),
		withTrimmedQueryValue("user", req.User),
		withTrimmedQueryValue("app", req.App),
		withTrimmedQueryValue("keyword", req.Keyword),
		withTrimmedQueryValue("template_type", req.TemplateType),
	))
}

// SearchResources 搜索服务树资源（agent-server -> app-server）。
func SearchResources(ctx context.Context, req *dto.SearchResourcesReq) (*dto.SearchResourcesResp, error) {
	return GetAPI[*dto.SearchResourcesResp](ctx, "/workspace/api/v1/service_tree/search_resources", buildQueryParams(
		withPaginationQuery(req.Page, req.PageSize),
		withTrimmedQueryValue("user", req.User),
		withTrimmedQueryValue("app", req.App),
		withTrimmedQueryValue("keyword", req.Keyword),
		withTrimmedQueryValue("resource_type", req.ResourceType),
	))
}

// GetFunctionInfo 根据 full_code_path 获取函数详情（agent-server -> app-server）。
func GetFunctionInfo(ctx context.Context, funcType string, fullCodePath string) (*dto.GetFunctionResp, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/function/info/"+strings.Trim(strings.TrimSpace(funcType), "/"), fullCodePath)
	return GetAPI[*dto.GetFunctionResp](ctx, path, nil)
}

// GetWorkspaceContext 获取工作台环境信息（agent-server -> app-server）
// fileSource 可选："" 或 "snapshot" 从快照表读；"runtime" 从 app-runtime 磁盘实时读（更准）
func GetWorkspaceContext(ctx context.Context, fullCodePath string, fileSource string) (*dto.GetWorkspaceContextResp, error) {
	return GetAPI[*dto.GetWorkspaceContextResp](ctx, "/workspace/api/v1/workspace/context", buildQueryParams(
		withFullCodePathQuery(fullCodePath),
		withTrimmedQueryValue("file_source", fileSource),
	))
}

// CreateDocs 创建 docs 类型节点（agent-server -> app-server）
// 使用现有接口 POST /workspace/api/v1/docs/crud
func CreateDocs(ctx context.Context, req *dto.CreateDocsReq) (*dto.CreateDocsResp, error) {
	return PostAPI[*dto.CreateDocsReq, *dto.CreateDocsResp](ctx, "/workspace/api/v1/docs/crud", req)
}

// UpdateDocs 更新 docs 类型节点（含文档内容）（agent-server -> app-server）
// 使用现有接口 PUT /workspace/api/v1/docs/crud/:id
func UpdateDocs(ctx context.Context, id int64, req *dto.UpdateDocsReq) error {
	path := fmt.Sprintf("/workspace/api/v1/docs/crud/%d", id)
	req.ID = id
	_, err := PutAPI[*dto.UpdateDocsReq, map[string]interface{}](ctx, path, req)
	return err
}

// CreatePackage 创建 package 类型节点（目录）（agent-server -> app-server）
// 使用现有接口 POST /workspace/api/v1/packages
func CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	return PostAPI[*dto.CreatePackageReq, *dto.CreatePackageResp](ctx, "/workspace/api/v1/packages", req)
}

// ReplaceFileContent 工作台文件 search-replace（agent-server -> app-server -> app-runtime 实时写盘）
func ReplaceFileContent(ctx context.Context, req *dto.ReplaceFileContentReq) (*dto.ReplaceFileContentResp, error) {
	return PostAPI[*dto.ReplaceFileContentReq, *dto.ReplaceFileContentResp](ctx, "/workspace/api/v1/workspace/files/replace", req)
}

// DeleteFile 工作台删除文件（删磁盘+删节点）（agent-server -> app-server）
func DeleteFile(ctx context.Context, req *dto.DeleteFileReq) (*dto.DeleteFileResp, error) {
	return PostAPI[*dto.DeleteFileReq, *dto.DeleteFileResp](ctx, "/workspace/api/v1/workspace/files/delete", req)
}

// ReadAppLog 读取应用日志（agent-server -> app-server）
func ReadAppLog(ctx context.Context, req *dto.ReadAppLogReq) (*dto.ReadAppLogResp, error) {
	return PostAPI[*dto.ReadAppLogReq, *dto.ReadAppLogResp](ctx, "/workspace/api/v1/workspace/logs/read", req)
}

// UpdateAppBuild 触发工作空间编译（仅编译不写文件，agent-server -> app-server）
// 统一走 canonical 入口 /workspace/api/v1/app/update，使用 resource_path 标识工作空间
func UpdateAppBuild(ctx context.Context, user, app string) (*dto.UpdateAppResp, error) {
	req := &dto.UpdateAppReq{
		ResourcePath: fmt.Sprintf("/%s/%s", user, app),
	}
	return PostAPI[*dto.UpdateAppReq, *dto.UpdateAppResp](ctx, "/workspace/api/v1/app/update", req)
}

// ========== 执行模式：查表 / 提交表单 / 查图表（agent 调用工作区标准接口） ==========

// TableSearch 调用工作区 Table 查询接口（GET table/search/{full-code-path}）
// fullCodePath 如 /luobei/myapp/tables/hr；queryParams 可含 page、page_size、sorts 等
func TableSearch(ctx context.Context, fullCodePath string, queryParams url.Values) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/table/search", fullCodePath)
	return GetAPI[map[string]interface{}](ctx, path, queryParams)
}

// FormSubmit 调用工作区 Form 提交接口（POST form/submit/{full-code-path}）
// fullCodePath 如 /luobei/myapp/plugins/cashier_desk；body 为表单字段 JSON
func FormSubmit(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/form/submit", fullCodePath)
	return PostAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// ChartQuery 调用工作区 Chart 查询接口（GET chart/query/{full-code-path}）
// fullCodePath 如 /luobei/myapp/charts/sales；queryParams 为图表查询条件
func ChartQuery(ctx context.Context, fullCodePath string, queryParams url.Values) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/chart/query", fullCodePath)
	return GetAPI[map[string]interface{}](ctx, path, queryParams)
}

// TableCreate 调用工作区 Table 新增接口（POST table/create/{full-code-path}）
// fullCodePath 为表格函数完整路径（如 /luobei/myapp/nps/nps_questionnaire_list）；body 为单条记录的字段 JSON，会触发 OnTableAddRow 回调
func TableCreate(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/table/create", fullCodePath)
	return PostAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// TableBatchCreate 调用工作区 Table 批量导入接口（POST table/batch-create/{full-code-path}）
// fullCodePath 为表格函数完整路径；body 为 { "data": [{ "field": "value" }] }，会触发 OnTableCreateInBatches 回调。
func TableBatchCreate(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/table/batch-create", fullCodePath)
	return PostAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// TableUpdate 调用工作区 Table 更新接口（PUT table/update/{full-code-path}）
// fullCodePath 为表格函数完整路径；body 为 { "id": 行ID, "updates": { "field": "value", ... } }，不传 old_values 时由 app-server 自动查表填充
func TableUpdate(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/table/update", fullCodePath)
	return PutAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// TableDelete 调用工作区 Table 删除接口（DELETE table/delete/{full-code-path}）
// fullCodePath 为表格函数完整路径；body 为 { "ids": [1, 2, 3] }，会触发 OnTableDeleteRows 回调。
func TableDelete(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/table/delete", fullCodePath)
	return DeleteBodyAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// CallbackOnSelectFuzzy 调用工作区 OnSelectFuzzy 回调（POST callback/on_select_fuzzy/{full-code-path}）
// 用于工作台测试带 OnSelectFuzzy 的表单/表格下拉回调查询。当前仅支持按关键词或空关键词：body 中 type=by_keyword，value=关键词（可为空）。不支持 by_value/by_values。fullCodePath 为配置了 OnSelectFuzzyMap 的 Form/Table 完整路径；body 含 code、type、value、request（可选）。
func CallbackOnSelectFuzzy(ctx context.Context, fullCodePath string, body map[string]interface{}) (map[string]interface{}, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/callback/on_select_fuzzy", fullCodePath)
	return PostAPI[map[string]interface{}, map[string]interface{}](ctx, path, body)
}

// CopyDirectoryViaWorkspace 通过 workspace API 复制本地目录（agent-server -> app-server）。
// source_directory_path 为本地完整目录路径；target_directory_path 为目标完整路径；target_app_id 由目标路径所在应用决定。
func CopyDirectoryViaWorkspace(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return PostAPI[*dto.CopyDirectoryReq, *dto.CopyDirectoryResp](ctx, "/workspace/api/v1/service_tree/copy", req)
}

// CreateScheduledTask 创建定时任务（agent-server -> app-server）
func CreateScheduledTask(ctx context.Context, req *dto.CreateScheduledTaskReq) (*dto.ScheduledTaskItem, error) {
	return PostAPI[*dto.CreateScheduledTaskReq, *dto.ScheduledTaskItem](ctx, "/workspace/api/v1/scheduled_tasks", req)
}

// ListScheduledTasks 查询定时任务列表（agent-server -> app-server）
func ListScheduledTasks(ctx context.Context, fullCodePath, status string, page, pageSize int) (*dto.ListScheduledTasksResp, error) {
	return GetAPI[*dto.ListScheduledTasksResp](ctx, "/workspace/api/v1/scheduled_tasks", buildQueryParams(
		withFullCodePathQuery(fullCodePath),
		withStatusQuery(status),
		withOptionalPaginationQuery(page, pageSize),
	))
}

// CancelScheduledTask 取消定时任务（agent-server -> app-server）
func CancelScheduledTask(ctx context.Context, taskID int64) error {
	path := fmt.Sprintf("/workspace/api/v1/scheduled_tasks/%d/cancel", taskID)
	_, err := PostAPI[map[string]interface{}, map[string]interface{}](ctx, path, map[string]interface{}{})
	return err
}

// ListScheduledTaskExecutions 查询某任务执行记录（agent-server -> app-server）
func ListScheduledTaskExecutions(ctx context.Context, taskID int64, status string, page, pageSize int) (*dto.ListScheduledTaskExecutionsResp, error) {
	path := fmt.Sprintf("/workspace/api/v1/scheduled_tasks/%d/executions", taskID)
	return GetAPI[*dto.ListScheduledTaskExecutionsResp](ctx, path, buildQueryParams(
		withStatusQuery(status),
		withOptionalPaginationQuery(page, pageSize),
	))
}
