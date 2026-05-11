package app

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

var Temp = &TableTemplate{
	BaseConfig: BaseConfig{},
}

type GetReq struct {
	Name string `json:"name"`
}

type Test struct {
	ID    int64  `json:"id" gorm:"primary_key"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (t *Test) TableName() string {
	return "test"
}

func GetHandle(ctx *Context, resp response.Response) error {
	var req Test
	err := ctx.ShouldBind(&req)
	if err != nil {
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	err = db.AutoMigrate(&Test{})
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	var tests []*Test
	db = db.Model(&Test{}).Where("name = ?", req.Name)
	pageInfo := &query.PageSortReq{PageSize: 20}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}
	if order := pageInfo.GetOrder(); order != "" {
		db = db.Order(order)
	}
	err = db.Offset(pageInfo.GetOffset()).Limit(pageInfo.GetLimit()).Find(&tests).Error
	if err != nil {
		return err
	}
	err = resp.Table(response.TableResult{
		Items:      tests,
		TotalCount: total,
		PageInfo:   pageInfo,
	}).Build()
	if err != nil {
		return err
	}
	return nil
}

func AddHandle(ctx *Context, resp response.Response) error {
	var req Test
	err := ctx.ShouldBind(&req)
	if err != nil {
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	err = db.AutoMigrate(&Test{})
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	err = db.Create(&req).Error
	if err != nil {
		return err
	}
	err = resp.Form(req).Build()
	if err != nil {
		return err
	}
	return nil
}
