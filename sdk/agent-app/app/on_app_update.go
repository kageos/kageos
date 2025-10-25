package app

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
	"github.com/nats-io/nats.go"
)

type ApiInfo struct {
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	Desc         string          `json:"desc"`
	Tags         []string        `json:"tags"`
	Router       string          `json:"router"`
	Method       string          `json:"method"`
	CreateTables []string        `json:"create_tables"`
	Request      []*widget.Field `json:"request"`
	Response     []*widget.Field `json:"response"`
}

// workplace 下的api-logs 是存储每个版本的api路径，diff的底层逻辑就是，上个版本跟这个版本进行对比
func (a *App) diffApi() (add []*ApiInfo, update []*ApiInfo, delete []*ApiInfo, err error) {
	//读取旧版的api
	//跟当前新版本的做对比，

	return nil, nil, nil, err
}

func (a *App) getApis() (apis []*ApiInfo, createTables []interface{}, err error) {
	for _, info := range a.routerInfo {

		base := info.Template.GetBaseConfig()
		api := &ApiInfo{
			Code:   info.getCode(),
			Name:   base.Name,
			Desc:   base.Desc,
			Tags:   base.Tags,
			Router: info.Router,
			Method: info.Method,
		}

		templateType := info.Template.TemplateType()
		if templateType == TemplateTypeTable {
			table := info.Template.(*TableTemplate).AutoCrudTable
			requestFields, responseFields, err := widget.DecodeTable(base.Request, table)
			if err != nil {
				return nil, nil, err
			}
			api.Request = requestFields
			api.Response = responseFields
		}
		if templateType == TemplateTypeForm {
			fields, responseFields, err := widget.DecodeForm(base.Request, base.Response)
			if err != nil {
				return nil, nil, err
			}
			api.Request = fields
			api.Response = responseFields
		}
		createTables = append(createTables, base.CreateTables...)

		apis = append(apis)
	}
	return apis, createTables, nil
}

// onAppUpdate 处理当api更新时候触发
func (a *App) onAppUpdate(msg *nats.Msg) {

	//生成当前版本的 vx.json 写到api-logs的目录，
	//然后diffApi，返回变动的api信息，
	//返回之前要先创建表，

}
