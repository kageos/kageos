package callback

import (
	"context"
	"encoding/json"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

type OnTableAddRowReq struct {
	Body interface{} `json:"body"`
}

type OnTableAddRowResp struct {
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
	ID        int                    `json:"id"`
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

// GetOldValues 获取旧值（用于审计）
func (c *OnTableUpdateRowReq) GetOldValues() map[string]interface{} {
	if c.OldValues == nil {
		return make(map[string]interface{})
	}
	return c.OldValues
}

type OnTableUpdateRowResp struct {
}
