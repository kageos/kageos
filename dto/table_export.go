package dto

type TableExportBlock struct {
	Index    int    `json:"index"`
	StartRow int64  `json:"start_row"`
	EndRow   int64  `json:"end_row"`
	RowCount int    `json:"row_count"`
	Cursor   string `json:"cursor"`
}

type TableExportPlanReq struct {
	Filters   map[string]interface{} `json:"filters"`
	ChunkSize int                    `json:"chunk_size"`
}

type TableExportPlanResp struct {
	Snapshot string             `json:"snapshot"`
	Total    int64              `json:"total"`
	Blocks   []TableExportBlock `json:"blocks"`
}

type TableExportChunkReq struct {
	Snapshot string                 `json:"snapshot"`
	Cursor   string                 `json:"cursor"`
	Limit    int                    `json:"limit"`
	Filters  map[string]interface{} `json:"filters"`
}

type TableExportChunkResp struct {
	Rows []map[string]interface{} `json:"rows"`
}
