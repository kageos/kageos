package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

type OnTableAddRowReq struct {
	Body interface{} `json:"body"`
}

type OnTableAddRowResp struct {
	Data interface{} `json:"data"`
}

type OnTableDeleteRowsReq struct {
	Ids []int `json:"ids"`
}

func (c *OnTableDeleteRowsReq) GetIds() []int {
	return c.Ids
}

type OnTableDeleteRowsResp struct {
}
type OnTableUpdateRowReq struct {
	ID             int                    `json:"id"`
	BindUpdatesMap map[string]interface{} `json:"bind_updates_map"` //原先的值，Updates会经过处理，为啥？因为例如附件这种json字段，Updates更新时候需要转换成字符串，但是我们通过BindUpdates的时候如果转换成字符串会导致这个字段无法序列化到结构体，所以我们BindUpdatesMap保存备份用于绑定结构体

	Updates   map[string]interface{} `json:"updates"`
	OldValues map[string]interface{} `json:"old_values"`
}

func (c *OnTableUpdateRowReq) GetId() int {
	// ⚠️ 关键：ID 现在由前端直接传递，不再从 Updates 中获取
	// 保持向后兼容：如果 ID 为 0，尝试从 Updates 中获取（兼容旧版本）
	if c.ID != 0 {
		return c.ID
	}
	// 如果既没有 ID 字段，Updates 中也没有 id，返回 0（由业务层处理错误）
	return 0
}

// GetUpdates 获取更新字段（只包含变更的字段）
// ⚠️ 注意：Updates 中可以包含 id（虽然 id 不会被真正更新，但保持数据结构一致性）
// 后端在使用 Updates 进行数据库更新时，GORM 的 Updates 方法会自动忽略 id 字段
// 如果业务代码需要显式排除 id，可以使用 Omit("id") 方法
func (c *OnTableUpdateRowReq) GetUpdates() map[string]interface{} {
	if c.Updates == nil {
		return make(map[string]interface{})
	}

	// 处理文件类型组件，将文件数据序列化为 JSON 字符串
	for k, v := range c.Updates {
		switch v.(type) {
		case map[string]interface{}: //说明是文件类型的组件，
			widgetType, ok := v.(map[string]interface{})["widget_type"]
			if !ok {
				continue
			}
			switch widgetType.(type) {
			case string:
				if widgetType == widget.TypeFiles {
					marshal, err := json.Marshal(v)
					if err != nil {
						logger.Infof(context.Background(), "Marshal failed: %v", err)
						continue
					}
					c.Updates[k] = string(marshal)
				}
			}
		}
	}

	// ⚠️ 不再移除 id 字段，因为：
	// 1. 保持数据结构一致性（所有变更字段都在 Updates 中）
	// 2. GORM 的 Updates 方法会自动忽略 id 字段，不会真正更新 id
	// 3. 如果业务代码需要排除 id，可以使用 Omit("id") 方法

	return c.Updates
}

// IsFieldUpdated 判断指定字段是否在此次更新中被变更
//
// 这是一个快捷方法，用于替代 `if _, hasField := updates["field"]; hasField` 的写法
//
// 示例：
//
//	if req.IsFieldUpdated("quantity") {
//	    // quantity 字段在此次更新中被变更
//	}
//	if req.IsFieldUpdated("unit_price") {
//	    // unit_price 字段在此次更新中被变更
//	}
func (c *OnTableUpdateRowReq) IsFieldUpdated(fieldName string) bool {
	if c.Updates == nil {
		return false
	}
	_, exists := c.Updates[fieldName]
	return exists
}

// GetOldValues 获取旧值（用于审计）
func (c *OnTableUpdateRowReq) GetOldValues() map[string]interface{} {
	if c.OldValues == nil {
		return make(map[string]interface{})
	}
	return c.OldValues
}

// BindUpdates 将 Updates map 绑定到目标结构体
//
// ⚠️ 重要说明：
//   - Updates 只包含此次更新中变更的字段，未更新的字段不在 Updates 中
//   - 绑定后，目标结构体中只有更新的字段有值，未更新的字段为零值
//   - 如果需要访问未更新的字段，应该从数据库中查询当前记录
//
// 使用 JSON 序列化/反序列化的方式，确保类型正确转换
//
// 示例：
//
//	var updateFields CrmMeetingRoom
//	if err := req.BindUpdates(&updateFields); err != nil {
//	    return nil, err
//	}
//	// 此时 updateFields 中只有更新的字段有值，例如：
//	// 如果只更新了 name，则 updateFields.Name 有值，其他字段为零值
func (c *OnTableUpdateRowReq) BindUpdates(target interface{}) error {
	if c.BindUpdatesMap == nil || len(c.BindUpdatesMap) == 0 {
		return nil
	}

	// 将 map 序列化为 JSON
	jsonData, err := json.Marshal(c.BindUpdatesMap)
	if err != nil {
		return fmt.Errorf("序列化 updates 失败: %w", err)
	}

	// 反序列化到目标结构体
	if err := json.Unmarshal(jsonData, target); err != nil {
		logger.Infof(context.Background(), "Unmarshal failed: %v jsonData ：%s", err, jsonData)
		return fmt.Errorf("反序列化到目标结构体失败: %w", err)
	}

	return nil
}

type OnTableUpdateRowResp struct {
}

// OnTableCreateInBatchesReq 批量创建请求
type OnTableCreateInBatchesReq struct {
	Data []map[string]interface{} `json:"data"` // 批量数据数组
}

// OnTableCreateInBatchesResp 批量创建响应
type OnTableCreateInBatchesResp struct {
	SuccessCount int                      `json:"success_count"` // 成功数量
	FailCount    int                      `json:"fail_count"`    // 失败数量
	Errors       []OnTableCreateBatchError `json:"errors"`       // 错误详情
}

// OnTableCreateBatchError 批量创建错误信息
type OnTableCreateBatchError struct {
	Index int    `json:"index"` // 数据索引（从0开始）
	Error string `json:"error"` // 错误信息
}
