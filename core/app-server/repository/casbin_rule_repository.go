package repository

import (
	"gorm.io/gorm"
)

// CasbinRuleRepository Casbin 规则仓储
// 用于直接操作 casbin_rule 表，支持扩展字段（如 app_id）
type CasbinRuleRepository struct {
	db *gorm.DB
}

// NewCasbinRuleRepository 创建 Casbin 规则仓储
func NewCasbinRuleRepository(db *gorm.DB) *CasbinRuleRepository {
	return &CasbinRuleRepository{
		db: db,
	}
}

// UpdateAppID 更新 casbin_rule 表的 app_id 字段
// 参数：
//   - username: 用户名（v0）
//   - resourcePath: 资源路径（v1）
//   - action: 操作（v2）
//   - appID: 应用ID
//
// 返回：
//   - rowsAffected: 更新的行数（0 表示未找到匹配记录）
//   - error: 如果更新失败返回错误
//
// 说明：
//   - 如果返回 0 行，可能是记录还未写入（时序问题），调用方应该重试
func (r *CasbinRuleRepository) UpdateAppID(username, resourcePath, action string, appID int64) (int64, error) {
	result := r.db.Exec(
		"UPDATE casbin_rule SET app_id = ? WHERE ptype = 'p' AND v0 = ? AND v1 = ? AND v2 = ?",
		appID, username, resourcePath, action,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// PermissionRecord 权限记录（Repository 层使用）
type PermissionRecord struct {
	ID       int64  `gorm:"column:id"`
	Ptype    string `gorm:"column:ptype"`
	V0       string `gorm:"column:v0"` // 用户
	V1       string `gorm:"column:v1"` // 资源路径
	V2       string `gorm:"column:v2"` // 操作
	AppID    int64  `gorm:"column:app_id"`
}

// GetPermissionsByAppIDAndUser 根据 app_id 和用户信息查询该工作空间下指定用户的所有权限
// ⭐ 优化：直接通过 app_id + 用户信息查询，利用索引，性能更好
// 参数：
//   - appID: 应用ID
//   - username: 用户名
//
// 返回：
//   - []PermissionRecord: 权限列表
//   - error: 如果查询失败返回错误
func (r *CasbinRuleRepository) GetPermissionsByAppIDAndUser(appID int64, username string) ([]PermissionRecord, error) {
	var rules []PermissionRecord
	err := r.db.Table("casbin_rule").
		Where("app_id = ? AND ptype = 'p' AND v0 = ?", appID, username).
		Find(&rules).Error
	if err != nil {
		return nil, err
	}

	return rules, nil
}

