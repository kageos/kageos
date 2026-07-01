package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	hrmodel "github.com/kageos/kageos/core/hr-server/model"
	hrrepository "github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// SystemUsername 系统用户名
	SystemUsername = "system"
	// SystemUserEmail 系统用户邮箱
	SystemUserEmail = "system@kageos.local"

	// TestUsername 测试用户名（用于执行/测试场景兜底）
	TestUsername = "test_user"
	// TestUserEmail 测试用户邮箱
	TestUserEmail = "test_user@kageos.local"
	// TestUserDepartmentPath 测试用户默认归属部门（虚拟组织/测试组）
	TestUserDepartmentPath = "/org/virtual/test"
)

// InitDefaultUsers 初始化默认用户：system + test_user（密码与 system 共用，test_user 归属 /org/virtual/test）
// 在 hr-server 启动时调用，且应在 InitDefaultDepartments 之后调用（保证虚拟组织/测试组已存在）
func InitDefaultUsers(ctx context.Context, db *gorm.DB) error {
	cfg := config.GetHRServerConfig()
	password, generated := getSystemUserPassword(cfg, ctx)

	if err := initSystemUserWithPassword(ctx, db, password, generated); err != nil {
		return err
	}
	if err := initTestUserWithPassword(ctx, db, password, generated); err != nil {
		return err
	}
	return nil
}

// InitSystemUser 初始化 system 用户（兼容旧调用；新逻辑请用 InitDefaultUsers）
func InitSystemUser(ctx context.Context, db *gorm.DB) error {
	cfg := config.GetHRServerConfig()
	password, generated := getSystemUserPassword(cfg, ctx)
	return initSystemUserWithPassword(ctx, db, password, generated)
}

// initSystemUserWithPassword 创建或更新 system 用户，使用给定密码
func initSystemUserWithPassword(ctx context.Context, db *gorm.DB, password string, generated bool) error {
	logger.Infof(ctx, "[SystemUser] 开始初始化 system 用户...")

	userRepo := hrrepository.NewUserRepository(db)

	existingUser, err := userRepo.GetUserByUsername(SystemUsername)
	if err == nil && existingUser != nil {
		if existingUser.Type != hrmodel.UserTypeSystem {
			existingUser.Type = hrmodel.UserTypeSystem
			if err := userRepo.UpdateUser(existingUser); err != nil {
				return fmt.Errorf("更新 system 用户类型失败: %w", err)
			}
			logger.Infof(ctx, "[SystemUser] 已更新 system 用户类型为系统用户")
		}
		if existingUser.PasswordHash == "" {
			if err := setSystemUserPassword(ctx, userRepo, existingUser, password, generated); err != nil {
				logger.Warnf(ctx, "[SystemUser] 设置系统账号密码失败: %v", err)
			}
		} else if !generated && bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(password)) != nil {
			if err := setSystemUserPassword(ctx, userRepo, existingUser, password, generated); err != nil {
				logger.Warnf(ctx, "[SystemUser] 同步系统账号密码失败: %v", err)
			}
		} else {
			logger.Infof(ctx, "[SystemUser] system 用户已存在，类型正确，密码已对齐")
		}
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	systemUser := &hrmodel.User{
		Username:      SystemUsername,
		Email:         SystemUserEmail,
		CompanyCode:   defaultCompanyCode(),
		PasswordHash:  string(hashedPassword),
		Status:        "active",
		EmailVerified: true,
		RegisterType:  "system",
		Type:          hrmodel.UserTypeSystem,
		CreatedBy:     "system",
		Nickname:      "系统",
		Signature:     "系统内置用户，用于管理系统工具、平台接口和提示词",
	}

	if err := userRepo.CreateUser(systemUser); err != nil {
		return fmt.Errorf("创建 system 用户失败: %w", err)
	}

	if generated {
		logger.Warnf(ctx, "[SystemUser] ⚠️  系统账号密码已自动生成，请妥善保管：")
		logger.Warnf(ctx, "[SystemUser] ⚠️  用户名: %s", SystemUsername)
		if shouldPrintGeneratedSecrets() {
			logger.Warnf(ctx, "[SystemUser] ⚠️  密码: %s", password)
		} else {
			logger.Warnf(ctx, "[SystemUser] ⚠️  密码已隐藏；如需启动日志打印，显式设置 KAGEOS_PRINT_GENERATED_SECRETS=1")
		}
		logger.Warnf(ctx, "[SystemUser] ⚠️  建议：在配置文件中设置 system_user.password 或环境变量 SYSTEM_USER_PASSWORD")
	} else {
		logger.Infof(ctx, "[SystemUser] 已创建 system 用户: %s（密码已从配置加载）", SystemUsername)
	}
	return nil
}

// initTestUserWithPassword 创建或更新 test_user，密码与 system 共用，归属 /org/virtual/test
func initTestUserWithPassword(ctx context.Context, db *gorm.DB, password string, generated bool) error {
	logger.Infof(ctx, "[TestUser] 开始初始化 test_user...")

	userRepo := hrrepository.NewUserRepository(db)

	existingUser, err := userRepo.GetUserByUsername(TestUsername)
	if err == nil && existingUser != nil {
		if existingUser.DepartmentFullPath != TestUserDepartmentPath {
			existingUser.DepartmentFullPath = TestUserDepartmentPath
			if err := userRepo.UpdateUser(existingUser); err != nil {
				logger.Warnf(ctx, "[TestUser] 更新 test_user 部门失败: %v", err)
			}
		}
		if existingUser.PasswordHash == "" {
			if err := setTestUserPassword(userRepo, existingUser, password); err != nil {
				logger.Warnf(ctx, "[TestUser] 设置 test_user 密码失败: %v", err)
			}
		} else if !generated && bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(password)) != nil {
			if err := setTestUserPassword(userRepo, existingUser, password); err != nil {
				logger.Warnf(ctx, "[TestUser] 同步 test_user 密码失败: %v", err)
			}
		}
		logger.Infof(ctx, "[TestUser] test_user 已存在，已确保归属 %s，密码已对齐", TestUserDepartmentPath)
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("test_user 密码加密失败: %w", err)
	}

	testUser := &hrmodel.User{
		Username:           TestUsername,
		Email:              TestUserEmail,
		CompanyCode:        defaultCompanyCode(),
		PasswordHash:       string(hashedPassword),
		Status:             "active",
		EmailVerified:      true,
		RegisterType:       "system",
		Type:               hrmodel.UserTypeNormal,
		CreatedBy:          SystemUsername,
		Nickname:           "测试用户",
		Signature:          "执行/测试场景兜底用户，归属虚拟组织/测试组",
		DepartmentFullPath: TestUserDepartmentPath,
	}

	if err := userRepo.CreateUser(testUser); err != nil {
		return fmt.Errorf("创建 test_user 失败: %w", err)
	}

	logger.Infof(ctx, "[TestUser] 已创建 test_user，归属 %s，密码与 system 共用", TestUserDepartmentPath)
	return nil
}

func setTestUserPassword(userRepo *hrrepository.UserRepository, user *hrmodel.User, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashed)
	return userRepo.UpdateUser(user)
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
		if shouldPrintGeneratedSecrets() {
			logger.Warnf(ctx, "[SystemUser] ⚠️  密码: %s", password)
		} else {
			logger.Warnf(ctx, "[SystemUser] ⚠️  密码已隐藏；如需启动日志打印，显式设置 KAGEOS_PRINT_GENERATED_SECRETS=1")
		}
		logger.Warnf(ctx, "[SystemUser] ⚠️  建议：在配置文件中设置 system_user.password 或环境变量 SYSTEM_USER_PASSWORD")
	} else {
		logger.Infof(ctx, "[SystemUser] 已为 system 用户设置密码（从配置加载）")
	}

	return nil
}

func shouldPrintGeneratedSecrets() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("KAGEOS_PRINT_GENERATED_SECRETS")))
	return value == "1" || value == "true" || value == "yes"
}
