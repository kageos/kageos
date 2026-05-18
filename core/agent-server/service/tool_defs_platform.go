package service

func platformTools(r *ToolRegistry) []Tool {
	return []Tool{
		&WebSearchTool{},
		&FetchURLContentTool{},
		&SearchToolsTool{registry: r},
		&SearchResourcesTool{},
	}
}
