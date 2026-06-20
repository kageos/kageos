package midnight_pub

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// StartDialogueReq 开启对话请求
type StartDialogueReq struct {
	Topic      string `json:"topic" widget:"name:话题;type:input"`
	Characters string `json:"characters" widget:"name:参与角色;type:input"`
}

// StartDialogueResp 开启对话响应
type StartDialogueResp struct {
	DialogueFragment string `json:"dialogue_fragment" widget:"name:对话片段;type:text_area"`
}

// dialogueTemplates 对话模板
var dialogueTemplates = []struct {
	topic string
	lines []string
}{
	{
		topic: "工作吐槽",
		lines: []string{
			"今天又修了一个祖传 bug，感觉自己像考古学家。",
			"加班到凌晨三点，结果早上八点又被叫醒。",
			"代码是能跑的，就是没人敢动它。",
			"产品说需求很简单，就是改一下整个架构。",
			"上线前：应该没问题。上线后：完了。",
		},
	},
	{
		topic: "理想",
		lines: []string{
			"我的理想是有天代码能自己写自己。",
			"那样你就可以专心写诗了。",
			"你会写诗？",
			"会，就是写得比代码还 bug 多。",
			"至少诗的 bug 不会影响生产环境。",
		},
	},
	{
		topic: "失眠",
		lines: []string{
			"又失眠了，今晚是第几次了？",
			"数羊数到羊都睡着了，我还是醒着。",
			"深夜的脑子最清醒，也最混乱。",
			"睡不着的时候，就来酒馆坐坐吧。",
			"至少这里有人陪。",
		},
	},
	{
		topic: "加班",
		lines: []string{
			"今天又加班到深夜，项目deadline快到了。",
			"你们程序员是不是都这样？",
			"不，我们还有周末...吧？",
			"算了，不说了，喝酒。",
			"来，干了这杯，明天继续。",
		},
	},
	{
		topic: "回忆",
		lines: []string{
			"三年前的这个时候，我还在那座城市。",
			"时间过得真快啊。",
			"是啊，有些人一转眼就再也见不到了。",
			"所以要珍惜当下，珍惜眼前人。",
			"来，再干一杯。",
		},
	},
	{
		topic: "星辰",
		lines: []string{
			"今晚的星星真亮。",
			"就像小时候在外婆家的院子里看到的那样。",
			"那时候没有手机，只有星空和蝉鸣。",
			"现在抬头看星空的机会越来越少了。",
			"所以更要珍惜这样的夜晚。",
		},
	},
}

// randomDialogueLines 随机生成对话
func generateRandomDialogue(topic, characters string) string {
	// 随机选择一个模板
	template := dialogueTemplates[rand.Intn(len(dialogueTemplates))]

	// 解析角色
	roles := strings.Split(characters, ",")
	if len(roles) < 2 {
		roles = []string{"程序员小李", "文艺青年阿诗"}
	}
	for i := range roles {
		roles[i] = strings.TrimSpace(roles[i])
		if roles[i] == "" {
			roles[i] = fmt.Sprintf("角色%d", i+1)
		}
	}

	// 生成对话
	var lines []string
	for i, line := range template.lines {
		role := roles[i%len(roles)]
		lines = append(lines, fmt.Sprintf("[%s] %s", role, line))
	}

	return strings.Join(lines, "\n")
}

func StartDialogueHandler(ctx *app.Context, resp response.Response) error {
	var req StartDialogueReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	rand.Seed(time.Now().UnixNano())

	// 如果没有指定话题或角色，随机生成
	topic := req.Topic
	characters := req.Characters

	if topic == "" {
		topics := []string{"工作吐槽", "理想", "失眠", "加班", "回忆", "星辰"}
		topic = topics[rand.Intn(len(topics))]
	}

	if characters == "" {
		characters = "程序员小李,文艺青年阿诗"
	}

	// 生成对话
	dialogue := generateRandomDialogue(topic, characters)

	// 保存对话记录
	db := ctx.GetGormDB()
	roles := strings.Split(characters, ",")
	for i, line := range strings.Split(dialogue, "\n") {
		if line == "" {
			continue
		}
		record := DialogueRecord{
			CharacterName: roles[i%len(roles)],
			CharacterCode: "guest",
			Content:       strings.TrimPrefix(line, "["+roles[i%len(roles)]+"] "),
			TopicTag:      topic,
			SpeakTime:     types.Time(time.Now()),
		}
		db.Create(&record)
	}

	// 增加角色出场次数
	for _, roleName := range roles {
		var character PubCharacter
		if err := db.Where("character_name = ?", strings.TrimSpace(roleName)).First(&character).Error; err == nil {
			db.Model(&character).Update("appear_count", character.AppearCount+1)
		}
	}

	return resp.Form(&StartDialogueResp{
		DialogueFragment: dialogue,
	}).Build()
}

func init() {
	packageContext.POST("start_dialogue.form", StartDialogueHandler, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "开启对话",
			Request:  &StartDialogueReq{},
			Response: &StartDialogueResp{},
		},
	})
}
