package main

import (
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/archive"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/audio"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/chart"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/database"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/diagram"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/document"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/file"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/html"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/image"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/json"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/message"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/ocr"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/pdf"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/runtime"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/table"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/text"
	_ "github.com/kageos/kageos/namespace/system/tools/code/api/video"

	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
