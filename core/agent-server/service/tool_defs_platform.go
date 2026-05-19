package service

func platformTools(r *ToolRegistry) []Tool {
	return []Tool{
		&SearchToolsTool{registry: r},
		&SearchResourcesTool{},
	}
}
