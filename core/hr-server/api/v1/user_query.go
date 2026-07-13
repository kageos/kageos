package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

// QueryUser 根据用户名精确查询用户
// @Summary 根据用户名精确查询用户
// @Description 根据用户名精确查询用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param username query string true "用户名"
// @Success 200 {object} dto.QueryUserResp "用户信息"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 404 {string} string "用户不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/users/query [get]
func (u *User) QueryUser(c *gin.Context) {
	var req dto.QueryUserReq
	var resp *dto.QueryUserResp
	var err error
	defer func() {
		logger.Debugf(c, "QueryUser req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 提取括号前的用户名部分（支持 "sina(新那)" 格式）
	username := extractUsernameFromDisplayName(req.Username)

	// 查询用户信息
	user, err := u.userService.GetUserByUsername(username)
	if err != nil {
		response.Internal(c, "用户不存在: "+err.Error())
		return
	}
	if err := u.ensureSameCompany(c, user); err != nil {
		response.Error(c, err)
		return
	}

	// 转换为DTO（包含详细信息）
	ctx := contextx.ToContext(c)
	userInfos := convertUsersToDTOBatch(ctx, []*model.User{user}, u.userService, u.departmentService)
	if len(userInfos) == 0 {
		response.Internal(c, "转换用户信息失败")
		return
	}
	resp = &dto.QueryUserResp{
		User: *userInfos[0],
	}
	response.OkWithData(c, resp)
}

// SearchUsersFuzzy 模糊查询用户
// @Summary 模糊查询用户
// @Description 根据关键词模糊查询用户（支持用户名、邮箱和昵称）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param keyword query string true "搜索关键词"
// @Param limit query int false "返回数量限制，默认10，最大100"
// @Success 200 {object} dto.SearchUsersFuzzyResp "用户列表"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/users/search [get]
func (u *User) SearchUsersFuzzy(c *gin.Context) {
	var req dto.SearchUsersFuzzyReq
	var resp *dto.SearchUsersFuzzyResp
	var err error
	defer func() {
		logger.Debugf(c, "SearchUsersFuzzy req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认limit
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// 提取括号前的关键词部分（支持 "sina(新那)" 格式）
	keyword := extractUsernameFromDisplayName(req.Keyword)

	// 查询用户列表
	requester := contextx.GetRequestUser(c)
	if requester == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}
	users, err := u.userService.SearchUsersFuzzyInRequesterCompany(requester, keyword, req.Limit)
	if err != nil {
		response.Internal(c, "查询失败: "+err.Error())
		return
	}

	// 转换为DTO（包含详细信息，批量查询）
	ctx := contextx.ToContext(c)
	dtoUserInfos := convertUsersToDTOBatch(ctx, users, u.userService, u.departmentService)
	userInfos := make([]dto.UserInfo, 0, len(dtoUserInfos))
	for _, userInfo := range dtoUserInfos {
		userInfos = append(userInfos, *userInfo)
	}

	resp = &dto.SearchUsersFuzzyResp{
		Users: userInfos,
	}
	response.OkWithData(c, resp)
}

// GetUsersByUsernames 批量获取用户信息
// @Summary 批量获取用户信息
// @Description 根据用户名列表批量获取用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param X-Token header string true "JWT Token"
// @Param request body dto.GetUsersByUsernamesReq true "批量查询请求"
// @Success 200 {object} dto.GetUsersByUsernamesResp "用户列表"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未认证"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/users [post]
func (u *User) GetUsersByUsernames(c *gin.Context) {
	var req dto.GetUsersByUsernamesReq
	var resp *dto.GetUsersByUsernamesResp
	var err error
	defer func() {
		logger.Debugf(c, "GetUsersByUsernames req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 查询用户列表
	requester := contextx.GetRequestUser(c)
	if requester == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}
	users, err := u.userService.GetUsersByUsernamesInRequesterCompany(requester, req.Usernames)
	if err != nil {
		response.Internal(c, "查询失败: "+err.Error())
		return
	}

	// 转换为DTO（包含详细信息，批量查询）
	ctx := contextx.ToContext(c)
	dtoUserInfos := convertUsersToDTOBatch(ctx, users, u.userService, u.departmentService)
	userInfos := make([]dto.UserInfo, 0, len(dtoUserInfos))
	for _, userInfo := range dtoUserInfos {
		userInfos = append(userInfos, *userInfo)
	}

	resp = &dto.GetUsersByUsernamesResp{
		Users: userInfos,
	}
	response.OkWithData(c, resp)
}
