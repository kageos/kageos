package xgorm

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/gorm"
)

/*
  GORM Callback 示例与最佳实践

  ## 最佳实践

  1. 注册顺序：Before 钩子用 Before("gorm:xxx") 插在 GORM 默认逻辑之前，After 同理。
  2. 钩子要轻量：不要在钩子里做重 IO、调外部服务；只做写库前/后的必要逻辑（审计、日志、计数）。
  3. 不要改 d.Statement 的 SQL 语义：Before 里可以改 Model 的字段值（如 CreatedBy），不要动 Where 等。
  4. 错误处理：钩子里若设置 d.AddError(err)，会中止后续执行并回滚；谨慎使用。
  5. 上下文：用 d.Statement.Context 取请求上下文，用于审计人、TraceId 等。

  ## 常见场景

  - 审计字段：BeforeCreate/BeforeUpdate/BeforeDelete 里从 context 取当前用户，写入 CreatedBy/UpdatedBy/DeletedBy。
  - 操作日志：AfterCreate/AfterUpdate/AfterDelete 打日志或发 MQ，便于审计/排查。
  - 指标：After 里对 Create/Update/Delete 计数，上报监控。
  - 软删统一处理：BeforeDelete 里只做日志或发事件，不改 SQL（GORM 已处理软删）。
*/

// RegisterCallbacks 仅注册「打日志」的钩子（Before/After Create、Update、Delete）。
// 适合开发/排查，生产可关掉或改为 Debug 级别。
func RegisterCallbacks(db *gorm.DB) {
	registerLoggingCallbacks(db)
}

// RegisterCallbacksWithAudit 注册「日志 + 审计字段」：从 context 取当前用户，写入 CreatedBy/UpdatedBy/DeletedBy。
// 需要业务在写库前执行 db = db.WithContext(ctx)，且 ctx 里能通过 contextx.GetRequestUser 拿到用户。
func RegisterCallbacksWithAudit(db *gorm.DB) {
	registerLoggingCallbacks(db)
	registerAuditCallbacks(db)
}

// registerLoggingCallbacks 仅日志
func registerLoggingCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:before_create").Register("xgorm:before_create", func(d *gorm.DB) {
		slog.Info("xgorm callback: before_create", "table", tableName(d), "model", d.Statement.Model)
	})
	db.Callback().Create().After("gorm:after_create").Register("xgorm:after_create", func(d *gorm.DB) {
		slog.Info("xgorm callback: after_create", "table", tableName(d), "rows", d.RowsAffected)
	})

	db.Callback().Update().Before("gorm:before_update").Register("xgorm:before_update", func(d *gorm.DB) {
		slog.Info("xgorm callback: before_update", "table", tableName(d), "model", d.Statement.Model)
	})
	db.Callback().Update().After("gorm:after_update").Register("xgorm:after_update", func(d *gorm.DB) {
		slog.Info("xgorm callback: after_update", "table", tableName(d), "rows", d.RowsAffected)
	})

	db.Callback().Delete().Before("gorm:before_delete").Register("xgorm:before_delete", func(d *gorm.DB) {
		slog.Info("xgorm callback: before_delete", "table", tableName(d), "model", d.Statement.Model)
	})
	db.Callback().Delete().After("gorm:after_delete").Register("xgorm:after_delete", func(d *gorm.DB) {
		slog.Info("xgorm callback: after_delete", "table", tableName(d), "rows", d.RowsAffected)
	})
}

// registerAuditCallbacks 从 d.Statement.Context 取当前用户，写入模型的 CreatedBy/UpdatedBy/DeletedBy（若有对应字段）。
// 模型需包含 pkg/gormx/models.Base 或同名字段。
func registerAuditCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:before_create").Register("xgorm:audit_created_by", func(d *gorm.DB) {
		setCreatedBy(d)
	})
	db.Callback().Update().Before("gorm:before_update").Register("xgorm:audit_updated_by", func(d *gorm.DB) {
		setUpdatedBy(d)
	})
	db.Callback().Delete().Before("gorm:before_delete").Register("xgorm:audit_deleted_by", func(d *gorm.DB) {
		setDeletedBy(d)
	})
}

func tableName(d *gorm.DB) string {
	if d.Statement.Schema != nil {
		return d.Statement.Schema.Table
	}
	return d.Statement.Table
}

func setCreatedBy(d *gorm.DB) {
	user := getCurrentUser(d.Statement.Context)
	if user == "" {
		return
	}
	setAuditField(d, "created_by", user)
}

func setUpdatedBy(d *gorm.DB) {
	user := getCurrentUser(d.Statement.Context)
	if user == "" {
		return
	}
	setAuditField(d, "updated_by", user)
}

func setDeletedBy(d *gorm.DB) {
	user := getCurrentUser(d.Statement.Context)
	if user == "" {
		return
	}
	setAuditField(d, "deleted_by", user)
}

func getCurrentUser(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return contextx.GetRequestUser(ctx)
}

// setAuditField 对 Statement 当前要写的模型设置审计字段（created_by/updated_by/deleted_by）。
// 若为批量更新且没有用 Model，可能拿不到单条 struct，只处理 Statement.Dest 为单条记录的情况。
func setAuditField(d *gorm.DB, column string, value string) {
	dest := d.Statement.Dest
	if dest == nil {
		return
	}
	rv := reflect.ValueOf(dest)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	field := rv.FieldByName(fieldNameForColumn(column))
	if !field.IsValid() || !field.CanSet() {
		return
	}
	if field.Kind() == reflect.String {
		field.SetString(value)
	}
}

func fieldNameForColumn(column string) string {
	switch column {
	case "created_by":
		return "CreatedBy"
	case "updated_by":
		return "UpdatedBy"
	case "deleted_by":
		return "DeletedBy"
	default:
		return ""
	}
}
