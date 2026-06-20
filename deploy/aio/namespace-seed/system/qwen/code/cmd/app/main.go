package main

import (
	_ "github.com/kageos/kageos/namespace/system/qwen/code/api/ticket"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
