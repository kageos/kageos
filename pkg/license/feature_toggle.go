package license

// FeatureToggle 功能开关接口
// 用于统一管理功能开关，支持 License 检查、A/B 测试、用户级别控制等
//
// 设计目的：
//   - 统一功能开关接口，便于测试和扩展
//   - 支持未来扩展（A/B 测试、用户级别控制等）
//   - 提高代码可测试性（可以 mock FeatureToggle）
type FeatureToggle interface {
	// IsEnabled 检查功能是否启用
	// 参数：
	//   - feature: 功能名称（使用 enterprise.FeatureXXX 常量）
	//
	// 返回：
	//   - bool: 功能是否启用
	IsEnabled(feature string) bool

	// IsEnabledForUser 检查功能是否对特定用户启用（未来扩展）
	// 参数：
	//   - feature: 功能名称
	//   - user: 用户名
	//
	// 返回：
	//   - bool: 功能是否对该用户启用
	//
	// 说明：
	//   - 当前实现：基于 License 检查，不考虑用户级别
	//   - 未来扩展：可以支持 A/B 测试、用户级别控制等
	IsEnabledForUser(feature string, user string) bool
}

// LicenseFeatureToggle License 功能开关实现
// 基于 License Manager 实现功能开关
type LicenseFeatureToggle struct {
	licenseMgr *Manager
}

// NewLicenseFeatureToggle 创建 License 功能开关
// 参数：
//   - licenseMgr: License 管理器（如果为 nil，使用全局管理器）
//
// 返回：
//   - FeatureToggle: 功能开关实例
func NewLicenseFeatureToggle(licenseMgr *Manager) FeatureToggle {
	if licenseMgr == nil {
		licenseMgr = GetManager()
	}
	return &LicenseFeatureToggle{
		licenseMgr: licenseMgr,
	}
}

// IsEnabled 检查功能是否启用
func (t *LicenseFeatureToggle) IsEnabled(feature string) bool {
	return t.licenseMgr.HasFeature(feature)
}

// IsEnabledForUser 检查功能是否对特定用户启用
// 当前实现：基于 License 检查，不考虑用户级别
// 未来扩展：可以支持 A/B 测试、用户级别控制等
func (t *LicenseFeatureToggle) IsEnabledForUser(feature string, user string) bool {
	// 当前实现：基于 License 检查
	return t.IsEnabled(feature)
	
	// 未来扩展示例：
	// if t.IsEnabled(feature) {
	//     // 检查用户是否在 A/B 测试组中
	//     if isUserInABTestGroup(user, feature) {
	//         return true
	//     }
	//     // 检查用户级别
	//     if isUserLevelEnabled(user, feature) {
	//         return true
	//     }
	// }
	// return false
}

// GetDefaultFeatureToggle 获取默认功能开关（基于全局 License Manager）
// 用于快速获取功能开关实例，无需手动创建
//
// 使用示例：
//   toggle := license.GetDefaultFeatureToggle()
//   if toggle.IsEnabled(enterprise.FeatureOperateLog) {
//       // 功能启用
//   }
func GetDefaultFeatureToggle() FeatureToggle {
	return NewLicenseFeatureToggle(GetManager())
}

