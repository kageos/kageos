package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	hrmodel "github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	hrrepository "github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// SystemUsername 系统用户名
	SystemUsername = "system"
	// SystemUserEmail 系统用户邮箱
	SystemUserEmail = "system@ai-agent-os.local"
)

// InitSystemUser 初始化 system 用户
// 在 hr-server 启动时调用，确保 system 用户存在
func InitSystemUser(ctx context.Context, db *gorm.DB) error {
	logger.Infof(ctx, "[SystemUser] 开始初始化 system 用户...")

	userRepo := hrrepository.NewUserRepository(db)
	cfg := config.GetHRServerConfig()

	// 检查 system 用户是否已存在
	existingUser, err := userRepo.GetUserByUsername(SystemUsername)
	if err == nil && existingUser != nil {
		// 已存在，检查类型是否正确
		if existingUser.Type != hrmodel.UserTypeSystem {
			// 更新类型为系统用户
			existingUser.Type = hrmodel.UserTypeSystem
			if err := userRepo.UpdateUser(existingUser); err != nil {
				return fmt.Errorf("更新 system 用户类型失败: %w", err)
			}
			logger.Infof(ctx, "[SystemUser] 已更新 system 用户类型为系统用户")
		}
		
		// ⭐ 如果系统账号没有密码，尝试设置密码
		if existingUser.PasswordHash == "" {
			password, generated := getSystemUserPassword(cfg, ctx)
			if err := setSystemUserPassword(ctx, userRepo, existingUser, password, generated); err != nil {
				logger.Warnf(ctx, "[SystemUser] 设置系统账号密码失败: %v", err)
				// 不中断启动，记录警告即可
			}
		} else {
			logger.Infof(ctx, "[SystemUser] system 用户已存在，类型正确，已有密码")
		}
		return nil
	}

	// 不存在，创建 system 用户
	password, generated := getSystemUserPassword(cfg, ctx)
	
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	systemUser := &hrmodel.User{
		Username:      SystemUsername,
		Email:         SystemUserEmail,
		PasswordHash:  string(hashedPassword), // ⭐ 设置密码
		Status:        "active",              // 系统用户默认激活
		EmailVerified: true,                   // 系统用户默认已验证
		RegisterType:  "system",              // 注册类型为 system
		Type:          hrmodel.UserTypeSystem,
		CreatedBy:     "system",
		Nickname:      "系统",
		Signature:     "系统内置用户，用于管理官方库",
	}

	if err := userRepo.CreateUser(systemUser); err != nil {
		return fmt.Errorf("创建 system 用户失败: %w", err)
	}

	// ⭐ 如果密码是生成的，输出到日志
	if generated {
		logger.Warnf(ctx, "[SystemUser] ⚠️  系统账号密码已自动生成，请妥善保管：")
		logger.Warnf(ctx, "[SystemUser] ⚠️  用户名: %s", SystemUsername)
		logger.Warnf(ctx, "[SystemUser] ⚠️  密码: %s", password)
		logger.Warnf(ctx, "[SystemUser] ⚠️  建议：在配置文件中设置 system_user.password 或环境变量 SYSTEM_USER_PASSWORD")
	} else {
		logger.Infof(ctx, "[SystemUser] 已创建 system 用户: %s（密码已从配置加载）", SystemUsername)
	}
	
	return nil
}

// getSystemUserPassword 获取系统账号密码（优先从配置/环境变量，否则生成随机密码）
func getSystemUserPassword(cfg *config.HRServerConfig, ctx context.Context) (string, bool) {
	// 优先从配置或环境变量获取
	if password := cfg.GetSystemUserPassword(); password != "" {
		return password, false
	}
	
	// 生成随机密码（16字节，base64编码后约24字符）
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		logger.Warnf(ctx, "[SystemUser] 生成随机密码失败，使用默认密码: %v", err)
		return "System@123456", true // 默认密码（不安全，仅用于开发）
	}
	
	password := base64.URLEncoding.EncodeToString(randomBytes)
	return password, true
}

// setSystemUserPassword 设置系统账号密码
func setSystemUserPassword(ctx context.Context, userRepo *hrrepository.UserRepository, user *hrmodel.User, password string, generated bool) error {
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 更新密码
	user.PasswordHash = string(hashedPassword)
	if err := userRepo.UpdateUser(user); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	// 如果密码是生成的，输出到日志
	if generated {
		logger.Warnf(ctx, "[SystemUser] ⚠️  系统账号密码已自动生成，请妥善保管：")
		logger.Warnf(ctx, "[SystemUser] ⚠️  用户名: %s", SystemUsername)
		logger.Warnf(ctx, "[SystemUser] ⚠️  密码: %s", password)
		logger.Warnf(ctx, "[SystemUser] ⚠️  建议：在配置文件中设置 system_user.password 或环境变量 SYSTEM_USER_PASSWORD")
	} else {
		logger.Infof(ctx, "[SystemUser] 已为 system 用户设置密码（从配置加载）")
	}

	return nil
}
