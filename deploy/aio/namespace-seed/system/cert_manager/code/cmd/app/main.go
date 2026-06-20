package main

import (
	_ "github.com/kageos/kageos/namespace/system/cert_manager/code/api/aliyun"
	_ "github.com/kageos/kageos/namespace/system/cert_manager/code/api/aliyun/cert_manager"
	_ "github.com/kageos/kageos/namespace/system/cert_manager/code/api/cloudflare"
	_ "github.com/kageos/kageos/namespace/system/cert_manager/code/api/cloudflare/cert_manager"
	_ "github.com/kageos/kageos/namespace/system/cert_manager/code/api/tencent"
	_ "github.com/kageos/kageos/namespace/system/cert_manager/code/api/tencent/cert_manager"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
