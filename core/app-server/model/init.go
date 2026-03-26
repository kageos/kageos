package model

import (
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	// 先迁移外键父表，再迁移子表，避免外键约束错误
	err := db.AutoMigrate(
		&Nats{},
		&Host{},
		&App{},
		&ServiceTree{},
		&Function{},
		&Package{},
		// 新增的认证相关表
		&EmailVerification{},
		&EmailCode{},
		&UserSession{},
		// 目录快照表（用于递归 Fork）
		&FileSnapshot{},
		// 操作日志表（企业版功能，但社区版也记录）
		&TableOperateLog{},
		&FormOperateLog{},
		// 目录更新历史表（用于记录API变更历史）
		&DirectoryUpdateHistory{},
		// 文档表（用于存储文档内容）
		&Docs{},
		// 版块帖子表（讨论区下的帖子，评论回复后续统一做）
		&BoardPost{},
		// 权限系统相关表
		&PermissionRequest{},
		&PermissionGrantLog{},
		&ApprovalPolicy{},
		// 角色系统相关表（企业版功能）
		&Role{},
		&RolePermission{},
		&RoleAssignment{},
		// 权限点表（企业版功能）
		&Action{},
		// 定时任务
		&ScheduledTask{},
		&ScheduledTaskExecution{},
	)
	if err != nil {
		return err
	}

	// 创建默认的NATS和Host记录
	return initDefaultData(db)
}

// initDefaultData 初始化默认数据
func initDefaultData(db *gorm.DB) error {
	// 检查是否已有NATS记录
	var natsCount int64
	if err := db.Model(&Nats{}).Count(&natsCount).Error; err != nil {
		return err
	}

	// 如果没有NATS记录，创建默认记录（Compose 等场景设 NATS_SEED_HOST=nats，见 ReconcileNatsHostFromEnv）
	if natsCount == 0 {
		seedHost := strings.TrimSpace(os.Getenv("NATS_SEED_HOST"))
		if seedHost == "" {
			seedHost = "localhost"
		}
		seedPort := 4222
		if p := strings.TrimSpace(os.Getenv("NATS_SEED_PORT")); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				seedPort = v
			}
		}
		defaultNats := &Nats{
			Host: seedHost,
			Port: seedPort,
		}
		if err := db.Create(defaultNats).Error; err != nil {
			return err
		}
	}

	// 检查是否已有Host记录
	var hostCount int64
	if err := db.Model(&Host{}).Count(&hostCount).Error; err != nil {
		return err
	}

	// 如果没有Host记录，创建默认记录
	if hostCount == 0 {
		// 获取第一个NATS记录
		var nats Nats
		if err := db.First(&nats).Error; err != nil {
			return err
		}

		defaultHost := &Host{
			Domain:   "localhost",
			NatsID:   nats.ID,
			Status:   "enabled",
			AppCount: 0,
		}
		if err := db.Create(defaultHost).Error; err != nil {
			return err
		}
	}

	return nil
}

// ReconcileNatsHostFromEnv 在 app-server 连接 DB 中的 NATS 地址之前调用：若设置了 NATS_SEED_HOST，
// 将仍为 localhost / 127.0.0.1 的 nats 记录更新为环境变量中的主机（可选 NATS_SEED_PORT 同步改端口）。
// 用于修复历史默认种子在 Docker Compose 下无法连上独立 nats 容器的问题。
func ReconcileNatsHostFromEnv(db *gorm.DB) error {
	h := strings.TrimSpace(os.Getenv("NATS_SEED_HOST"))
	if h == "" {
		return nil
	}
	updates := map[string]interface{}{"host": h}
	if p := strings.TrimSpace(os.Getenv("NATS_SEED_PORT")); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			updates["port"] = v
		}
	}
	return db.Model(&Nats{}).Where("host IN ?", []string{"localhost", "127.0.0.1"}).Updates(updates).Error
}
