package enterprise

// Feature 功能名称常量
// 用于在 License 中标识不同的企业功能
// 不同版本（企业版、旗舰版、至尊版等）支持的功能不同
const (
	// FeatureOperateLog 操作日志功能
	// 支持记录用户在平台上的所有操作行为（新增、更新、删除等）
	// 支持版本：enterprise, flagship
	FeatureOperateLog = "operate_log"
	
	// 后续可以添加更多功能常量，例如：
	// FeatureWorkflow = "workflow"        // 工作流功能
	// FeatureAdvancedAnalytics = "advanced_analytics"  // 高级分析功能
	// FeatureCustomBranding = "custom_branding"  // 自定义品牌功能
)

