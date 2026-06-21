package ai_asset_manager

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// Asset 统一资产表
type Asset struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	Title    string `json:"title" gorm:"column:title" widget:"name:标题;type:input" validate:"max=200"`
	Category string `json:"category" gorm:"column:category" widget:"name:分类;type:input" validate:"max=100"`
	Tags     string `json:"tags" gorm:"column:tags" widget:"name:标签;type:input;placeholder:多个标签用逗号分隔" validate:"max=500"`
	Prompt   string `json:"prompt" gorm:"column:prompt;type:text" widget:"name:提示词;type:text_area"`
	Desc     string `json:"desc" gorm:"column:description" widget:"name:描述;type:text_area" validate:"max=500"`
	// 图片地址（纯URL，存储用）
	ImageURL string `json:"image_url" gorm:"column:image_url" widget:"name:图片地址;type:input;placeholder:支持网络URL" validate:"max=500"`
	// 图片预览（富文本HTML，导入/新增/更新时自动生成 <img src="..." />）
	ImagePreview string `json:"image_preview" gorm:"column:image_preview;type:text" widget:"name:图片预览;type:richtext;height:300" hide:"create,update"`
	// 状态
	Status string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:草稿,已发布,已归档;options_colors:909399,67C23A,FF9800;render_default:草稿" validate:"required,oneof=草稿 已发布 已归档"`
}

func (a *Asset) TableName() string { return "asset" }

// buildImagePreviewHTML 根据图片URL生成富文本HTML
func buildImagePreviewHTML(imageURL string) string {
	if imageURL == "" {
		return ""
	}
	return fmt.Sprintf("<img src=\"%s\" style=\"max-width:100%%;border-radius:8px;\" />", imageURL)
}

var AssetTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "资产列表",
		Desc:    "统一管理图片资产和Prompt模板，支持按标题、分类、标签搜索。",
		Request: &AssetListReq{},
		CreateTables: []interface{}{
			&Asset{},
		},
	},
	AutoCrudTable: &Asset{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row Asset
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		// 有图片地址但无预览时，自动生成富文本HTML
		if row.ImageURL != "" && row.ImagePreview == "" {
			row.ImagePreview = buildImagePreviewHTML(row.ImageURL)
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[AssetAdd] 创建资产失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		if err := req.BindChangedFields(&Asset{}); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()

		// 更新了图片地址但没更新预览时，自动刷新富文本
		if _, hasURL := updates["image_url"]; hasURL {
			imageURL := updates["image_url"].(string)
			if preview, ok := updates["image_preview"].(string); !ok || preview == "" {
				if imageURL != "" {
					updates["image_preview"] = buildImagePreviewHTML(imageURL)
				}
			}
		}

		err := db.Model(&Asset{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[AssetUpdate] 更新资产失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&Asset{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
			"deleted_at": time.Now(),
		}).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[AssetDelete] 删除资产失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// AssetListReq 资产列表请求
type AssetListReq struct {
	Title             string `json:"title" form:"title" widget:"name:标题;type:input"`
	Category          string `json:"category" form:"category" widget:"name:分类;type:input"`
	Tags              string `json:"tags" form:"tags" widget:"name:标签;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:草稿,已发布,已归档;options_colors:909399,67C23A,FF9800"`
	StartTime         string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime           string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

func AssetList(ctx *app.Context, resp response.Response) error {
	var req AssetListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "AssetList ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Asset{})

	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.Category != "" {
		queryDB = queryDB.Where("category LIKE ?", "%"+req.Category+"%")
	}
	if req.Tags != "" {
		queryDB = queryDB.Where("tags LIKE ?", "%"+req.Tags+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("id DESC")
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "AssetList Count err: %v", err)
		return err
	}

	var lists []*Asset
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "AssetList Find err: %v", err)
		return err
	}

	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// AssetImportReq 从旧SQLite导入资产请求
type AssetImportReq struct {
	SQLiteFile string `json:"sqlite_file" widget:"name:SQLite文件;type:files;accept:.sqlite,.db;max_count:1" validate:"required"`
	SourceType string `json:"source_type" widget:"name:数据来源;type:select;options:image_assets,prompt_templates;options_colors:409EFF,67C23A;render_default:image_assets" validate:"required,oneof=image_assets prompt_templates"`
}

// AssetImportResp 导入响应
type AssetImportResp struct {
	ImportedCount int    `json:"imported_count" widget:"name:导入条数;type:integer"`
	SkippedCount  int    `json:"skipped_count" widget:"name:跳过条数;type:integer"`
	Message       string `json:"message" widget:"name:说明;type:text"`
}

func AssetImport(ctx *app.Context, resp response.Response) error {
	var req AssetImportReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.SQLiteFile)
	if len(inputFiles) == 0 || inputFiles[0] == "" {
		return resp.Form(&AssetImportResp{Message: "文件未上传成功，请重新上传"}).Build()
	}
	sqlitePath := inputFiles[0]
	defer fs.RemoveFiles(inputFiles)

	// 读取旧 SQLite
	records, err := readOldSQLite(sqlitePath, req.SourceType)
	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[AssetImport] 读取SQLite失败, req: %+v, err: %v", req, err)
		return resp.Form(&AssetImportResp{Message: "读取SQLite失败：" + err.Error()}).Build()
	}

	if len(records) == 0 {
		return resp.Form(&AssetImportResp{
			ImportedCount: 0,
			SkippedCount:  0,
			Message:       "未找到可导入的记录",
		}).Build()
	}

	db := ctx.GetGormDB()
	inserted := 0
	skipped := 0

	err = db.Transaction(func(tx *gorm.DB) error {
		for _, r := range records {
			if r.Title == "" {
				skipped++
				continue
			}
			if r.Status == "" {
				r.Status = "草稿"
			}
			if r.ImagePreview == "" && r.ImageURL != "" {
				r.ImagePreview = buildImagePreviewHTML(r.ImageURL)
			}

			asset := Asset{
				Title:        r.Title,
				Category:     r.Category,
				Tags:         r.Tags,
				Prompt:       r.Prompt,
				Desc:         r.Desc,
				ImageURL:     r.ImageURL,
				ImagePreview: r.ImagePreview,
				Status:       r.Status,
			}

			if err := tx.Create(&asset).Error; err != nil {
				logger.Warnf(ctx, "[AssetImport] 写入失败, title=%s, err=%v", asset.Title, err)
				continue
			}
			inserted++
		}
		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[AssetImport] 批量写入失败, req: %+v, err: %v", req, err)
		return resp.Form(&AssetImportResp{
			ImportedCount: inserted,
			SkippedCount:  skipped,
			Message:       fmt.Sprintf("部分写入失败，已导入 %d 条", inserted),
		}).Build()
	}

	return resp.Form(&AssetImportResp{
		ImportedCount: inserted,
		SkippedCount:  skipped,
		Message:       fmt.Sprintf("导入完成！新增 %d 条资产记录（跳过 %d 条空标题记录）", inserted, skipped),
	}).Build()
}

// importRecord 导入记录结构
type importRecord struct {
	Title        string
	Category     string
	Tags         string
	Prompt       string
	Desc         string
	ImageURL     string
	ImagePreview string
	Status       string
}

// readOldSQLite 读取旧版 SQLite，返回可导入记录
func readOldSQLite(sqlitePath, sourceType string) ([]importRecord, error) {
	db, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	switch sourceType {
	case "image_assets":
		return readImageAssets(db)
	case "prompt_templates":
		return readPromptTemplates(db)
	default:
		return nil, fmt.Errorf("未知数据来源: %s", sourceType)
	}
}

func readImageAssets(db *sql.DB) ([]importRecord, error) {
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='image_assets'").Scan(&tableName)
	if err != nil {
		return nil, fmt.Errorf("image_assets 表不存在")
	}

	rows, err := db.Query(`SELECT title, category, tags_json, original_prompt, review_summary, public_url, review_status FROM image_assets`)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var records []importRecord
	for rows.Next() {
		var title, category, tagsJSON, originalPrompt, reviewSummary, publicURL, reviewStatus string
		if err := rows.Scan(&title, &category, &tagsJSON, &originalPrompt, &reviewSummary, &publicURL, &reviewStatus); err != nil {
			continue
		}

		if title == "" {
			continue
		}

		tags := ""
		if tagsJSON != "" && tagsJSON != "[]" {
			var tagsList []interface{}
			if err := json.Unmarshal([]byte(tagsJSON), &tagsList); err == nil {
				var tagStrs []string
				for _, t := range tagsList {
					tagStrs = append(tagStrs, fmt.Sprintf("%v", t))
				}
				tags = strings.Join(tagStrs, ",")
			}
		}

		status := "草稿"
		switch reviewStatus {
		case "approved":
			status = "已发布"
		case "rejected":
			status = "已归档"
		}

		imageURL := publicURL
		imagePreview := ""
		if imageURL != "" {
			imagePreview = buildImagePreviewHTML(imageURL)
		}

		records = append(records, importRecord{
			Title:        strings.TrimSpace(title),
			Category:     strings.TrimSpace(category),
			Tags:         tags,
			Prompt:       originalPrompt,
			Desc:         reviewSummary,
			ImageURL:     imageURL,
			ImagePreview: imagePreview,
			Status:       status,
		})
	}

	return records, nil
}

func readPromptTemplates(db *sql.DB) ([]importRecord, error) {
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='prompt_templates'").Scan(&tableName)
	if err != nil {
		return nil, fmt.Errorf("prompt_templates 表不存在")
	}

	rows, err := db.Query(`SELECT title, category, tags, prompt_content, description, status FROM prompt_templates`)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	statusMap := map[string]string{
		"draft":     "草稿",
		"published": "已发布",
		"archived":  "已归档",
	}

	var records []importRecord
	for rows.Next() {
		var title, category, tags, promptContent, description, status string
		if err := rows.Scan(&title, &category, &tags, &promptContent, &description, &status); err != nil {
			continue
		}

		if title == "" {
			continue
		}

		records = append(records, importRecord{
			Title:    strings.TrimSpace(title),
			Category: strings.TrimSpace(category),
			Tags:     tags,
			Prompt:   promptContent,
			Desc:     description,
			Status:   statusMap[status],
		})
	}

	return records, nil
}

var AssetImportTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "从SQLite导入资产",
		Desc:     "上传旧版SQLite文件，选择数据来源表，一键导入到统一资产表。",
		Request:  &AssetImportReq{},
		Response: &AssetImportResp{},
	},
}

func init() {
	packageContext.GET("asset_list.table", AssetList, AssetTemplate)
	packageContext.POST("asset_import.form", AssetImport, AssetImportTemplate)
}
