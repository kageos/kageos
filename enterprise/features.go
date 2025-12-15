package enterprise

// Feature 功能名称常量
// 用于在 License 中标识不同的企业功能
// 不同版本（企业版、旗舰版、至尊版等）支持的功能不同
//
// 使用方式：
//   - 在 License 的 Features 结构体中定义功能开关
//   - 通过 license.Manager.HasFeature(enterprise.FeatureOperateLog) 检查功能
//   - 在中间件中使用 RequireFeature(enterprise.FeatureOperateLog) 保护接口
const (
	// FeatureOperateLog 操作日志功能
	// 支持记录用户在平台上的所有操作行为（新增、更新、删除等）
	// 支持版本：enterprise, flagship
	FeatureOperateLog = "operate_log"

	// FeatureOrganization 组织架构功能
	// 支持多层级组织管理、组织成员管理、应用归属组织等
	// 支持版本：enterprise, flagship
	FeatureOrganization = "organization"

	// FeaturePermission 权限管理功能
	// 支持基于组织的 RBAC、角色管理、权限检查等
	// 支持版本：enterprise, flagship
	FeaturePermission = "permission"

	// FeatureWorkflow 工作流功能
	// 支持自动化工作流、审批流程等
	// 支持版本：enterprise, flagship
	FeatureWorkflow = "workflow"

	// FeatureApproval 审批流程功能
	// 支持审批流程管理、审批记录等
	// 支持版本：enterprise, flagship
	FeatureApproval = "approval"

	// FeatureScheduledTask 定时任务功能
	// 支持定时任务调度、任务管理等
	// 支持版本：flagship
	FeatureScheduledTask = "scheduled_task"

	// FeatureRecycleBin 回收站功能
	// 支持数据回收站、数据恢复等
	// 支持版本：enterprise, flagship
	FeatureRecycleBin = "recycle_bin"

	// FeatureChangeLog 变更日志功能
	// 支持应用版本变更记录、变更历史等
	// 支持版本：enterprise, flagship
	FeatureChangeLog = "change_log"

	// FeatureNotification 通知中心功能
	// 支持系统通知、消息推送等
	// 支持版本：enterprise, flagship
	FeatureNotification = "notification"

	// FeatureConfigManagement 配置管理功能
	// 支持配置管理、配置版本控制等
	// 支持版本：flagship
	FeatureConfigManagement = "config_management"

	// FeatureQuickLink 快链功能
	// 支持快速链接、快捷方式等
	// 支持版本：enterprise, flagship
	FeatureQuickLink = "quick_link"
)

