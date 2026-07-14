package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

// GetUserInfo 获取当前登录用户信息
// @Summary 获取当前登录用户信息
// @Description 根据请求header中的username获取当前登录用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Success 200 {object} dto.UserInfo "用户信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 404 {string} string "用户不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/users/me [get]
func (u *User) GetUserInfo(c *gin.Context) {
	var resp *dto.UserInfo
	var err error
	defer func() {
		logger.Debugf(c, "GetUserInfo resp:%+v err:%v", resp, err)
	}()

	// 从context获取username（JWTAuth中间件已从header获取并设置到context）
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}

	// 查询用户信息
	ctx := contextx.ToContext(c)
	user, err := u.userService.GetUserByUsername(ctx, username)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := u.ensureSameCompany(c, user); err != nil {
		response.Error(c, err)
		return
	}

	// 转换为DTO（包含详细信息）
	userInfos := convertUsersToDTOBatch(ctx, []*model.User{user}, u.userService, u.departmentService)
	if len(userInfos) == 0 {
		response.Internal(c, "转换用户信息失败")
		return
	}
	resp = userInfos[0]
	response.OkWithData(c, resp)
}

// UpdateUser 更新用户信息
// @Summary 更新用户信息
// @Description 更新当前登录用户的昵称、签名、头像、性别等信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param request body dto.UpdateUserReq true "更新用户信息请求"
// @Success 200 {object} dto.UpdateUserResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 404 {string} string "用户不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/users/me [put]
func (u *User) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserReq
	var resp *dto.UpdateUserResp
	var err error
	defer func() {
		logger.Debugf(c, "UpdateUser req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 从context获取username
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}

	// 检查是否有字段需要更新
	if req.Nickname == nil && req.Signature == nil && req.Avatar == nil && req.Gender == nil {
		response.BadRequest(c, "至少需要提供一个更新字段")
		return
	}

	// 更新用户信息（直接传递指针，nil 表示不更新，非 nil 表示更新）
	ctx := contextx.ToContext(c)
	user, err := u.userService.UpdateUser(ctx, username, req.Nickname, req.Signature, req.Avatar, req.Gender)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 转换为DTO（包含详细信息）
	userInfos := convertUsersToDTOBatch(ctx, []*model.User{user}, u.userService, u.departmentService)
	if len(userInfos) == 0 {
		response.Internal(c, "转换用户信息失败")
		return
	}
	resp = &dto.UpdateUserResp{
		User: *userInfos[0],
	}
	response.OkWithData(c, resp)
}
