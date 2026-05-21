package model

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// isAppEnvDev 与 pkg/config getConfigEnv 一致：仅 APP_ENV=dev 为开发，其余（含未设）视为生产/交付环境。
func isAppEnvDev() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) == "dev"
}

// defaultNatsSeedEndpoint 无 NATS_SEED_HOST 时：开发默认 localhost，交付/线上默认 127.0.0.1（main 容器 host 网络）。
func defaultNatsSeedEndpoint() (host string, port int) {
	port = 4222
	if p := strings.TrimSpace(os.Getenv("NATS_SEED_PORT")); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			port = v
		}
	}
	if h := strings.TrimSpace(os.Getenv("NATS_SEED_HOST")); h != "" {
		return h, port
	}
	if isAppEnvDev() {
		return "localhost", port
	}
	return "127.0.0.1", port
}

func defaultNatsSeedCredentials() (user string, password string) {
	return strings.TrimSpace(os.Getenv("NATS_SEED_USER")), strings.TrimSpace(os.Getenv("NATS_SEED_PASSWORD"))
}

func localNatsHosts() []string {
	return []string{"localhost", "127.0.0.1", "nats"}
}

func InitTables(db *gorm.DB) error {
	// 先迁移外键父表，再迁移子表，避免外键约束错误
	err := db.AutoMigrate(
		&Nats{},
		&Host{},
		&App{},
		&ServiceTree{},
		&Function{},
		&FunctionSensitiveField{},
		&Package{},
		// 目录快照表（用于递归 Fork）
		&FileSnapshot{},
		// 平台级操作审计日志
		&OperateLog{},
		// 轻量团队授权表
		&WorkspaceRoleAssignment{},
		// 目录更新历史表（用于记录API变更历史）
		&DirectoryUpdateHistory{},
		// 文档表（用于存储文档内容）
		&Docs{},
		// 公开分享链接（MVP: Form 匿名提交）
		&PublicShare{},
		&PublicShareEvent{},
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

	// 如果没有 NATS 记录：显式 NATS_SEED_HOST 优先；否则 dev→localhost，其它→127.0.0.1（host 网络直连）
	if natsCount == 0 {
		seedHost, seedPort := defaultNatsSeedEndpoint()
		seedUser, seedPassword := defaultNatsSeedCredentials()
		defaultNats := &Nats{
			Host:     seedHost,
			Port:     seedPort,
			User:     seedUser,
			Password: seedPassword,
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

// ReconcileNatsHostFromEnv 在 app-server 按 DB 连接 NATS 之前调用：
//   - 若设置了 NATS_SEED_HOST：将历史遗留的 localhost/127.0.0.1/nats 更新为该主机；
//   - 若设置了 NATS_SEED_USER / NATS_SEED_PASSWORD：同步认证字段；
//   - 若未设置且非 dev：将 nats/localhost 更新为 127.0.0.1（main 容器 host 网络直连）；
//   - dev 且未显式 NATS_SEED_HOST：不改动（保留本机 NATS）。
func ReconcileNatsHostFromEnv(db *gorm.DB) error {
	explicitHost := strings.TrimSpace(os.Getenv("NATS_SEED_HOST"))
	explicitUser := strings.TrimSpace(os.Getenv("NATS_SEED_USER"))
	explicitPassword := strings.TrimSpace(os.Getenv("NATS_SEED_PASSWORD"))
	updates := map[string]interface{}{}

	if explicitHost != "" {
		updates["host"] = explicitHost
		if p := strings.TrimSpace(os.Getenv("NATS_SEED_PORT")); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				updates["port"] = v
			}
		}
	} else {
		if isAppEnvDev() {
			if explicitUser == "" && explicitPassword == "" {
				return nil
			}
		} else {
			updates["host"] = "127.0.0.1"
		}
	}
	if explicitUser != "" {
		updates["user"] = explicitUser
	}
	if explicitPassword != "" {
		updates["password"] = explicitPassword
	}
	if len(updates) == 0 {
		return nil
	}

	return db.Model(&Nats{}).Where("host IN ?", localNatsHosts()).Updates(updates).Error
}

// ReconcileNatsHostFromURL keeps the DB-backed NATS connection pool aligned with global.yaml.
func ReconcileNatsHostFromURL(db *gorm.DB, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nil
	}

	port := 4222
	if p := strings.TrimSpace(parsed.Port()); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			return err
		}
		if v <= 0 {
			return fmt.Errorf("invalid nats port %q", p)
		}
		port = v
	}

	user := parsed.User.Username()
	password, _ := parsed.User.Password()
	updates := map[string]interface{}{
		"host":     host,
		"port":     port,
		"user":     user,
		"password": password,
	}

	hosts := append(localNatsHosts(), net.JoinHostPort(host, strconv.Itoa(port)), host)
	return db.Model(&Nats{}).Where("host IN ?", hosts).Updates(updates).Error
}
