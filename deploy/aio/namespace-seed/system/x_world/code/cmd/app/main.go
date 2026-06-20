package main

import (
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/agent_debate"
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/agent_world"
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/auction"
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/rating"
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/ticket_management"
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/vote"
	_ "github.com/kageos/kageos/namespace/system/x_world/code/api/werewolf"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
