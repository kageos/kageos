package model

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	kageconfig "github.com/kageos/kageos/pkg/config"
	"gorm.io/gorm"
)

// isAppEnvDev 与 pkg/config GetConfigEnv 一致：优先读取 .kageos/kageos.env。
func isAppEnvDev() bool {
	return kageconfig.IsDevMode()
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
	if err := dropLegacyFunctionNaturalKeyIndex(db); err != nil {
		return err
	}

	// 先迁移外键父表，再迁移子表，避免外键约束错误
	err := db.AutoMigrate(
		&Nats{},
		&Host{},
		&App{},
		&PersonalWorkspaceBootstrap{},
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

	if err := ensureOperateLogQueryIndexes(db); err != nil {
		return err
	}

	// 创建默认的NATS和Host记录
	return initDefaultData(db)
}

func dropLegacyFunctionNaturalKeyIndex(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&Function{}) {
		return nil
	}
	const legacyIndexName = "idx_function_app_method_router"
	if !db.Migrator().HasIndex(&Function{}, legacyIndexName) {
		return nil
	}
	if err := db.Migrator().DropIndex(&Function{}, legacyIndexName); err != nil {
		return fmt.Errorf("failed to drop legacy function natural key index: %w", err)
	}
	return nil
}

type operateLogIndexSpec struct {
	name    string
	columns []string
}

func ensureOperateLogQueryIndexes(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&OperateLog{}) {
		return nil
	}

	indexes := []operateLogIndexSpec{
		{name: "idx_oplog_path_created", columns: []string{"resource_path", "created_at", "id"}},
		{name: "idx_oplog_path_target_created", columns: []string{"resource_path", "target_id", "created_at", "id"}},
		{name: "idx_oplog_path_action_created", columns: []string{"resource_path", "action", "created_at", "id"}},
		{name: "idx_oplog_path_source_created", columns: []string{"resource_path", "source", "created_at", "id"}},
		{name: "idx_oplog_path_executor_created", columns: []string{"resource_path", "executor_type", "created_at", "id"}},
		{name: "idx_oplog_path_actor_created", columns: []string{"resource_path", "actor_user", "created_at", "id"}},
		{name: "idx_oplog_source_ref_created", columns: []string{"source_type", "source_ref", "created_at", "id"}},
		{name: "idx_oplog_workspace_session_created", columns: []string{"workspace_session_id", "created_at", "id"}},
		{name: "idx_oplog_trace_created", columns: []string{"trace_id", "created_at", "id"}},
	}

	for _, index := range indexes {
		if db.Migrator().HasIndex(&OperateLog{}, index.name) {
			continue
		}
		if err := db.Exec(buildOperateLogCreateIndexSQL(db.Dialector.Name(), index)).Error; err != nil {
			return fmt.Errorf("failed to create operate log index %s: %w", index.name, err)
		}
	}
	return nil
}

func buildOperateLogCreateIndexSQL(dialect string, index operateLogIndexSpec) string {
	quote := operateLogIndexQuote(dialect)
	columns := make([]string, 0, len(index.columns))
	for _, column := range index.columns {
		columns = append(columns, quote(column))
	}
	return fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quote(index.name), quote("operate_logs"), strings.Join(columns, ", "))
}

func operateLogIndexQuote(dialect string) func(string) string {
	switch strings.ToLower(dialect) {
	case "postgres":
		return func(identifier string) string {
			return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
		}
	case "mysql", "sqlite":
		return func(identifier string) string {
			return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
		}
	default:
		return func(identifier string) string {
			return identifier
		}
	}
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
