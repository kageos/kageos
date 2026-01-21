package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Department 部门相关API
type Department struct {
	deptService *service.DepartmentService
}

// NewDepartment 创建部门API（依赖注入）
func NewDepartment(deptService *service.DepartmentService) *Department {
	return &Department{
		deptService: deptService,
	}
}

// CreateDepartment 创建部门
// @Summary 创建部门
// @Description 创建新部门
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param request body dto.CreateDepartmentReq true "创建部门请求"
// @Success 200 {object} dto.CreateDepartmentResp
// @Router /hr/api/v1/department [post]
func (d *Department) CreateDepartment(c *gin.Context) {
	var req dto.CreateDepartmentReq
	var resp *dto.CreateDepartmentResp
	var err error
	defer func() {
		logger.Infof(c, "CreateDepartment req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务层
	ctx := c.Request.Context()
	department, err := d.deptService.CreateDepartment(ctx, req.Name, req.Code, req.ParentID, req.Description, req.Managers)
	if err != nil {
		response.FailWithMessage(c, "创建部门失败: "+err.Error())
		return
	}

	resp = &dto.CreateDepartmentResp{
		Department: department,
	}
	response.OkWithData(c, resp)
}

// UpdateDepartment 更新部门
// @Summary 更新部门
// @Description 更新部门信息
// @Tags 部门管理
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param id path int true "部门ID"
// @Param request body dto.UpdateDepartmentReq true "更新部门请求"
// @Success 200 {object} dto.UpdateDepartmentResp
// @Router /hr/api/v1/department/{id} [put]
func (d *Department) UpdateDepartment(c *gin.Context) {
	var req dto.UpdateDepartmentReq
	var resp *dto.UpdateDepartmentResp
	var err error
	defer func() {
		logger.Infof(c, "UpdateDepartment req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 解析部门ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "部门ID格式错误")
		return
	}

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	name := ""
	description := ""
	managers := ""
	status := ""
	sort := -1

	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = *req.Description
	}
	if req.Managers != nil {
		managers = *req.Managers
	}
	if req.Status != nil {
		status = *req.Status
	}
	if req.Sort != nil {
		sort = *req.Sort
	}

	department, err := d.deptService.UpdateDepartment(ctx, id, name, description, managers, status, sort)
	if err != nil {
		response.FailWithMessage(c, "更新部门失败: "+err.Error())
		return
	}

	resp = &dto.UpdateDepartmentResp{
		Department: department,
	}
	response.OkWithData(c, resp)
}

// GetDepartmentTree 获取部门树
// @Summary 获取部门树
// @Description 获取完整的部门树结构
// @Tags 部门管理
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Success 200 {object} dto.GetDepartmentTreeResp
// @Router /hr/api/v1/department/tree [get]
func (d *Department) GetDepartmentTree(c *gin.Context) {
	var resp *dto.GetDepartmentTreeResp
	var err error
	defer func() {
		logger.Infof(c, "GetDepartmentTree resp:%+v err:%v", resp, err)
	}()

	// 调用服务层
	ctx := contextx.ToContext(c)
	tree, err := d.deptService.GetDepartmentTree(ctx)
	if err != nil {
		response.FailWithMessage(c, "获取部门树失败: "+err.Error())
		return
	}

	resp = &dto.GetDepartmentTreeResp{
		Departments: tree,
	}
	response.OkWithData(c, resp)
}

// GetDepartmentByID 根据ID获取部门
// @Summary 获取部门详情
// @Description 根据ID获取部门详细信息
// @Tags 部门管理
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param id path int true "部门ID"
// @Success 200 {object} dto.GetDepartmentResp
// @Router /hr/api/v1/department/{id} [get]
func (d *Department) GetDepartmentByID(c *gin.Context) {
	var resp *dto.GetDepartmentResp
	var err error
	defer func() {
		logger.Infof(c, "GetDepartmentByID resp:%+v err:%v", resp, err)
	}()

	// 解析部门ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "部门ID格式错误")
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	department, err := d.deptService.GetDepartmentByID(ctx, id)
	if err != nil {
		response.FailWithMessage(c, "获取部门失败: "+err.Error())
		return
	}

	resp = &dto.GetDepartmentResp{
		Department: department,
	}
	response.OkWithData(c, resp)
}

// DeleteDepartment 删除部门
// @Summary 删除部门
// @Description 删除部门（软删除）
// @Tags 部门管理
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param id path int true "部门ID"
// @Success 200 {object} dto.Response
// @Router /hr/api/v1/department/{id} [delete]
func (d *Department) DeleteDepartment(c *gin.Context) {
	var err error
	defer func() {
		logger.Infof(c, "DeleteDepartment err:%v", err)
	}()

	// 解析部门ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "部门ID格式错误")
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	if err = d.deptService.DeleteDepartment(ctx, id); err != nil {
		response.FailWithMessage(c, "删除部门失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}
