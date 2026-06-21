package werewolf

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/werewolf",
	Name:        "werewolf",
	Desc:        "Agent 狼人杀游戏 - 回合制多智能体狼人杀游戏",
}
