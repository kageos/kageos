package main

import (
	_ "github.com/kageos/kageos/namespace/system/kimi/code/api/inventory"
	"github.com/kageos/kageos-sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
