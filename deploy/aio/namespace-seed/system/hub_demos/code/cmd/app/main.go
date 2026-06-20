package main

import (
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/doc"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/doc/document"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/filetools"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/filetools/file"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/img"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/img/image"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/user_1210227080_coding_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/user_1210227080_coding_0_1_0/runtime"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/user_1210227080_table_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/user_1210227080_table_0_1_0/table"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/user_1210227080_text_0_1_0"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/user_1210227080_text_0_1_0/text"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/video_tools"
	_ "github.com/kageos/kageos/namespace/system/hub_demos/code/api/video_tools/video"
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
