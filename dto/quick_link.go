package dto

// CreateQuickLinkReq 创建快链请求
type CreateQuickLinkReq struct {
	Name           string                            `json:"name" binding:"required"`           // 快链名称
	FunctionRouter string                            `json:"function_router" binding:"required"` // 函数路由
	FunctionMethod string                            `json:"function_method" binding:"required"` // 函数HTTP方法
	TemplateType   string                            `json:"template_type" binding:"required"`   // 模板类型:form,table,chart
	RequestParams  map[string]interface{}            `json:"request_params"`                    // 表单参数(完整的FieldValue结构)
	FieldMetadata  map[string]map[string]interface{} `json:"field_metadata"`                    // 字段元数据(editable,readonly,hint,highlight)
	Metadata       map[string]interface{}            `json:"metadata"`                           // 其他元数据(table_state,chart_filters,response_params)
}

// CreateQuickLinkResp 创建快链响应
type CreateQuickLinkResp struct {
	ID   int64  `json:"id"`   // 快链ID
	Name string `json:"name"` // 快链名称
	URL  string `json:"url"`  // 生成的快链URL
}

// GetQuickLinkResp 获取快链响应
type GetQuickLinkResp struct {
	ID             int64                             `json:"id"`
	Name           string                            `json:"name"`
	FunctionRouter string                            `json:"function_router"`
	FunctionMethod string                            `json:"function_method"`
	TemplateType   string                            `json:"template_type"`
	RequestParams  map[string]interface{}            `json:"request_params"`
	FieldMetadata  map[string]map[string]interface{} `json:"field_metadata"`
	Metadata       map[string]interface{}            `json:"metadata"`
	CreatedAt      string                            `json:"created_at"`
	UpdatedAt      string                            `json:"updated_at"`
	CreatedBy      string                            `json:"created_by"`
}

// ListQuickLinksReq 快链列表请求
type ListQuickLinksReq struct {
	FunctionRouter string `form:"function_router"` // 可选：按函数路由过滤
	Page           int    `form:"page"`            // 页码，默认1
	PageSize       int    `form:"page_size"`        // 每页数量，默认10
}

// ListQuickLinksResp 快链列表响应
type ListQuickLinksResp struct {
	List  []QuickLinkItem `json:"list"`  // 快链列表
	Total int64           `json:"total"` // 总数
}

// QuickLinkItem 快链列表项
type QuickLinkItem struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	FunctionRouter string `json:"function_router"`
	FunctionMethod string `json:"function_method"`
	TemplateType   string `json:"template_type"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// UpdateQuickLinkReq 更新快链请求
type UpdateQuickLinkReq struct {
	Name          string                            `json:"name"`           // 快链名称
	RequestParams map[string]interface{}            `json:"request_params"` // 表单参数
	FieldMetadata map[string]map[string]interface{} `json:"field_metadata"` // 字段元数据
	Metadata      map[string]interface{}            `json:"metadata"`       // 其他元数据
}

// UpdateQuickLinkResp 更新快链响应
type UpdateQuickLinkResp struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

