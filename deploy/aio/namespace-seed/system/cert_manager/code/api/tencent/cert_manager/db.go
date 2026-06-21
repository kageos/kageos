package cert_manager

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"gorm.io/gorm"
)

func certManagerDB(ctx *app.Context) (*gorm.DB, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	return db, nil
}

func certManagerTables() []interface{} {
	return []interface{}{
		&CertTencentConfig{},
		&CertTencentDomain{},
		&CertTencentRequest{},
		&CertTencentAsset{},
	}
}
