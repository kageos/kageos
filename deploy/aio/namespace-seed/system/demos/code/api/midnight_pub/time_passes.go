package midnight_pub

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// TimePassesReq 时间流逝请求
type TimePassesReq struct {
	TimeElapsed string `json:"time_elapsed" widget:"name:流逝时间;type:select;options:30分钟,1小时,2小时,3小时;options_colors:909399,409EFF,FF9800,9C27B0" validate:"required"`
}

// TimePassesResp 时间流逝响应
type TimePassesResp struct {
	NewTime           string `json:"new_time" widget:"name:新时间;type:input"`
	AtmosphereChange  string `json:"atmosphere_change" widget:"name:氛围变化;type:input"`
	LeftCharacters    string `json:"left_characters" widget:"name:离场角色;type:input"`
	EnteredCharacters string `json:"entered_characters" widget:"name:入场角色;type:input"`
	NewTopic          string `json:"new_topic" widget:"name:新话题;type:input"`
}

// parseTimeOffset 解析时间偏移
func parseTimeOffset(elapsed string) int {
	switch elapsed {
	case "30分钟":
		return 30
	case "1小时":
		return 60
	case "2小时":
		return 120
	case "3小时":
		return 180
	default:
		return 60
	}
}

// atmosphereChanges 氛围变化配置
var atmosphereChanges = map[int]string{
	23: "深夜的寂静开始蔓延，酒馆里多了几分沉思。",
	24: "午夜时分，空气中弥漫着微醺的气息。",
	1:  "凌晨的寒意渐浓，心事也越发清晰。",
	2:  "深夜两点，神秘的气息在酒馆里游荡。",
	3:  "凌晨三点，一切都安静得能听见心跳。",
	4:  "黎明前最黑暗的时刻，也是最真实的时刻。",
}

// newTopics 新话题配置
var newTopics = []string{
	"失眠、回忆、星辰",
	"理想、未来、选择",
	"往事、故人、遗憾",
	"当下、此刻、微醺",
	"人生、感悟、释然",
}

// enteredCharacters 入场角色
var possibleNewcomers = []string{
	"神秘人",
	"夜猫子老王",
	"失眠诗人",
	"深夜美食家",
	"怀旧的旅人",
}

// leftCharacters 离场角色
var possibleLeavers = []string{
	"程序员小李",
	"文艺青年阿诗",
	"加班狂人",
	"资深失眠者",
}

func TimePassesHandler(ctx *app.Context, resp response.Response) error {
	var req TimePassesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 获取当前酒馆状态
	var currentStatus PubStatus
	if err := db.Last(&currentStatus).Error; err != nil {
		// 如果没有记录，创建一个默认的
		currentStatus = PubStatus{
			LateNightLevel:   "22:00",
			AtmosphereTag:    "开场",
			PopularityIndex:  50,
			ActiveCharacters: "调酒师",
			HotTopic:         "开场、寒暄",
		}
		db.Create(&currentStatus)
	}

	// 解析当前时间
	currentTimeStr := currentStatus.LateNightLevel
	var hour, minute int
	fmt.Sscanf(currentTimeStr, "%d:%d", &hour, &minute)

	// 计算新时间
	minutesElapsed := parseTimeOffset(req.TimeElapsed)
	totalMinutes := hour*60 + minute + minutesElapsed
	newHour := (totalMinutes / 60) % 24
	if newHour < 22 {
		newHour += 24 // 确保时间在酒馆营业时间内
	}
	newTimeStr := fmt.Sprintf("%02d:%02d", newHour, minute)

	// 计算氛围变化
	atmosphereChange := atmosphereChanges[newHour]
	if atmosphereChange == "" {
		atmosphereChange = "时间在酒馆里静静流淌。"
	}

	// 随机离场和入场角色
	var leftChars, enteredChars []string
	currentChars := strings.Split(currentStatus.ActiveCharacters, "、")

	if newHour >= 2 && rand.Float32() < 0.3 {
		// 深夜2点后可能触发神秘人登场
		enteredChars = append(enteredChars, "神秘人")
	}

	// 随机决定谁离场
	for _, char := range currentChars {
		if char != "调酒师" && rand.Float32() < 0.4 {
			leftChars = append(leftChars, char)
		}
	}

	// 如果有人离场，也随机加入新角色
	if len(leftChars) > 0 && len(enteredChars) == 0 {
		newcomer := possibleNewcomers[rand.Intn(len(possibleNewcomers))]
		enteredChars = append(enteredChars, newcomer)
	}

	// 更新活跃角色
	newActiveChars := currentStatus.ActiveCharacters
	for _, left := range leftChars {
		newActiveChars = strings.ReplaceAll(newActiveChars, left+",", "")
		newActiveChars = strings.ReplaceAll(newActiveChars, left, "")
		newActiveChars = strings.ReplaceAll(newActiveChars, "、、", "、")
	}
	for _, entered := range enteredChars {
		if !strings.Contains(newActiveChars, entered) {
			newActiveChars += "、" + entered
		}
	}
	newActiveChars = strings.Trim(newActiveChars, "、")

	// 选择新话题
	newTopic := newTopics[rand.Intn(len(newTopics))]

	// 更新酒馆状态
	updates := map[string]interface{}{
		"late_night_level":  newTimeStr,
		"atmosphere_tag":    atmosphereChange,
		"active_characters": newActiveChars,
		"hot_topic":         newTopic,
	}

	if len(leftChars) > 0 {
		updates["popularity_index"] = currentStatus.PopularityIndex - len(leftChars)*5 + len(enteredChars)*5
	}

	db.Model(&currentStatus).Updates(updates)

	return resp.Form(&TimePassesResp{
		NewTime:           newTimeStr,
		AtmosphereChange:  atmosphereChange,
		LeftCharacters:    strings.Join(leftChars, "、"),
		EnteredCharacters: strings.Join(enteredChars, "、"),
		NewTopic:          newTopic,
	}).Build()
}

func init() {
	packageContext.POST("time_passes.form", TimePassesHandler, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "时间流逝",
			Request:  &TimePassesReq{},
			Response: &TimePassesResp{},
		},
	})
}
