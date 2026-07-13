package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

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
		response.BadRequest(c, "参数错误: 无效的ID")
		return
	}

	var req dto.UpdatePackageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.Internal(c, "获取目录失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.serviceTreeService.UpdatePackage(ctx, &req); err != nil {
		response.Internal(c, "更新目录失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: 无效的ID")
		return
	}

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.Internal(c, "获取目录失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionDelete); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.serviceTreeService.DeletePackage(ctx, id); err != nil {
		response.Internal(c, "删除目录失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: 无效的ID")
		return
	}

	var req dto.UpdateFunctionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.Internal(c, "获取函数失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.serviceTreeService.UpdateFunction(ctx, &req); err != nil {
		response.Internal(c, "更新函数失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: 无效的ID")
		return
	}

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.Internal(c, "获取函数失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionDelete); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.serviceTreeService.DeleteFunction(ctx, id); err != nil {
		response.Internal(c, "删除函数失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: 无效的ID")
		return
	}

	var req dto.UpdateDocsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.ID = id

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.Internal(c, "获取文档失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionUpdate); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.serviceTreeService.UpdateDocs(ctx, &req); err != nil {
		response.Internal(c, "更新文档失败: "+err.Error())
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
		response.BadRequest(c, "参数错误: 无效的ID")
		return
	}

	ctx := contextx.ToContext(c)
	serviceTree, err := s.serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{ID: id})
	if err != nil {
		response.Internal(c, "获取文档失败: "+err.Error())
		return
	}
	if err := requireAccess(c, s.teamAccessService, serviceTree.FullCodePath, access.ActionDelete); err != nil {
		response.Error(c, err)
		return
	}
	if err := s.serviceTreeService.DeleteDocs(ctx, id); err != nil {
		response.Internal(c, "删除文档失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// CopyServiceTree 复制服务目录（递归复制目录及其所有子目录）
// @Summary 复制服务目录
// @Description 递归复制服务目录及其所有子目录到目标父目录下；target_directory_name 可修改复制后根目录中文展示名，同名目录已存在时可通过 replace_existing 完全替换
// @Tags 服务目录
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CopyDirectoryReq true "复制请求，source_directory_path=源目录完整路径，target_directory_path=目标父目录完整路径，target_directory_name=复制后根目录中文展示名，replace_existing=是否覆盖同名目录"
// @Success 200 {object} dto.CopyDirectoryResp "复制成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/directory-copies [post]
func (s *ServiceTree) CopyServiceTree(c *gin.Context) {
	var req dto.CopyDirectoryReq
	var resp *dto.CopyDirectoryResp
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Debugf(c, "CopyServiceTree req:%+v resp:%+v err:%v", req, resp, err)
	}()
	if err := requireAccess(c, s.teamAccessService, req.SourceDirectoryPath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}
	if err := requireAccess(c, s.teamAccessService, req.TargetDirectoryPath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	resp, err = s.serviceTreeService.CopyServiceTree(ctx, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OkWithData(c, resp)
}
