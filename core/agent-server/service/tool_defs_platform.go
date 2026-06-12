package service

func platformTools(r *ToolRegistry) []Tool {
	return []Tool{
		&SearchTool{registry: r},
		&WebSearchTool{},
		&NotifyUserTool{publisher: r.messagePublisher},
	}
}
