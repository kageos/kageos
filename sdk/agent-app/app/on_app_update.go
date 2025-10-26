package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"os"
	"path/filepath"
	"time"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/model"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

// 获取API日志目录
func (a *App) getApiLogsDir() string {
	return "/app/workplace/api-logs"
}

// 获取当前版本的API文件路径
func (a *App) getCurrentVersionFile() string {
	return filepath.Join(a.getApiLogsDir(), fmt.Sprintf("%s.json", env.Version))
}

// 获取上一版本的API文件路径
func (a *App) getPreviousVersionFile() string {
	// 首先尝试直接推断上一版本号
	// 假设版本号格式为 v1, v2, v3...
	if len(env.Version) > 0 && env.Version[0] == 'v' {
		numStr := env.Version[1:]
		var current int
		if n, err := fmt.Sscanf(numStr, "%d", &current); err == nil && n == 1 {
			if current > 1 {
				prevVersion := fmt.Sprintf("v%d", current-1)
				prevFile := filepath.Join(a.getApiLogsDir(), prevVersion+".json")
				// 检查文件是否存在
				if _, err := os.Stat(prevFile); err == nil {
					return prevFile
				}
			}
		}
	}

	// 如果直接推断失败，再遍历目录查找上一版本
	apiLogsDir := a.getApiLogsDir()
	files, err := os.ReadDir(apiLogsDir)
	if err != nil {
		return ""
	}

	var maxVersion string
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		version := file.Name()[:len(file.Name())-5] // 去掉.json
		if version > maxVersion && version < env.Version {
			maxVersion = version
		}
	}

	if maxVersion != "" {
		return filepath.Join(apiLogsDir, maxVersion+".json")
	}

	return ""
}

// 保存当前版本的API信息到文件
func (a *App) saveCurrentVersion(apis []*model.ApiInfo) error {
	apiLogsDir := a.getApiLogsDir()

	// 创建目录
	if err := os.MkdirAll(apiLogsDir, 0755); err != nil {
		return fmt.Errorf("failed to create api logs directory: %w", err)
	}

	// 构建版本信息
	versionInfo := &model.ApiVersion{
		Version:   env.Version,
		Timestamp: time.Now(),
		Apis:      apis,
	}

	// 序列化
	data, err := json.MarshalIndent(versionInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version info: %w", err)
	}

	// 写入文件
	versionFile := a.getCurrentVersionFile()
	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

// 加载指定版本的API信息
func (a *App) loadVersion(versionFile string) ([]*model.ApiInfo, error) {
	if versionFile == "" {
		return []*model.ApiInfo{}, nil
	}

	data, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.ApiInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	var versionInfo model.ApiVersion
	if err := json.Unmarshal(data, &versionInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version info: %w", err)
	}

	return versionInfo.Apis, nil
}

// deepEqualStrings 比较字符串切片是否相等
func (a *App) deepEqualStrings(a1, a2 []string) bool {
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

// deepEqualInterfaces 比较interface{}是否相等（通过JSON序列化）
func (a *App) deepEqualInterfaces(a1, a2 interface{}) bool {
	if a1 == nil && a2 == nil {
		return true
	}
	if a1 == nil || a2 == nil {
		return false
	}

	j1, err1 := json.Marshal(a1)
	j2, err2 := json.Marshal(a2)

	if err1 != nil || err2 != nil {
		// 如果序列化失败，回退到直接比较
		return a1 == a2
	}

	return string(j1) == string(j2)
}

// 比较两个API信息是否相同
func (a *App) compareApis(oldApi, newApi *model.ApiInfo) bool {
	// 比较基本信息（排除版本信息，因为版本信息会自然变化）
	if oldApi.Name != newApi.Name ||
		oldApi.Desc != newApi.Desc ||
		!a.deepEqualStrings(oldApi.Tags, newApi.Tags) ||
		!a.deepEqualStrings(oldApi.CreateTables, newApi.CreateTables) {
		return false
	}

	// 比较请求参数
	if !a.compareFields(oldApi.Request, newApi.Request) {
		return false
	}

	// 比较响应参数
	if !a.compareFields(oldApi.Response, newApi.Response) {
		return false
	}

	return true
}

// 比较字段列表
func (a *App) compareFields(oldFields, newFields []*widget.Field) bool {
	if len(oldFields) != len(newFields) {
		return false
	}

	// 创建字段映射
	oldMap := make(map[string]*widget.Field)
	newMap := make(map[string]*widget.Field)

	for _, field := range oldFields {
		oldMap[field.Code] = field
	}

	for _, field := range newFields {
		newMap[field.Code] = field
	}

	// 比较每个字段
	for code, oldField := range oldMap {
		newField, exists := newMap[code]
		if !exists {
			return false
		}

		if !a.compareField(oldField, newField) {
			return false
		}
	}

	return true
}

// 比较单个字段
func (a *App) compareField(oldField, newField *widget.Field) bool {
	return oldField.Code == newField.Code &&
		oldField.Name == newField.Name &&
		oldField.Desc == newField.Desc &&
		oldField.Widget.Type == newField.Widget.Type &&
		a.deepEqualInterfaces(oldField.Widget.Config, newField.Widget.Config) &&
		oldField.Validation == newField.Validation
}

// 检查版本是否存在于版本列表中
func (a *App) containsVersion(versions []string, version string) bool {
	for _, v := range versions {
		if v == version {
			return true
		}
	}
	return false
}

// 生成API的唯一键
func (a *App) getApiKey(api *model.ApiInfo) string {
	return fmt.Sprintf("%s:%s", api.Method, api.Router)
}

// 执行API差异对比
func (a *App) diffApi() (add []*model.ApiInfo, update []*model.ApiInfo, delete []*model.ApiInfo, err error) {
	// 获取当前版本的API
	currentApis, _, err := a.getApis()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get current apis: %w", err)
	}

	// 加载上一版本的API
	previousVersionFile := a.getPreviousVersionFile()
	previousApis, err := a.loadVersion(previousVersionFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load previous version: %w", err)
	}

	// 创建API映射
	currentMap := make(map[string]*model.ApiInfo)
	previousMap := make(map[string]*model.ApiInfo)

	for _, api := range currentApis {
		key := a.getApiKey(api)
		currentMap[key] = api
	}

	for _, api := range previousApis {
		key := a.getApiKey(api)
		previousMap[key] = api
	}

	// 找出新增的API
	for key, currentApi := range currentMap {
		if _, exists := previousMap[key]; !exists {
			// 新增的API，设置AddedVersion为当前版本
			newApi := *currentApi
			newApi.AddedVersion = env.Version
			add = append(add, &newApi)
		}
	}

	// 找出删除的API
	for key, api := range previousMap {
		if _, exists := currentMap[key]; !exists {
			delete = append(delete, api)
		}
	}

	// 找出修改的API
	for key, currentApi := range currentMap {
		if previousApi, exists := previousMap[key]; exists {
			// 对于修改的API，保持原始的AddedVersion，并添加当前版本到UpdateVersions
			modifiedApi := *currentApi
			modifiedApi.AddedVersion = previousApi.AddedVersion

			// 复制原有的更新版本列表，并添加当前版本
			modifiedApi.UpdateVersions = make([]string, len(previousApi.UpdateVersions))
			copy(modifiedApi.UpdateVersions, previousApi.UpdateVersions)

			// 添加当前版本到更新列表（如果不存在的话）
			if !a.containsVersion(modifiedApi.UpdateVersions, env.Version) {
				modifiedApi.UpdateVersions = append(modifiedApi.UpdateVersions, env.Version)
			}

			if !a.compareApis(previousApi, &modifiedApi) {
				update = append(update, &modifiedApi)
			} else {
				// 如果API没有真正变更，不应该添加到update列表
				// 保持原来的UpdateVersions列表
				modifiedApi.UpdateVersions = previousApi.UpdateVersions
			}
		}
	}

	return add, update, delete, nil
}

// 获取当前所有API信息
func (a *App) getApis() (apis []*model.ApiInfo, createTables []interface{}, err error) {
	for _, info := range a.routerInfo {
		base := info.Template.GetBaseConfig()
		api := &model.ApiInfo{
			Code:           info.getCode(),
			Name:           base.Name,
			Desc:           base.Desc,
			Tags:           base.Tags,
			Router:         info.Router,
			Method:         info.Method,
			AddedVersion:   "",         // 不预设版本，让diff逻辑来正确设置
			UpdateVersions: []string{}, // 初始化空的更新版本列表
		}

		templateType := info.Template.TemplateType()
		api.TemplateType = string(templateType)
		if templateType == TemplateTypeTable {
			table := info.Template.(*TableTemplate).AutoCrudTable
			requestFields, responseFields, err := widget.DecodeTable(base.Request, table)
			if err != nil {
				return nil, nil, err
			}
			api.Request = requestFields
			api.Response = responseFields

		}
		if templateType == TemplateTypeForm {
			fields, responseFields, err := widget.DecodeForm(base.Request, base.Response)
			if err != nil {
				return nil, nil, err
			}
			api.Request = fields
			api.Response = responseFields
		}

		// 提取创建表的名称
		for _, createTable := range base.CreateTables {
			if createTable != nil {
				createTables = append(createTables, createTable)

				// 使用GORM的Tabler接口获取表名
				if tabler, ok := createTable.(interface{ TableName() string }); ok {
					api.CreateTables = append(api.CreateTables, tabler.TableName())
				}
			}
		}

		apis = append(apis, api)
	}
	return apis, createTables, nil
}

// onAppUpdate 处理当api更新时候触发
func (a *App) onAppUpdate(msg *nats.Msg) {

	logger.Infof(context.Background(), "OnAppUpdate received: %s", msg.Subject)
	// 1. 获取当前所有API
	currentApis, tables, err := a.getApis()
	if err != nil {
		// 发送错误响应
		a.sendErrorResponse(msg, fmt.Sprintf("Failed to get current APIs: %v", err))
		return
	}
	db := getGormDB()
	if db != nil {
		for _, table := range tables {
			err := db.AutoMigrate(table)
			if err != nil {
				a.sendErrorResponse(msg, fmt.Sprintf("Failed to migrate table: %v", err))
				return
			}
		}
	}

	// 2. 保存当前版本到API日志
	if err := a.saveCurrentVersion(currentApis); err != nil {
		// 发送错误响应
		a.sendErrorResponse(msg, fmt.Sprintf("Failed to save current version: %v", err))
		return
	}

	// 3. 执行API差异对比
	add, update, delete, err := a.diffApi()
	if err != nil {
		// 发送错误响应
		a.sendErrorResponse(msg, fmt.Sprintf("Failed to diff APIs: %v", err))
		return
	}

	// 4. 构建差异结果
	diffData := &model.DiffData{
		Add:    add,
		Update: update,
		Delete: delete,
	}
	rsp := subjects.Message{
		Data: diffData,
	}

	// 5. 发送成功响应
	//a.sendSuccessResponse(msg, diffData)
	msgx.RespSuccessMsg(msg, rsp)
}

// 发送成功响应 - 使用原请求消息直接响应
func (a *App) sendSuccessResponse(msg *nats.Msg, data *model.DiffData) {
	//response := &model.UpdateResponse{
	//	Status:    "success",
	//	Message:   message,
	//	Data:      data,
	//	Version:   env.Version,
	//	Timestamp: time.Now(),
	//}

	rsp := subjects.Message{
		Type:      subjects.MessageTypeStatusOnAppUpdate,
		Data:      data,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Timestamp: time.Now(),
	}

	responseData, _ := json.Marshal(rsp)

	// 直接响应原请求消息
	if responseData != nil {
		// 创建新的响应消息
		responseMsg := nats.NewMsg(msg.Subject)
		responseMsg.Header = msg.Header
		responseMsg.Data = responseData
		msg.RespondMsg(responseMsg)
	}
}

// 发送错误响应
func (a *App) sendErrorResponse(msg *nats.Msg, message string) {
	rsp := subjects.Message{
		ErrorMsg:  message,
		Type:      subjects.MessageTypeStatusOnAppUpdate,
		Data:      nil,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Timestamp: time.Now(),
	}

	responseData, _ := json.Marshal(rsp)

	// 直接响应原请求消息
	if responseData != nil {
		// 创建新的响应消息
		responseMsg := nats.NewMsg(msg.Subject)
		responseMsg.Header = msg.Header
		responseMsg.Data = responseData
		msg.RespondMsg(responseMsg)
	}
}

// 发送错误响应

// getGormDB 获取数据库连接
// 注意：这里需要根据实际的App结构来实现
// 如果App有数据库连接的字段或方法，需要相应修改
func (a *App) getGormDB() *gorm.DB {
	return getGormDB()
}
