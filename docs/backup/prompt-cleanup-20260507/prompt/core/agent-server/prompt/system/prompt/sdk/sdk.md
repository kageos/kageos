
首先介绍一下agent-app的sdk

agent-app sdk是ai-agent-os系统的一个应用开发sdk，这个sdk的特殊之处在于我们的
ai-agent-os 是一个以go的package为目录，go文件处理函数为api（前端的表现是一个表单（form）表格（table）图表（chart））的组织形式
我们只要在工作台通过write_go_file 写入满足我们agent-app sdk的代码，然后我们编译后自动会把对应的应用模块渲染到我们的前端页面，我们无需写任何前端代码
因为我们的sdk内部自动把我们的前端的展示逻辑给包含了，主要的逻辑是：

例如我们的form其实就是一个输入对应一个输出，
```go

type ConcatVideoReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:20" validate:"required"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，例如 merged_video.mp4"`
	Resolution     string `json:"resolution" widget:"name:输出分辨率;type:select;options:自动跟第一个视频一致,1080p(1920x1080),720p(1280x720),480p(854x480);options_colors:409EFF,67C23A,909399,E6A23C;render_default:自动跟第一个视频一致"`
}

type ConcatVideoResp struct {
	OutputFiles string `json:"output_files" widget:"name:拼接后的视频;type:files"`
	ConcatInfo  string `json:"concat_info" widget:"name:拼接信息;type:text"`
}


```

可以看到，我们可以通过写标签来满足我们的页面展示形式，这个在前端的展示逻辑是这样的
输入字段
    上传视频文件 （上传组件）
    输出文件名（输入框）
    输出分辨率（静态下拉选择器有后面的几个选项：自动跟第一个视频一致,1080p(1920x1080),720p(1280x720),480p(854x480)）

            提交按钮
输出字段
    拼接后的视频（文件展示）
    拼接信息（纯文本组件）

逻辑就是我们可以在输入框框输入数据，然后点击提交我们会返回我们的输出字段的数据，然后字段渲染到我们的页面上懂吗





然后是我们的table

table和我们的form的组件系统是完全一致的，只不过table是这样的
