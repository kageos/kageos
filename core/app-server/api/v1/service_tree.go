package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

type ServiceTree struct {
	serviceTreeService *service.ServiceTreeService
	teamAccessService  *service.TeamAccessService
}

// NewServiceTree 创建 ServiceTree 处理器（依赖注入）
func NewServiceTree(serviceTreeService *service.ServiceTreeService, teamAccessService *service.TeamAccessService) *ServiceTree {
	return &ServiceTree{
		serviceTreeService: serviceTreeService,
		teamAccessService:  teamAccessService,
	}
}

// CreatePackage 创建 package 类型节点（专门的接口）
// @Summary 创建目录
// @Description 创建 package 类型的服务目录节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreatePackageReq true "创建目录请求"
// @Success 200 {object} dto.CreatePackageResp
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/packages [post]
func (s *ServiceTree) CreatePackage(c *gin.Context) {
	var req dto.CreatePackageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := req.ParentFullCodePath
	if resourcePath == "" {
		resourcePath = access.AppRootPath(req.User, req.App)
	}
	if err := requireAccess(c, s.teamAccessService, resourcePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreatePackage(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "创建目录失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// CreateFunction 创建 function 类型节点（专门的接口）
// @Summary 创建函数
// @Description 创建 function 类型的函数节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreateFunctionReq true "创建函数请求"
// @Success 200 {object} dto.CreateFunctionResp
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/functions [post]
func (s *ServiceTree) CreateFunction(c *gin.Context) {
	var req dto.CreateFunctionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.DirectoryPath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreateFunction(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "创建函数失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// CreateDocs 创建 docs 类型节点（专门的接口）
// @Summary 创建文档
// @Description 创建 docs 类型的文档节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreateDocsReq true "创建文档请求"
// @Success 200 {object} dto.CreateDocsResp
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/docs/crud [post]
func (s *ServiceTree) CreateDocs(c *gin.Context) {
	var req dto.CreateDocsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := req.ParentFullCodePath
	if resourcePath == "" {
		resourcePath = access.AppRootPath(req.User, req.App)
	}
	if err := requireAccess(c, s.teamAccessService, resourcePath, access.ActionWrite); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreateDocs(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "创建文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetServiceTreeDetail 获取服务目录详情
// @Summary 获取服务目录详情
// @Description 根据ID或full-code-path获取服务目录详情
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id query int false "服务目录ID（优先使用）"
// @Param full_code_path query string false "完整代码路径（如果未提供ID则使用）"
// @Success 200 {object} dto.GetServiceTreeDetailResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "服务目录不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/service_tree/detail [get]
func (s *ServiceTree) GetServiceTreeDetail(c *gin.Context) {
	var req dto.GetServiceTreeDetailReq

	// 从 query 参数获取 ID
	idStr := c.Query("id")
	if idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.FailWithMessage(c, "无效的ID参数")
			return
		}
		req.ID = id
	}

	// 从 query 参数获取 full_code_path
	req.FullCodePath = c.Query("full_code_path")

	if req.ID == 0 && req.FullCodePath == "" {
		response.FailWithMessage(c, "必须提供 ID 或 full_code_path 参数")
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "获取服务目录详情失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, resp.FullCodePath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// UpdatePackage 更新 package 类型节点（专门的接口）
// @Summary 更新目录
// @Description 更新 package 类型的服务目录节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "目录ID"
// @Param request body dto.UpdatePackageReq true "更新目录请求"
// @Success 200 {string} string "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/packages/{id} [put]
func (s *ServiceTree) UpdatePackage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}

	var req dto.UpdatePackageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.FailWithMessage(c, "获取目录失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.serviceTreeService.UpdatePackage(ctx, &req); err != nil {
		response.FailWithMessage(c, "更新目录失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "更新成功")
}

// DeletePackage 删除 package 类型节点（专门的接口）
// @Summary 删除目录
// @Description 删除 package 类型的服务目录节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "目录ID"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/packages/{id} [delete]
func (s *ServiceTree) DeletePackage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.FailWithMessage(c, "获取目录失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionDelete); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.serviceTreeService.DeletePackage(ctx, id); err != nil {
		response.FailWithMessage(c, "删除目录失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// UpdateFunction 更新 function 类型节点（专门的接口）
// @Summary 更新函数
// @Description 更新 function 类型的函数节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "函数ID"
// @Param request body dto.UpdateFunctionReq true "更新函数请求"
// @Success 200 {string} string "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/functions/{id} [put]
func (s *ServiceTree) UpdateFunction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}

	var req dto.UpdateFunctionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.FailWithMessage(c, "获取函数失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.serviceTreeService.UpdateFunction(ctx, &req); err != nil {
		response.FailWithMessage(c, "更新函数失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "更新成功")
}

// DeleteFunction 删除 function 类型节点（专门的接口）
// @Summary 删除函数
// @Description 删除 function 类型的函数节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "函数ID"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/functions/{id} [delete]
func (s *ServiceTree) DeleteFunction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.FailWithMessage(c, "获取函数失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionDelete); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.serviceTreeService.DeleteFunction(ctx, id); err != nil {
		response.FailWithMessage(c, "删除函数失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// UpdateDocs 更新 docs 类型节点（专门的接口）
// @Summary 更新文档
// @Description 更新 docs 类型的文档节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "文档ID"
// @Param request body dto.UpdateDocsReq true "更新文档请求"
// @Success 200 {string} string "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/docs/crud/{id} [put]
func (s *ServiceTree) UpdateDocs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}

	var req dto.UpdateDocsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.FailWithMessage(c, "获取文档失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionUpdate); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.serviceTreeService.UpdateDocs(ctx, &req); err != nil {
		response.FailWithMessage(c, "更新文档失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "更新成功")
}

// DeleteDocs 删除 docs 类型节点（专门的接口）
// @Summary 删除文档
// @Description 删除 docs 类型的文档节点
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param id path int true "文档ID"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /workspace/api/v1/docs/crud/{id} [delete]
func (s *ServiceTree) DeleteDocs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "参数错误: 无效的ID")
		return
	}

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.FailWithMessage(c, "获取文档失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionDelete); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.serviceTreeService.DeleteDocs(ctx, id); err != nil {
		response.FailWithMessage(c, "删除文档失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// CopyServiceTree 复制服务目录（递归复制目录及其所有子目录）
// @Summary 复制服务目录
// @Description 递归复制服务目录及其所有子目录到目标目录，保持目录结构
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CopyDirectoryReq true "复制请求，source_directory_path=源目录完整路径，target_directory_path=目标目录完整路径"
// @Success 200 {object} dto.CopyDirectoryResp "复制成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/service_tree/copy [post]
func (s *ServiceTree) CopyServiceTree(c *gin.Context) {
	var req dto.CopyDirectoryReq
	var resp *dto.CopyDirectoryResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "CopyServiceTree req:%+v resp:%+v err:%v", req, resp, err)
	}()
	if err := requireAccess(c, s.teamAccessService, req.SourceDirectoryPath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.TargetDirectoryPath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err = s.serviceTreeService.CopyServiceTree(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// AddFunctions 向服务目录添加函数（服务间调用，不需要JWT验证）
// @Summary 向服务目录添加函数
// @Description 接收来自 agent-server 的 Go 源码，写入到工作空间对应目录；默认同步构建，skip_build=true 时仅写文件。
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param X-Trace-Id header string false "追踪ID（用于链路追踪）"
// @Param X-Request-User header string false "请求用户（用于审计）"
// @Param X-Token header string false "Token（服务间调用时透传）"
// @Param request body dto.AddFunctionsReq true "添加函数请求"
// @Success 200 {object} dto.AddFunctionsResp "处理成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/service_tree/add_functions [post]
func (s *ServiceTree) AddFunctions(c *gin.Context) {
	var req dto.AddFunctionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(c, "[ServiceTree API] 解析请求失败: %v", err)
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp, err := s.serviceTreeService.AddFunctions(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[ServiceTree API] 处理失败: %v", err)
		response.FailWithMessage(c, "处理失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// ExportCapabilityBundle 导出标准能力包 JSON。
func (s *ServiceTree) ExportCapabilityBundle(c *gin.Context) {
	var req dto.ExportCapabilityBundleReq
	if c.Request.Method == http.MethodGet {
		if err := c.ShouldBindQuery(&req); err != nil {
			response.FailWithMessage(c, "参数错误: "+err.Error())
			return
		}
	} else if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	sourcePaths := req.SourceDirectoryPaths
	if req.SourceDirectoryPath != "" {
		sourcePaths = append(sourcePaths, req.SourceDirectoryPath)
	}
	if req.SourceRootPath != "" {
		sourcePaths = append(sourcePaths, req.SourceRootPath)
	}
	for _, sourcePath := range sourcePaths {
		if err := requireAccess(c, s.teamAccessService, sourcePath, access.ActionRead); err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
	}
	resp, err := s.serviceTreeService.ExportCapabilityBundle(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// InstallCapabilityBundle 将能力包安装到目标目录节点下。
func (s *ServiceTree) InstallCapabilityBundle(c *gin.Context) {
	var req dto.InstallCapabilityBundleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := requireAccess(c, s.teamAccessService, req.TargetDirectoryPath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	resp, err := s.serviceTreeService.InstallCapabilityBundle(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithDetailed(c, resp, resp.Message)
}

// SearchFunctions 搜索函数
// @Summary 搜索函数
// @Description 根据关键词、类型等条件搜索函数，支持分页
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string false "用户名（可选，用于过滤应用）"
// @Param app query string false "应用名（可选，用于过滤应用）"
// @Param keyword query string false "搜索关键词（可选，用于搜索名称和路径）"
// @Param template_type query string false "模板类型过滤（可选，如：form、table、chart）"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.SearchFunctionsResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/service_tree/search_functions [get]
func (s *ServiceTree) SearchFunctions(c *gin.Context) {
	var req dto.SearchFunctionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	// 验证分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大每页数量
	}

	ctx := contextx.ToContext(c)
	req.CurrentUser = contextx.GetRequestUser(ctx)
	resp, err := s.serviceTreeService.SearchFunctions(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "搜索函数失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// SearchResources 全站资源搜索
// @Summary 全站资源搜索
// @Description 根据关键词搜索目录、函数和文档，支持分页
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param user query string false "用户名（可选，用于过滤应用）"
// @Param app query string false "应用名（可选，用于过滤应用）"
// @Param keyword query string false "搜索关键词"
// @Param resource_type query string false "资源类型（all/package/function/docs）"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(20)
// @Success 200 {object} dto.SearchResourcesResp "搜索成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/service_tree/search_resources [get]
func (s *ServiceTree) SearchResources(c *gin.Context) {
	var req dto.SearchResourcesReq
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
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	ctx := contextx.ToContext(c)
	req.CurrentUser = contextx.GetRequestUser(ctx)
	resp, err := s.serviceTreeService.SearchResources(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "搜索资源失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetWorkspaceContext 获取工作台环境信息
// GET /workspace/api/v1/workspace/context?full_code_path=...&file_source=snapshot|runtime
func (s *ServiceTree) GetWorkspaceContext(c *gin.Context) {
	var req dto.GetWorkspaceContextReq
	req.FullCodePath = c.Query("full_code_path")
	req.FileSource = c.Query("file_source")
	if req.FullCodePath == "" {
		response.FailWithMessage(c, "full_code_path 必填")
		return
	}

	ctx := contextx.ToContext(c)
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionRead); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	resp, err := s.serviceTreeService.GetWorkspaceContext(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "获取工作台环境信息失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// ReplaceFileContent 工作台文件 search-replace（实时写盘）
// POST /workspace/api/v1/workspace/files/replace
func (s *ServiceTree) ReplaceFileContent(c *gin.Context) {
	var req dto.ReplaceFileContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" || req.FileName == "" || len(req.Replacements) == 0 {
		response.FailWithMessage(c, "full_code_path、file_name、replacements 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.ReplaceFileContent(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "替换文件失败: "+err.Error())
		return
	}
	if !resp.Success {
		response.FailWithDetailed(c, resp, resp.Message)
		return
	}
	response.OkWithData(c, resp)
}

// DeleteFile 工作台删除文件（删磁盘+删节点）
// POST /workspace/api/v1/workspace/files/delete
func (s *ServiceTree) DeleteFile(c *gin.Context) {
	var req dto.DeleteFileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" || req.FileName == "" {
		response.FailWithMessage(c, "full_code_path、file_name 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionDelete); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.DeleteFile(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "删除文件失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// ReadAppLog 读取应用日志（支持 version、关键词检索）
// POST /workspace/api/v1/workspace/logs/read
func (s *ServiceTree) ReadAppLog(c *gin.Context) {
	var req dto.ReadAppLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}
	if req.FullCodePath == "" {
		response.FailWithMessage(c, "full_code_path 必填")
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.ReadAppLog(ctx, &req)
	if err != nil {
		response.FailWithMessage(c, "读取日志失败: "+err.Error())
		return
	}
	response.OkWithData(c, resp)
}
