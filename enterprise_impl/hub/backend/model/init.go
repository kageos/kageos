package model

import (
	"gorm.io/gorm"
)

// InitTables 初始化数据库表
//
// Hub 两表方案（单源）：
//
//  1. HubDirectory（hub_directories）
//     一个「已发布目录」一条。存元信息：name、description、category、tags、full_code_path、
//     publisher、服务费、当前版本号等。Version/VersionNum 指向当前最新版本。
//     关联：被 HubSnapshot 通过 HubDirectoryID 引用。
//
//  2. HubSnapshot（hub_snapshots）
//     每个版本一条，同一目录多版本 = 多行。存：Version、VersionNum、IsCurrent、Description（更新说明）。
//     快照三字段（单源）：SnapshotTree（目录结构，展示用）、SnapshotFiles（文件列表，复制用）、
//     SnapshotFunctionDefs（函数定义列表，预览用）。SnapshotData 保留兼容旧数据。
//     关联：HubDirectoryID -> HubDirectory。
func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&HubDirectory{},     // 1. 已发布目录元信息
		&HubSnapshot{},     // 2. 每版本一条，三字段 + SnapshotData 兼容
		&HubDirectoryStar{}, // 3. 目录星星记录（类似 GitHub star）
	)
}

