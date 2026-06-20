package main

import (
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/dingtalk"
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/feishu"
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/github"
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/google"
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/notion"
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/wecom"
	_ "github.com/kageos/kageos/namespace/system/connector/code/api/wecom/wecom_group_robot"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
