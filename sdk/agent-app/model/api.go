package model

import (
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/jsonx"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
	"strings"
	"time"
)

// ApiInfo API信息结构
type ApiInfo struct {
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Desc           string          `json:"desc"`
	Tags           []string        `json:"tags"`
	Router         string          `json:"router"`
	Method         string          `json:"method"`
	CreateTables   []string        `json:"create_tables"`
	Callback       []string        `json:"callback"`
	Request        []*widget.Field `json:"request"`
	Response       []*widget.Field `json:"response"`
	AddedVersion   string          `json:"added_version"`   // API首次添加的版本
	UpdateVersions []string        `json:"update_versions"` // API更新过的版本列表
	TemplateType   string          `json:"template_type"`
	User           string          `json:"user"`
	App            string          `json:"app"`
	FullCodePath   string          `json:"full_code_path"`
}

func (a *ApiInfo) BuildFullCodePath() string {
	router := strings.Trim(a.Router, "/")
	if router == "" {
		return fmt.Sprintf("/%s/%s", a.User, a.App)
	}
	return fmt.Sprintf("/%s/%s/%s", a.User, a.App, router)
}
func (a *ApiInfo) GetParentFullCodePath() string {
	if a.FullCodePath == "" {
		return ""
	}

	// 去掉末尾的斜杠并分割路径
	pathParts := strings.Split(strings.Trim(a.FullCodePath, "/"), "/")
	if len(pathParts) <= 1 {
		return ""
	}

	// 返回父级路径（去掉最后一个部分）
	parentParts := pathParts[:len(pathParts)-1]
	if len(parentParts) == 0 {
		return ""
	}
	return "/" + strings.Join(parentParts, "/")
}

// GetAppPrefix 获取应用前缀
func (a *ApiInfo) GetAppPrefix() string {
	return fmt.Sprintf("/%s/%s", a.User, a.App)
}

// GetRelativePath 获取相对于应用根目录的路径
func (a *ApiInfo) GetRelativePath() string {
	if a.FullCodePath == "" {
		return ""
	}

	appPrefix := a.GetAppPrefix()
	if strings.HasPrefix(a.FullCodePath, appPrefix) {
		return strings.TrimPrefix(a.FullCodePath, appPrefix)
	}
	return a.FullCodePath
}

// GetFunctionName 获取函数名称（从code字段获取，底层从路由最后一个部分赋值）
func (a *ApiInfo) GetFunctionName() string {
	return a.Code
}

// GetPackagePath 获取包路径（不包含函数名）
func (a *ApiInfo) GetPackagePath() string {
	parentPath := a.GetParentFullCodePath()
	appPrefix := a.GetAppPrefix()

	// 如果父级路径就是应用根目录，返回应用前缀
	if parentPath == "" || parentPath == appPrefix {
		return appPrefix
	}

	return parentPath
}

// GetPackageChain 获取包链（从应用到函数的各级包名）
func (a *ApiInfo) GetPackageChain() []string {
	relativePath := a.GetRelativePath()
	if relativePath == "" || relativePath == "/" {
		return []string{}
	}

	parts := strings.Split(strings.Trim(relativePath, "/"), "/")
	if len(parts) <= 1 {
		return []string{}
	}

	// 排除最后一个函数名，只返回包链
	return parts[:len(parts)-1]
}

// IsEqual 比较当前API与另一个API是否相等（排除版本信息）
// 比较的字段包括：Name, Desc, Tags, CreateTables, Request, Response
func (a *ApiInfo) IsEqual(other *ApiInfo) bool {
	if other == nil {
		return false
	}

	// 比较基本信息（排除版本信息，因为版本信息会自然变化）
	if a.Name != other.Name ||
		a.Desc != other.Desc ||
		!equalStrings(a.Tags, other.Tags) ||
		!equalStrings(a.CreateTables, other.CreateTables) {
		return false
	}

	// 比较请求参数和响应参数
	return jsonx.DeepEqual(a.Request, other.Request) &&
		jsonx.DeepEqual(a.Response, other.Response)
}

// equalStrings 比较两个字符串切片是否相等
func equalStrings(a1, a2 []string) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i, v := range a1 {
		if a2[i] != v {
			return false
		}
	}
	return true
}

// ApiVersion 版本化的API信息
type ApiVersion struct {
	Version   string     `json:"version"`
	Timestamp time.Time  `json:"timestamp"`
	Apis      []*ApiInfo `json:"apis"`
}
