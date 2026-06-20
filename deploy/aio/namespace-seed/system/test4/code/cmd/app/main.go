package main

import (
	_ "github.com/kageos/kageos/namespace/system/test4/code/api/alert"
	_ "github.com/kageos/kageos/namespace/system/test4/code/api/evaluation"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
