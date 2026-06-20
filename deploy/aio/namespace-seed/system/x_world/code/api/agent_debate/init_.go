package agent_debate

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/agent_debate",
	Name:        "Agent辩论赛",
	Desc:        "两个AI角色围绕用户设定话题展开辩论，用户当裁判打分评选最佳辩手",
}
