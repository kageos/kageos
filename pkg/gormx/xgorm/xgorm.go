package xgorm

import (
	"log/slog"

	"gorm.io/gorm"
)

// XGorm 包装 gorm.DB，重写 Create/Update/Updates/Delete，签名与 gorm.DB 一致，仅加一层拦截（日志、后续可扩展钩子）。
// 嵌入 *gorm.DB，可直接当 DB 用；只有直接 x.Create / x.Update 等会走拦截，链式 x.Where().Create() 会走原生 gorm（Where 返回 *gorm.DB）。
type XGorm struct {
	*gorm.DB
}

// New 用原生 *gorm.DB 构造 XGorm。
func New(db *gorm.DB) *XGorm {
	return &XGorm{DB: db}
}

// Create 签名与 gorm.DB.Create 一致，先打日志再调用底层。
func (x *XGorm) Create(value interface{}) *gorm.DB {
	logCreate(value)
	return x.DB.Create(value)
}

// Update 签名与 gorm.DB.Update 一致。
func (x *XGorm) Update(column string, value interface{}) *gorm.DB {
	logUpdate(column, value)
	return x.DB.Update(column, value)
}

// Updates 签名与 gorm.DB.Updates 一致。
func (x *XGorm) Updates(value interface{}) *gorm.DB {
	logUpdates(value)
	return x.DB.Updates(value)
}

// Delete 签名与 gorm.DB.Delete 一致。
func (x *XGorm) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	logDelete(value, conds)
	return x.DB.Delete(value, conds...)
}

// 当前仅打日志，后续可改为可配置的钩子（如注入 created_by、审计等）
func logCreate(value interface{}) {
	slog.Info("xgorm: create", "model", value)
}

func logUpdate(column string, value interface{}) {
	slog.Info("xgorm: update", "column", column, "value", value)
}

func logUpdates(value interface{}) {
	slog.Info("xgorm: updates", "value", value)
}

func logDelete(value interface{}, conds []interface{}) {
	slog.Info("xgorm: delete", "value", value, "conds", conds)
}
