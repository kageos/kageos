package model

import (
	"encoding/json"
	"gorm.io/gorm"
	"strings"
)

// SingleResourcePath only narrows a legacy archive when its complete record
// count belongs to one path. A top-N summary of mixed content cannot grant access.
func (b *LogArchiveBatch) SingleResourcePath() string {
	if b.ResourcePath != "" {
		return b.ResourcePath
	}
	var summary struct {
		Paths []struct {
			Path  string `json:"resource_path"`
			Count int64  `json:"count"`
		} `json:"top_resource_paths"`
	}
	if json.Unmarshal(b.SummaryJSON, &summary) != nil || len(summary.Paths) != 1 || b.RecordCount <= 0 || summary.Paths[0].Count != b.RecordCount {
		return ""
	}
	path := summary.Paths[0].Path
	root := "/" + b.TenantUser + "/" + b.App
	if path == root || strings.HasPrefix(path, root+"/") {
		return path
	}
	return ""
}

func backfillLogArchiveScopes(db *gorm.DB) error {
	var cursor int64
	for {
		var batches []LogArchiveBatch
		if err := db.Where("id > ? AND COALESCE(resource_path,'') = '' AND summary_json IS NOT NULL", cursor).Order("id").Limit(500).Find(&batches).Error; err != nil {
			return err
		}
		for _, batch := range batches {
			cursor = batch.ID
			if path := batch.SingleResourcePath(); path != "" {
				if err := db.Model(&LogArchiveBatch{}).Where("id = ? AND COALESCE(resource_path,'') = ''", batch.ID).Update("resource_path", path).Error; err != nil {
					return err
				}
			}
		}
		if len(batches) < 500 {
			return nil
		}
	}
}
