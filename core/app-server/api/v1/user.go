package v1

import (
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// User 用户相关API
type User struct {
	userService *service.UserService
}

// NewUser 创建用户API（依赖注入）
func NewUser(userService *service.UserService) *User {
	return &User{
		userService: userService,
	}
}

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
// @Router /api/v1/user/info [get]
func (u *User) GetUserInfo(c *gin.Context) {
	var resp *dto.UserInfo
	var err error
	defer func() {
		logger.Infof(c, "GetUserInfo resp:%+v err:%v", resp, err)
	}()

	// 从context获取username（JWTAuth中间件已从header获取并设置到context）
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "未提供用户信息")
		return
	}

	// 查询用户信息
	user, err := u.userService.GetUserByUsername(username)
	if err != nil {
		response.FailWithMessage(c, "用户不存在: "+err.Error())
		return
	}

	// 转换为DTO
	userInfo := convertUserToDTO(user)
	resp = userInfo
	response.OkWithData(c, resp)
}

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
// @Router /api/v1/user/query [get]
func (u *User) QueryUser(c *gin.Context) {
	var req dto.QueryUserReq
	var resp *dto.QueryUserResp
	var err error
	defer func() {
		logger.Infof(c, "QueryUser req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 提取括号前的用户名部分（支持 "sina(新那)" 格式）
	username := extractUsernameFromDisplayName(req.Username)

	// 查询用户信息
	user, err := u.userService.GetUserByUsername(username)
	if err != nil {
		response.FailWithMessage(c, "用户不存在: "+err.Error())
		return
	}

	// 转换为DTO
	userInfo := convertUserToDTO(user)
	resp = &dto.QueryUserResp{
		User: *userInfo,
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
// @Router /api/v1/user/search_fuzzy [get]
func (u *User) SearchUsersFuzzy(c *gin.Context) {
	var req dto.SearchUsersFuzzyReq
	var resp *dto.SearchUsersFuzzyResp
	var err error
	defer func() {
		logger.Infof(c, "SearchUsersFuzzy req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 设置默认limit
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// 提取括号前的关键词部分（支持 "sina(新那)" 格式）
	keyword := extractUsernameFromDisplayName(req.Keyword)

	// 查询用户列表
	users, err := u.userService.SearchUsersFuzzy(keyword, req.Limit)
	if err != nil {
		response.FailWithMessage(c, "查询失败: "+err.Error())
		return
	}

	// 转换为DTO
	userInfos := make([]dto.UserInfo, 0, len(users))
	for _, user := range users {
		userInfo := convertUserToDTO(user)
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
// @Router /api/v1/users [post]
func (u *User) GetUsersByUsernames(c *gin.Context) {
	var req dto.GetUsersByUsernamesReq
	var resp *dto.GetUsersByUsernamesResp
	var err error
	defer func() {
		logger.Infof(c, "GetUsersByUsernames req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 查询用户列表
	users, err := u.userService.GetUsersByUsernames(req.Usernames)
	if err != nil {
		response.FailWithMessage(c, "查询失败: "+err.Error())
		return
	}

	// 转换为DTO
	userInfos := make([]dto.UserInfo, 0, len(users))
	for _, user := range users {
		userInfo := convertUserToDTO(user)
		userInfos = append(userInfos, *userInfo)
	}

	resp = &dto.GetUsersByUsernamesResp{
		Users: userInfos,
	}
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
// @Router /api/v1/user/update [put]
func (u *User) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserReq
	var resp *dto.UpdateUserResp
	var err error
	defer func() {
		logger.Infof(c, "UpdateUser req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 从context获取username
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "未提供用户信息")
		return
	}

	// 检查是否有字段需要更新
	if req.Nickname == nil && req.Signature == nil && req.Avatar == nil && req.Gender == nil {
		response.FailWithMessage(c, "至少需要提供一个更新字段")
		return
	}

	// 更新用户信息（直接传递指针，nil 表示不更新，非 nil 表示更新）
	user, err := u.userService.UpdateUser(username, req.Nickname, req.Signature, req.Avatar, req.Gender)
	if err != nil {
		response.FailWithMessage(c, "更新失败: "+err.Error())
		return
	}

	// 转换为DTO
	userInfo := convertUserToDTO(user)
	resp = &dto.UpdateUserResp{
		User: *userInfo,
	}
	response.OkWithData(c, resp)
}

// extractUsernameFromDisplayName 从显示名称中提取用户名
// 原因：前端可能传递的是显示名称格式（如 "sina(新那)"），其中括号前是用户名，括号内是昵称
// 为了支持用户通过前端展示的名称进行查询，需要提取括号前的用户名部分进行实际查询
// 示例：
//   - "sina(新那)" -> "sina"
//   - "sina" -> "sina"（没有括号时返回原字符串）
func extractUsernameFromDisplayName(displayName string) string {
	return strings.TrimSpace(strings.Split(displayName, "(")[0])
}

// convertUserToDTO 将model.User转换为dto.UserInfo
func convertUserToDTO(user *model.User) *dto.UserInfo {
	return &dto.UserInfo{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		RegisterType:  user.RegisterType,
		Avatar:        user.Avatar,
		Nickname:      user.Nickname,
		Signature:     user.Signature,
		Gender:        user.Gender,
		EmailVerified: user.EmailVerified,
		Status:        user.Status,
		CreatedAt:     time.Time(user.CreatedAt).Format(time.RFC3339),
	}
}
