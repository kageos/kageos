# xgorm

对 GORM 的薄封装：可选「包装重写 Create/Update」和「Callback 钩子」，用于日志、审计等。

## Callback 用法

### 1. 仅打日志（开发/排查）

```go
import "github.com/kageos/kageos/pkg/gormx/xgorm"

db, _ := gorm.Open(...)
xgorm.RegisterCallbacks(db)
// 之后所有 Create/Update/Delete 都会在 before/after 打日志
db.Create(&user)
db.Model(&user).Update("status", "inactive")
```

### 2. 日志 + 审计字段（CreatedBy/UpdatedBy/DeletedBy）

模型需包含 `CreatedBy`、`UpdatedBy`、`DeletedBy`（如 `pkg/gormx/models.Base`）。写库前必须 `db = db.WithContext(ctx)`，且 ctx 里能通过 `contextx.GetRequestUser(ctx)` 拿到当前用户。

```go
xgorm.RegisterCallbacksWithAudit(db)

// 在 handler/service 里
db := s.db.WithContext(c.Request.Context())
db.Create(&user)   // 自动填 user.CreatedBy
db.Save(&user)     // 自动填 user.UpdatedBy
db.Delete(&user)   // 自动填 user.DeletedBy（软删时）
```

## Callback 最佳实践

| 要点 | 说明 |
|------|------|
| **钩子要轻** | 只做写库前/后的必要逻辑（审计、日志、计数），不要重 IO、调外部 API。 |
| **Before 可改 Model** | 在 BeforeCreate/BeforeUpdate 里可以改要写入的 struct 字段（如 CreatedBy），不要动 Where/条件。 |
| **慎用 d.AddError** | 钩子里若 `d.AddError(err)` 会中止执行并回滚，只在真正非法时用。 |
| **上下文** | 用 `d.Statement.Context` 取当前请求 context，用于审计人、TraceId。 |
| **注册一次** | 在 db 创建后、业务使用前统一注册，避免重复注册。 |

## 常见场景

| 场景 | 做法 |
|------|------|
| **审计字段** | 使用 `RegisterCallbacksWithAudit`，BeforeCreate/BeforeUpdate/BeforeDelete 里从 context 取用户写入 CreatedBy/UpdatedBy/DeletedBy。 |
| **操作日志** | AfterCreate/AfterUpdate/AfterDelete 里打结构化日志或发 MQ，便于审计/排查。 |
| **指标** | After 里对 Create/Update/Delete 按表或按操作计数，上报监控。 |
| **软删** | GORM 已处理软删，BeforeDelete 里只做日志或发事件即可，不必改 SQL。 |

## XGorm 包装（可选）

需要「显式用包装」且只拦截直接调用时，可用 XGorm：

```go
x := xgorm.New(db)
x.Create(&user)   // 会打 xgorm 日志
x.Where("id=?", 1).Create(&user)  // 不会走 xgorm.Create（Where 返回 *gorm.DB）
```

若要**所有**写操作都经过钩子，请用 Callback，不要依赖 XGorm 包装。
