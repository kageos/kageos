package dto

type PageInfoReq struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
type PageInfoResp struct {
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int         `json:"total_count"`
	Items      interface{} `json:"items"`
}
