package main

import (
	_ "github.com/kageos/kageos/namespace/system/test_v3/code/api/test"
	_ "github.com/kageos/kageos/namespace/system/test_v3/code/api/ticket"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
