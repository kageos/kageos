package v1

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Auth 认证相关API
type Auth struct {
	authService  *service.AuthService
	emailService *service.EmailService
}

// NewAuth 创建认证API（依赖注入）
func NewAuth(authService *service.AuthService, emailService *service.EmailService) *Auth {
	return &Auth{
		authService:  authService,
		emailService: emailService,
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
	
	err = a.emailService.SendVerificationCode(req.Email, codeType, ipAddress, userAgent)
	if err != nil {
		response.FailWithMessage(c, "发送验证码失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "验证码已发送")
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
		logger.Infof(c, "Register req:%+v resp:%+v err:%v", req, resp, err)
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
	userID, err := a.authService.RegisterUser(req.Username, req.Email, req.Password)
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
		logger.Infof(c, "Login req:%+v resp:%+v err:%v", req, resp, err)
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

	resp = &dto.LoginResp{
		Token:        token,
		RefreshToken: refreshToken,
		User: dto.UserInfo{
			ID:            user.ID,
			Username:      user.Username,
			Email:         user.Email,
			RegisterType:  user.RegisterType,
			Avatar:        user.Avatar,
			EmailVerified: user.EmailVerified,
			Status:        user.Status,
			CreatedAt:     time.Time(user.CreatedAt).Format(time.RFC3339),
		},
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
