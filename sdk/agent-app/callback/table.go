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
}

type OnTableUpdateRowResp struct {
}
