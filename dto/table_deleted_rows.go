package dto

// GetTableDeletedRowsReq 查询 Table 回收站记录。
type GetTableDeletedRowsReq struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// GetTableDeletedRowsResp Table 回收站查询结果。
type GetTableDeletedRowsResp struct {
	Rows        []map[string]interface{} `json:"rows"`
	Total       int64                    `json:"total"`
	Page        int                      `json:"page"`
	PageSize    int                      `json:"page_size"`
	Table       string                   `json:"table,omitempty"`
	PackagePath string                   `json:"package_path,omitempty"`
}

// RestoreTableRowsReq 恢复 Table 软删除记录。
type RestoreTableRowsReq struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// RestoreTableRowsResp Table 记录恢复结果。
type RestoreTableRowsResp struct {
	Rows     []map[string]interface{} `json:"rows"`
	Restored int64                    `json:"restored"`
}

type PurgeTableRowsReq struct {
	IDs []int64 `json:"ids" binding:"required"`
}
