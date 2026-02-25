package apicall

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// UpdateAppResp 与 app_runtime_namespace 中定义一致，避免循环依赖时使用
var _ = (*dto.UpdateAppResp)(nil)

// GetServiceTreeDetailByFullCodePath 根据 full_code_path 获取服务目录详情（agent-server -> app-server）
// 用于从 full_code_path 解析出节点信息（含 id/tree_id），供 add_functions、权限等使用
func GetServiceTreeDetailByFullCodePath(ctx context.Context, fullCodePath string) (*dto.GetServiceTreeDetailResp, error) {
	params := url.Values{}
	params.Set("full_code_path", fullCodePath)
	return GetAPI[*dto.GetServiceTreeDetailResp](ctx, "/workspace/api/v1/service_tree/detail", params)
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
	// 构建查询参数
	path := "/workspace/api/v1/service_tree/search_functions"
	params := url.Values{}
	params.Set("page", strconv.Itoa(req.Page))
	params.Set("page_size", strconv.Itoa(req.PageSize))
	if req.User != "" {
		params.Set("user", req.User)
	}
	if req.App != "" {
		params.Set("app", req.App)
	}
	if req.Keyword != "" {
		params.Set("keyword", req.Keyword)
	}
	if req.TemplateType != "" {
		params.Set("template_type", req.TemplateType)
	}

	return GetAPI[*dto.SearchFunctionsResp](ctx, path, params)
}

// CreateServiceTree 创建服务目录（agent-server -> app-server）
func CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	return PostAPI[*dto.CreateServiceTreeReq, *dto.CreateServiceTreeResp](ctx, "/workspace/api/v1/service_tree", req)
}

// GetServiceTreeByID 根据ID获取服务目录（agent-server -> app-server）
func GetServiceTreeByID(ctx context.Context, req *dto.GetServiceTreeByIDReq) (*dto.GetServiceTreeResp, error) {
	path := fmt.Sprintf("/workspace/api/v1/service_tree/%d", req.ID)
	return GetAPI[*dto.GetServiceTreeResp](ctx, path, nil)
}

// GetWorkspaceContext 获取工作台环境信息（agent-server -> app-server）
// fileSource 可选："" 或 "snapshot" 从快照表读；"runtime" 从 app-runtime 磁盘实时读（更准）
func GetWorkspaceContext(ctx context.Context, fullCodePath string, fileSource string) (*dto.GetWorkspaceContextResp, error) {
	params := url.Values{}
	params.Set("full_code_path", fullCodePath)
	if fileSource != "" {
		params.Set("file_source", fileSource)
	}
	return GetAPI[*dto.GetWorkspaceContextResp](ctx, "/workspace/api/v1/workspace/context", params)
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

// UpdateAppBuild 触发工作空间编译（仅编译不写文件，agent-server -> app-server）
// 路径为 /api/v1/app/update/{user}/{app}，body 传 {} 即可，只需 user 和 app 即可更新
func UpdateAppBuild(ctx context.Context, user, app string) (*dto.UpdateAppResp, error) {
	path := "/workspace/api/v1/app/update/" + url.PathEscape(user) + "/" + url.PathEscape(app)
	return PostAPI[*dto.UpdateAppReq, *dto.UpdateAppResp](ctx, path, &dto.UpdateAppReq{})
}

// ========== 执行模式：查表 / 提交表单 / 查图表（agent 调用工作区标准接口） ==========

// TableSearch 调用工作区 Table 查询接口（GET table/search/{full-code-path}）
// fullCodePath 如 /luobei/myapp/tables/hr；queryParams 可含 page、page_size、sorts 等
func TableSearch(ctx context.Context, fullCodePath string, queryParams url.Values) (map[string]interface{}, error) {
	path := "/workspace/api/v1/table/search" + fullCodePath
	return GetAPI[map[string]interface{}](ctx, path, queryParams)
}

// FormSubmit 调用工作区 Form 提交接口（POST form/submit/{full-code-path}）
// fullCodePath 如 /luobei/myapp/plugins/cashier_desk；body 为表单字段 JSON
func FormSubmit(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := "/workspace/api/v1/form/submit" + fullCodePath
	return PostAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// ChartQuery 调用工作区 Chart 查询接口（GET chart/query/{full-code-path}）
// fullCodePath 如 /luobei/myapp/charts/sales；queryParams 为图表查询条件
func ChartQuery(ctx context.Context, fullCodePath string, queryParams url.Values) (map[string]interface{}, error) {
	path := "/workspace/api/v1/chart/query" + fullCodePath
	return GetAPI[map[string]interface{}](ctx, path, queryParams)
}

// TableCreate 调用工作区 Table 新增接口（POST table/create/{full-code-path}）
// fullCodePath 为表格函数完整路径（如 /luobei/myapp/nps/nps_questionnaire_list）；body 为单条记录的字段 JSON，会触发 OnTableAddRow 回调
func TableCreate(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := "/workspace/api/v1/table/create" + fullCodePath
	return PostAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// TableUpdate 调用工作区 Table 更新接口（PUT table/update/{full-code-path}）
// fullCodePath 为表格函数完整路径；body 为 { "id": 行ID, "updates": { "field": "value", ... } }，不传 old_values 时由 app-server 自动查表填充
func TableUpdate(ctx context.Context, fullCodePath string, body interface{}) (map[string]interface{}, error) {
	path := "/workspace/api/v1/table/update" + fullCodePath
	return PutAPI[interface{}, map[string]interface{}](ctx, path, body)
}

// PublishDirectoryToHubViaWorkspace 通过 workspace API 发布目录到 Hub（agent-server -> app-server）
// 首次将当前工作区目录或指定目录发布到应用市场
func PublishDirectoryToHubViaWorkspace(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	return PostAPI[*dto.PublishDirectoryToHubReq, *dto.PublishDirectoryToHubResp](ctx, "/workspace/api/v1/service_tree/publish_to_hub", req)
}

// PushDirectoryToHubViaWorkspace 通过 workspace API 推送目录到 Hub（更新已发布的目录，agent-server -> app-server）
// 类似 git push，会递增版本号
func PushDirectoryToHubViaWorkspace(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	return PostAPI[*dto.PushDirectoryToHubReq, *dto.PushDirectoryToHubResp](ctx, "/workspace/api/v1/service_tree/push_to_hub", req)
}

// CopyDirectoryViaWorkspace 通过 workspace API 复制目录（支持从 Hub 链接复制到本地，agent-server -> app-server）
// source_directory_path 可为 hub://host/path@version；target_directory_path 为目标完整路径；target_app_id 由目标路径所在应用决定
func CopyDirectoryViaWorkspace(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return PostAPI[*dto.CopyDirectoryReq, *dto.CopyDirectoryResp](ctx, "/workspace/api/v1/service_tree/copy", req)
}
