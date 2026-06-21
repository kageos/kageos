package main

import (
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/audio_toolkit_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/audio_toolkit_0_1_0/audio"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/database_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/database_0_1_0/database"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/graph_viz_tool_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/graph_viz_tool_0_1_0/diagram"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/html_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/html_0_1_0/html"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/meeting_room_management_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/meeting_room_management_0_1_0/meeting"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/nps_survey_management_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/nps_survey_management_0_1_0/nps"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/printdrop_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demo_user_1210227080/code/api/printdrop_0_1_0/printdrop"
	"github.com/kageos/kageos-sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
