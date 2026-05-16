package enterprise

// Feature 功能名称常量。
// MVP 阶段只保留已经有真实实现的企业功能位。
//
// 使用方式：
//   - 在 License 的 Features 结构体中定义功能开关
//   - 通过 license.Manager.HasFeature(enterprise.FeatureOperateLog) 检查功能
//   - 在中间件中使用 RequireFeature(enterprise.FeatureOperateLog) 保护接口
const (
	// FeatureOperateLog 操作日志功能
	// 支持记录用户在平台上的所有操作行为（新增、更新、删除等）
	// 支持版本：enterprise
	FeatureOperateLog = "operate_log"

	// FeaturePermission 高级权限治理功能
	// 基础用户授权和权限检查社区版可用；组织架构授权、权限申请审批、自定义角色等高级治理能力由该功能控制。
	// 支持版本：enterprise
	FeaturePermission = "permission"
)
