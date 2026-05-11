package response

import (
	"testing"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type tableQueryTestItem struct {
	ID       int `gorm:"primaryKey"`
	Name     string
	Category string
	Score    int
}

func setupTableQueryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&tableQueryTestItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	items := []tableQueryTestItem{
		{Name: "alpha", Category: "a", Score: 50},
		{Name: "beta", Category: "b", Score: 40},
		{Name: "gamma", Category: "a", Score: 30},
		{Name: "delta", Category: "b", Score: 20},
		{Name: "epsilon", Category: "a", Score: 10},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func TestTableQueryDefaultsToFirstPage(t *testing.T) {
	db := setupTableQueryTestDB(t)
	var rows []tableQueryTestItem
	var total int64
	if err := db.Model(&tableQueryTestItem{}).Count(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	pageInfo := &query.PageSortReq{}
	if err := db.Model(&tableQueryTestItem{}).
		Offset(pageInfo.GetOffset()).
		Limit(pageInfo.GetLimit()).
		Find(&rows).Error; err != nil {
		t.Fatalf("find: %v", err)
	}

	resp := (&RunFunctionResp{}).Table(TableResult{
		Items:      rows,
		TotalCount: total,
		PageInfo:   pageInfo,
	})
	if err := resp.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(rows) != 5 {
		t.Fatalf("len(rows) = %d, want 5", len(rows))
	}
	got := resp.(*RunFunctionResp).TableData.Paginated
	if got == nil {
		t.Fatal("Paginated = nil")
	}
	if got.CurrentPage != 1 || got.PageSize != 20 || got.TotalCount != 5 || got.TotalPages != 1 {
		t.Fatalf("Paginated = %#v", got)
	}
}

func TestTableQuerySortsAndPaginates(t *testing.T) {
	db := setupTableQueryTestDB(t)
	var rows []tableQueryTestItem

	pageInfo := &query.PageSortReq{Page: 2, PageSize: 2, Sorts: "-score"}
	queryDB := db.Model(&tableQueryTestItem{})
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if order := pageInfo.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	if err := queryDB.Offset(pageInfo.GetOffset()).Limit(pageInfo.GetLimit()).Find(&rows).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	resp := (&RunFunctionResp{}).Table(TableResult{
		Items:      rows,
		TotalCount: total,
		PageInfo:   pageInfo,
	})
	if err := resp.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].Name != "gamma" || rows[1].Name != "delta" {
		t.Fatalf("rows = %#v", rows)
	}
	got := resp.(*RunFunctionResp).TableData.Paginated
	if got.CurrentPage != 2 || got.PageSize != 2 || got.TotalCount != 5 || got.TotalPages != 3 {
		t.Fatalf("Paginated = %#v", got)
	}
}

func TestTableQueryUsesCallerWhereWithoutSearchProtocol(t *testing.T) {
	db := setupTableQueryTestDB(t)
	var rows []tableQueryTestItem

	queryDB := db.Model(&tableQueryTestItem{}).Where("category = ?", "missing")
	pageInfo := &query.PageSortReq{PageSize: 2}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if err := queryDB.Offset(pageInfo.GetOffset()).Limit(pageInfo.GetLimit()).Find(&rows).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	resp := (&RunFunctionResp{}).Table(TableResult{
		Items:      rows,
		TotalCount: total,
		PageInfo:   pageInfo,
	})
	if err := resp.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("len(rows) = %d, want 0", len(rows))
	}
	got := resp.(*RunFunctionResp).TableData.Paginated
	if got.TotalCount != 0 || got.TotalPages != 0 || got.PageSize != 2 {
		t.Fatalf("Paginated = %#v", got)
	}
}
