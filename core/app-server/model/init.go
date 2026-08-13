package model

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/access"
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
	if err := removeRetiredOperateLogCompanyColumn(db); err != nil {
		return err
	}
	if err := migrateLegacyPermissionPrincipals(db); err != nil {
		return err
	}
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
		// 操作日志离线归档批次摘要
		&LogArchiveBatch{},
		// 轻量团队授权表
		&WorkspaceRoleAssignment{},
		// 权限申请与审批状态；实际权限仍落在 WorkspaceRoleAssignment
		&WorkspacePermissionRequest{},
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
	if err := backfillPermissionAssignmentKeys(db); err != nil {
		return err
	}

	if err := ensureOperateLogQueryIndexes(db); err != nil {
		return err
	}

	// 创建默认的NATS和Host记录
	return initDefaultData(db)
}

// migrateLegacyPermissionPrincipals upgrades username-only role assignments to
// the generic user/department principal model without losing existing grants.
func migrateLegacyPermissionPrincipals(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&legacyPermissionAssignment{}) {
		return nil
	}
	migrator := db.Migrator()
	if !migrator.HasColumn(&legacyPermissionAssignment{}, "Username") {
		return nil
	}
	if !migrator.HasColumn(&WorkspaceRoleAssignment{}, "PrincipalType") {
		if err := migrator.AddColumn(&WorkspaceRoleAssignment{}, "PrincipalType"); err != nil {
			return fmt.Errorf("add permission principal type: %w", err)
		}
	}
	if !migrator.HasColumn(&WorkspaceRoleAssignment{}, "PrincipalKey") {
		if err := migrator.AddColumn(&WorkspaceRoleAssignment{}, "PrincipalKey"); err != nil {
			return fmt.Errorf("add permission principal key: %w", err)
		}
	}
	if err := db.Table((WorkspaceRoleAssignment{}).TableName()).
		Where("principal_key = '' OR principal_key IS NULL").
		Updates(map[string]any{
			"principal_type": "user",
			"principal_key":  gorm.Expr("username"),
		}).Error; err != nil {
		return fmt.Errorf("backfill permission principals: %w", err)
	}
	for _, indexName := range []string{"idx_team_access_scope", "idx_team_access_user"} {
		if migrator.HasIndex(&legacyPermissionAssignment{}, indexName) {
			if err := migrator.DropIndex(&legacyPermissionAssignment{}, indexName); err != nil {
				return fmt.Errorf("drop retired permission index %s: %w", indexName, err)
			}
		}
	}
	var dropErr error
	if db.Dialector.Name() == "sqlite" {
		dropErr = db.Exec("ALTER TABLE `workspace_role_assignments` DROP COLUMN `username`").Error
	} else {
		dropErr = migrator.DropColumn(&legacyPermissionAssignment{}, "Username")
	}
	if dropErr != nil {
		return fmt.Errorf("drop retired permission username column: %w", dropErr)
	}
	return nil
}

// backfillPermissionAssignmentKeys makes grants idempotent without placing a
// very wide composite unique index on organization and resource paths. Legacy
// duplicate rows are collapsed while preserving the longest active grant.
func backfillPermissionAssignmentKeys(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&WorkspaceRoleAssignment{}) ||
		!db.Migrator().HasColumn(&WorkspaceRoleAssignment{}, "AssignmentKey") {
		return nil
	}

	var missing int64
	if err := db.Unscoped().Model(&WorkspaceRoleAssignment{}).
		Where("assignment_key IS NULL OR assignment_key = ''").
		Count(&missing).Error; err != nil {
		return fmt.Errorf("count permission assignment keys: %w", err)
	}
	if missing == 0 {
		return nil
	}

	var assignments []*WorkspaceRoleAssignment
	if err := db.Unscoped().Order("id ASC").Find(&assignments).Error; err != nil {
		return fmt.Errorf("load permission assignments for key migration: %w", err)
	}
	groups := make(map[string][]*WorkspaceRoleAssignment, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		key := access.PermissionAssignmentKey(
			assignment.TenantUser,
			assignment.App,
			access.Principal{
				Type: access.PrincipalType(assignment.PrincipalType),
				Key:  assignment.PrincipalKey,
			},
			assignment.ResourcePath,
			access.RoleCode(assignment.RoleCode),
		)
		groups[key] = append(groups[key], assignment)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Model(&WorkspaceRoleAssignment{}).
			Where("id > 0").
			Update("assignment_key", nil).Error; err != nil {
			return fmt.Errorf("reset permission assignment keys: %w", err)
		}
		for key, group := range groups {
			if len(group) == 0 {
				continue
			}
			winner := group[0]
			for _, candidate := range group[1:] {
				if preferPermissionAssignment(candidate, winner) {
					winner = candidate
				}
			}
			duplicateIDs := make([]int64, 0, len(group)-1)
			for _, assignment := range group {
				if assignment.ID != winner.ID {
					duplicateIDs = append(duplicateIDs, assignment.ID)
				}
			}
			if len(duplicateIDs) > 0 {
				if err := tx.Unscoped().Where("id IN ?", duplicateIDs).
					Delete(&WorkspaceRoleAssignment{}).Error; err != nil {
					return fmt.Errorf("remove duplicate permission assignments: %w", err)
				}
			}
			if err := tx.Unscoped().Model(&WorkspaceRoleAssignment{}).
				Where("id = ?", winner.ID).
				Update("assignment_key", key).Error; err != nil {
				return fmt.Errorf("backfill permission assignment key: %w", err)
			}
		}
		return nil
	})
}

func preferPermissionAssignment(candidate, current *WorkspaceRoleAssignment) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateActive := !candidate.DeletedAt.Valid
	currentActive := !current.DeletedAt.Valid
	if candidateActive != currentActive {
		return candidateActive
	}
	if candidateActive {
		if (candidate.ExpiresAt == nil) != (current.ExpiresAt == nil) {
			return candidate.ExpiresAt == nil
		}
		if candidate.ExpiresAt != nil && current.ExpiresAt != nil && !candidate.ExpiresAt.Equal(*current.ExpiresAt) {
			return candidate.ExpiresAt.After(*current.ExpiresAt)
		}
	}
	if candidate.UpdatedAt.GetUnix() != current.UpdatedAt.GetUnix() {
		return candidate.UpdatedAt.GetUnix() > current.UpdatedAt.GetUnix()
	}
	return candidate.ID > current.ID
}

type legacyPermissionAssignment struct {
	WorkspaceRoleAssignment `gorm:"embedded"`
	Username                string `gorm:"column:username"`
}

func (legacyPermissionAssignment) TableName() string {
	return "workspace_role_assignments"
}

func removeRetiredOperateLogCompanyColumn(db *gorm.DB) error {
	legacyColumn := &retiredOperateLogCompanyColumn{}
	if db == nil || !db.Migrator().HasTable(legacyColumn) || !db.Migrator().HasColumn(legacyColumn, "CompanyCode") {
		return nil
	}
	if err := db.Migrator().DropColumn(legacyColumn, "CompanyCode"); err != nil {
		return fmt.Errorf("drop retired operate log company column: %w", err)
	}
	return nil
}

type retiredOperateLogCompanyColumn struct {
	CompanyCode string `gorm:"column:company_code"`
}

func (retiredOperateLogCompanyColumn) TableName() string {
	return "operate_logs"
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
		{name: "idx_oplog_created_id", columns: []string{"created_at", "id"}},
		{name: "idx_oplog_scope_created_id", columns: []string{"tenant_user", "app", "created_at", "id"}},
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
