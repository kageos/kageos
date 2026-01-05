package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// UserAllocation 用户分配相关API
type UserAllocation struct {
	userService       *service.UserService
	departmentService *service.DepartmentService
}

// NewUserAllocation 创建用户分配API（依赖注入）
func NewUserAllocation(userService *service.UserService, departmentService *service.DepartmentService) *UserAllocation {
	return &UserAllocation{
		userService:       userService,
		departmentService: departmentService,
	}
}

// AssignUser 分配用户组织架构
// @Summary 分配用户组织架构
// @Description 分配用户的部门和 Leader
// @Tags 用户分配
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param request body dto.AssignUserReq true "分配用户请求"
// @Success 200 {object} dto.AssignUserResp
// @Router /hr/api/v1/user/assign [post]
func (u *UserAllocation) AssignUser(c *gin.Context) {
	var req dto.AssignUserReq
	var resp *dto.AssignUserResp
	var err error
	defer func() {
		logger.Infof(c, "AssignUser req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	user, err := u.userService.AssignUserOrganization(ctx, req.Username, req.DepartmentFullPath, req.LeaderUsername)
	if err != nil {
		response.FailWithMessage(c, "分配失败: "+err.Error())
		return
	}

	// 转换为DTO（包含详细信息）
	userInfos := convertUsersToDTOBatch(ctx, []*model.User{user}, u.userService, u.departmentService)
	if len(userInfos) == 0 {
		response.FailWithMessage(c, "转换用户信息失败")
		return
	}
	resp = &dto.AssignUserResp{
		User: *userInfos[0],
	}
	response.OkWithData(c, resp)
}

// GetUsersByDepartment 根据部门完整路径获取用户列表
// @Summary 根据部门获取用户列表
// @Description 根据部门完整路径获取该部门下的所有用户
// @Tags 用户分配
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param department_full_path query string true "部门完整路径"
// @Success 200 {object} dto.GetUsersByDepartmentResp
// @Router /hr/api/v1/user/department [get]
func (u *UserAllocation) GetUsersByDepartment(c *gin.Context) {
	var req dto.GetUsersByDepartmentReq
	var resp *dto.GetUsersByDepartmentResp
	var err error
	defer func() {
		logger.Infof(c, "GetUsersByDepartment req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	users, err := u.userService.GetUsersByDepartmentFullPath(ctx, req.DepartmentFullPath)
	if err != nil {
		response.FailWithMessage(c, "获取部门用户失败: "+err.Error())
		return
	}

	// 转换为DTO（包含详细信息，批量查询）
	dtoUserInfos := convertUsersToDTOBatch(ctx, users, u.userService, u.departmentService)
	resultUserInfos := make([]dto.UserInfo, 0, len(dtoUserInfos))
	for _, userInfo := range dtoUserInfos {
		resultUserInfos = append(resultUserInfos, *userInfo)
	}

	resp = &dto.GetUsersByDepartmentResp{
		Users: resultUserInfos,
	}
	response.OkWithData(c, resp)
}

