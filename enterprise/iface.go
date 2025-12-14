package enterprise

import "gorm.io/gorm"

// InitOptions 企业功能初始化选项
// 用于向企业功能模块注入系统依赖（如数据库连接等）
// 所有企业功能模块在初始化时都会接收这个选项
type InitOptions struct {
	DB *gorm.DB // 数据库连接，用于企业功能的数据持久化
}

// Init 企业功能初始化接口
// 所有企业版本的功能模块都必须实现这个接口
// 实现该接口的功能模块在初始化时会接收到系统依赖（如数据库连接）
//
// 使用场景：
//   - 企业版插件在加载时通过此接口进行初始化
//   - 社区版使用空实现，不会执行任何操作
//
// 示例：
//   type MyEnterpriseFeature struct {
//       db *gorm.DB
//   }
//   func (m *MyEnterpriseFeature) Init(opt *InitOptions) error {
//       m.db = opt.DB
//       return m.db.AutoMigrate(&MyTable{})
//   }
type Init interface {
	Init(opt *InitOptions) error
}
