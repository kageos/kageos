package callback

type OnTableAddRowReq struct {
	Row interface{} `json:"row"`
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
	return c.ID
}
func (c *OnTableUpdateRowReq) GetUpdates() map[string]interface{} {
	return c.Updates
}

type OnTableUpdateRowResp struct {
}
