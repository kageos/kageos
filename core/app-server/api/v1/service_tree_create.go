package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

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
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := req.ParentFullCodePath
	if resourcePath == "" {
		resourcePath = access.AppRootPath(req.User, req.App)
	}
	if err := requireAccess(c, s.teamAccessService, resourcePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreatePackage(ctx, &req)
	if err != nil {
		response.Internal(c, "创建目录失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.DirectoryPath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreateFunction(ctx, &req)
	if err != nil {
		response.Internal(c, "创建函数失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	resourcePath := req.ParentFullCodePath
	if resourcePath == "" {
		resourcePath = access.AppRootPath(req.User, req.App)
	}
	if err := requireAccess(c, s.teamAccessService, resourcePath, access.ActionWrite); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.serviceTreeService.CreateDocs(ctx, &req)
	if err != nil {
		response.Internal(c, "创建文档失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}
