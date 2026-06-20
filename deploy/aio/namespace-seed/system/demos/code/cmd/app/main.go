package main

import (
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/ai_asset_manager"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/brandcheck"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/cert_manager"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/inventory"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/meeting"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/midnight_pub"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/nps"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/printdrop"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/rfpfiller"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/weixin"
	_ "github.com/kageos/kageos/namespace/system/demos/code/api/weixin/wechat_articles"

	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
