package v1

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

// Auth 认证相关API
type Auth struct {
	authService       *service.AuthService
	emailService      *service.EmailService
	userService       *service.UserService
	departmentService *service.DepartmentService
}

// NewAuth 创建认证API（依赖注入）
func NewAuth(authService *service.AuthService, emailService *service.EmailService, userService *service.UserService, departmentService *service.DepartmentService) *Auth {
	return &Auth{
		authService:       authService,
		emailService:      emailService,
		userService:       userService,
		departmentService: departmentService,
	}
}

// SendEmailCode 发送邮箱验证码
// @Summary 发送邮箱验证码
// @Description 向指定邮箱发送验证码，用于注册验证
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.SendEmailCodeReq true "发送验证码请求"
// @Success 200 {object} dto.SendEmailCodeResp "发送成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/send_email_code [post]
func (a *Auth) SendEmailCode(c *gin.Context) {
	var req dto.SendEmailCodeReq
	var resp *dto.SendEmailCodeResp
	var err error
	defer func() {
		logger.Infof(c, "SendEmailCode req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 获取客户端信息
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 发送验证码（根据 codeType 参数决定发送类型，默认为 register）
	codeType := c.Query("type")
	if codeType == "" {
		codeType = "register"
	}

	debugCode, sendErr := a.emailService.SendVerificationCode(req.Email, codeType, ipAddress, userAgent)
	err = sendErr
	if err != nil {
		response.FailWithMessage(c, "发送验证码失败: "+err.Error())
		return
	}

	resp = &dto.SendEmailCodeResp{DebugCode: debugCode}
	response.OkWithDetailed(c, resp, "验证码已发送")
}

// Register 用户注册
// @Summary 用户注册
// @Description 使用邮箱验证码注册新用户
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.RegisterReq true "注册请求"
// @Success 200 {object} dto.RegisterResp "注册成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/register [post]
func (a *Auth) Register(c *gin.Context) {
	var req dto.RegisterReq
	var resp *dto.RegisterResp
	var err error
	defer func() {
		logger.Infof(c, "Register username=%s email=%s company_action=%s company_code=%s resp:%+v err:%v",
			req.Username, req.Email, req.CompanyAction, req.CompanyCode, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证验证码
	err = a.emailService.VerifyCode(req.Email, req.Code, "register")
	if err != nil {
		response.FailWithMessage(c, "验证码错误或已过期: "+err.Error())
		return
	}

	// 注册用户
	userID, err := a.authService.RegisterUser(req.Username, req.Email, req.Password, req.CompanyAction, req.CompanyCode, req.CompanyName, req.CompanyLogoURL)
	if err != nil {
		response.FailWithMessage(c, "注册失败: "+err.Error())
		return
	}

	// 激活用户（因为验证码已验证通过）
	err = a.authService.ActivateUser(userID)
	if err != nil {
		logger.Errorf(c, "[Auth] Failed to activate user %d: %v", userID, err)
		// 不返回错误，因为用户已创建成功
	}

	resp = &dto.RegisterResp{
		UserID: userID,
	}

	response.OkWithData(c, resp)
}

// SearchCompanies 模糊搜索企业
// @Summary 模糊搜索企业
// @Description 注册加入企业时，根据企业名称或企业代码搜索可加入企业
// @Tags 认证管理
// @Produce json
// @Param keyword query string false "搜索关键字"
// @Param limit query int false "返回数量"
// @Success 200 {object} dto.SearchCompaniesResp "搜索成功"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/companies/search [get]
func (a *Auth) SearchCompanies(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	limit := 10
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		if parsedLimit, parseErr := strconv.Atoi(rawLimit); parseErr == nil {
			limit = parsedLimit
		}
	}

	companies, err := a.authService.SearchCompaniesFuzzy(keyword, limit)
	if err != nil {
		response.FailWithMessage(c, "搜索企业失败: "+err.Error())
		return
	}

	options := make([]dto.CompanyOption, 0, len(companies))
	for _, company := range companies {
		options = append(options, dto.CompanyOption{
			Code:    company.Code,
			Name:    company.Name,
			LogoURL: company.LogoURL,
		})
	}
	response.OkWithData(c, &dto.SearchCompaniesResp{Companies: options})
}

// CreateUserBySecret 超管一键创建用户（免邮箱验证，仅 system 用户可操作，用于创建测试用户）
// @Summary 一键创建用户（仅 system 超管）
// @Description 仅已登录的 system 用户可调用，直接创建用户无需邮箱验证。
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.CreateUserBySecretReq true "创建请求"
// @Success 200 {object} dto.CreateUserBySecretResp "创建成功"
// @Failure 403 {string} string "仅 system 用户可操作"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/user/create_user_by_secret [post]
func (a *Auth) CreateUserBySecret(c *gin.Context) {
	currentUser := contextx.GetRequestUser(c)
	if currentUser != "system" {
		response.FailWithMessage(c, "仅 system 超管可操作")
		c.Abort()
		return
	}

	var req dto.CreateUserBySecretReq
	var resp *dto.CreateUserBySecretResp
	var err error
	defer func() {
		logger.Infof(c, "CreateUserBySecret req:{Username:%s} resp:%+v err:%v", req.Username, resp, err)
	}()

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	userID, err := a.authService.CreateUserBySecretKey(req.Username, req.Password)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.CreateUserBySecretResp{UserID: userID}
	response.OkWithData(c, resp)
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名和密码登录
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.LoginReq true "登录请求"
// @Success 200 {object} dto.LoginResp "登录成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "用户名或密码错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/login [post]
func (a *Auth) Login(c *gin.Context) {
	var req dto.LoginReq
	var resp *dto.LoginResp
	var err error
	defer func() {
		logger.Infof(c, "Login username=%s remember=%v err:%v", req.Username, req.Remember, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 登录用户
	user, token, refreshToken, err := a.authService.LoginUser(req.Username, req.Password, req.Remember)
	if err != nil {
		response.FailWithMessage(c, "登录失败: "+err.Error())
		return
	}

	// ⭐ 转换为DTO（包含详细信息：组织架构和Leader信息）
	ctx := contextx.ToContext(c)
	userInfos := convertUsersToDTOBatch(ctx, []*model.User{user}, a.userService, a.departmentService)
	if len(userInfos) == 0 {
		response.FailWithMessage(c, "转换用户信息失败")
		return
	}

	resp = &dto.LoginResp{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *userInfos[0],
	}

	response.OkWithData(c, resp)
}

// RefreshToken 刷新Token
// @Summary 刷新Token
// @Description 使用RefreshToken刷新JWT Token
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenReq true "刷新Token请求"
// @Success 200 {object} dto.RefreshTokenResp "刷新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "RefreshToken无效"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/refresh [post]
func (a *Auth) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenReq
	var resp *dto.RefreshTokenResp
	var err error
	defer func() {
		logger.Infof(c, "RefreshToken req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 刷新Token
	newAccessToken, newRefreshToken, err := a.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		response.FailWithMessage(c, "刷新Token失败: "+err.Error())
		return
	}

	resp = &dto.RefreshTokenResp{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
	}

	response.OkWithData(c, resp)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 用户登出，使Token失效
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.LogoutReq true "登出请求"
// @Success 200 {object} dto.LogoutResp "登出成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/logout [post]
func (a *Auth) Logout(c *gin.Context) {
	var req dto.LogoutReq
	var resp *dto.LogoutResp
	var err error
	defer func() {
		logger.Infof(c, "Logout req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 登出用户
	err = a.authService.LogoutUser(req.Token)
	if err != nil {
		response.FailWithMessage(c, "登出失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "登出成功")
}

// ForgotPassword 忘记密码（简化版：直接通过验证码重置密码）
// @Summary 忘记密码
// @Description 验证邮箱和验证码，直接重置密码
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordReq true "忘记密码请求"
// @Success 200 {object} dto.ForgotPasswordResp "重置成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /hr/api/v1/auth/forgot_password [post]
func (a *Auth) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordReq
	var resp *dto.ForgotPasswordResp
	var err error
	defer func() {
		logger.Infof(c, "ForgotPassword req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证验证码（使用 "forgot_password" 作为 codeType）
	err = a.emailService.VerifyCode(req.Email, req.Code, "forgot_password")
	if err != nil {
		response.FailWithMessage(c, "验证码错误或已过期: "+err.Error())
		return
	}

	// 直接重置密码（验证码已验证，用户存在性在 ResetPasswordByEmail 中检查）
	err = a.authService.ResetPasswordByEmail(req.Email, req.Password)
	if err != nil {
		response.FailWithMessage(c, "重置密码失败: "+err.Error())
		return
	}

	resp = &dto.ForgotPasswordResp{}
	response.OkWithMessage(c, "密码重置成功")
}
