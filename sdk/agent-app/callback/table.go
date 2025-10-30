package callback

type OnTableAddRowReq struct {
	Body interface{} `json:"body"`
}

type OnTableAddRowResp struct {
}

type OnTableDeleteRowsReq struct {
	Ids []int `json:"ids"`
}

func (c *OnTableDeleteRowsReq) GetIds() []int {
	return c.Ids
}

type OnTableDeleteRowsResp struct {
}
type OnTableUpdateRowReq struct {
	ID      int                    `json:"id"`
	Updates map[string]interface{} `json:"updates"`
}

func (c *OnTableUpdateRowReq) GetId() int {
	if c.ID != 0 {
		return c.ID
	}
	switch v := c.Updates["id"].(type) {
	case int:
		c.ID = v
		// 直接使用id
	case float64:
		c.ID = int(v)
		// 使用id
	default:
		// 处理不支持的类型
		panic("unknown id type")
	}
	return c.ID

}
func (c *OnTableUpdateRowReq) GetUpdates() map[string]interface{} {
	return c.Updates
}

type OnTableUpdateRowResp struct {
}
