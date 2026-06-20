package main

import (
	_ "github.com/kageos/kageos/namespace/system/ticket_sys/code/api/ticket"
	_ "github.com/kageos/kageos/namespace/system/ticket_sys/code/api/v1"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
