package audio

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/audio_toolkit_0_1_0/audio",
	Name:        "音频工具",
	Desc:        "基于 FFmpeg/FFprobe 的音频提取、转码、裁剪、信息读取和波形图生成能力，适合处理录音、播客、视频音轨和常见音频格式转换。",
}
