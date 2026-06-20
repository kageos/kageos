package main

import (
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/ce_shi"
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/ce_shi/ce_shi"
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/ce_shi/meeting"
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/ce_shi/ticket_management"
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/hot_news"
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/sales_leads"
	_ "github.com/kageos/kageos/namespace/system/test22/code/api/ticket_management"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
