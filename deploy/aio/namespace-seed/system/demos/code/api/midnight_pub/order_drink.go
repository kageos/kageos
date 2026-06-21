package midnight_pub

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// OrderDrinkReq 点一杯酒请求
type OrderDrinkReq struct {
	CharacterName string `json:"character_name" widget:"name:角色名;type:input" validate:"required"`
	DrinkName     string `json:"drink_name" widget:"name:酒名;type:select;options:whiskey_neat,whiskey_sour,martini,beer,mojito,old_fashioned;options_colors:E6A23C,FF9800,9C27B0,FFC107,4CAF50,795548" validate:"required"`
	Mood          string `json:"mood" widget:"name:心情;type:select;options:放松,感慨,忧伤,愉悦,沉思;options_colors:4CAF50,FF9800,9C27B0,FFC107,607D8B"`
	Message       string `json:"message" widget:"name:配文;type:text_area"`
}

// OrderDrinkResp 点一杯酒响应
type OrderDrinkResp struct {
	Bartender   string `json:"bartender" widget:"name:调酒师;type:input"`
	DrinkServed string `json:"drink_served" widget:"name:上酒;type:input"`
	DrinkLine   string `json:"drink_line" widget:"name:配酒台词;type:text_area"`
}

// drinkMenu 酒单配置
var drinkMenu = map[string]string{
	"whiskey_neat":  "纯威士忌",
	"whiskey_sour":  "威士忌酸",
	"martini":       "马天尼",
	"beer":          "啤酒",
	"mojito":        "莫吉托",
	"old_fashioned": "古典鸡尾酒",
}

// bartenderLines 调酒师台词
var bartenderLines = map[string]map[string]string{
	"放松": {
		"whiskey_neat":  "好的，一杯纯威士忌。今晚的夜色正好，配得上这份悠闲。",
		"whiskey_sour":  "好的，一杯威士忌酸。酸甜的味道就像你此刻轻松的心情。",
		"martini":       "好的，一杯马天尼。清爽利落，正是放松的好选择。",
		"beer":          "好的，一杯冰啤酒。泡沫丰富，一口下去满是畅快。",
		"mojito":        "好的，一杯莫吉托。青柠薄荷的清香，就像你的心情一样清新。",
		"old_fashioned": "好的，一杯古典鸡尾酒。经典的配方，经典的夜晚。",
	},
	"感慨": {
		"whiskey_neat":  "好的，一杯纯威士忌。今晚的烈度刚好，就像沉淀已久的情绪。",
		"whiskey_sour":  "好的，一杯威士忌酸。酸中带甜，感慨万千。",
		"martini":       "好的，一杯马天尼。有时候，一杯马天尼就足够诉说往事。",
		"beer":          "好的，一杯啤酒。有些话，就像泡沫一样，说出来就散了。",
		"mojito":        "好的，一杯莫吉托。青涩的回忆，总是这样清爽又带着点苦涩。",
		"old_fashioned": "好的，一杯古典鸡尾酒。往事如酒，越陈越香。",
	},
	"忧伤": {
		"whiskey_neat":  "好的，一杯纯威士忌。有时候烈酒比言语更能抚慰人心。",
		"whiskey_sour":  "好的，一杯威士忌酸。愿这酸涩能带走一些不愉快。",
		"martini":       "好的，一杯马天尼。有些心事，只有在深夜才敢拿出来。",
		"beer":          "好的，一杯啤酒。酒精度不高，却足够温暖。",
		"mojito":        "好的，一杯莫吉托。青柠的酸，也许刚好中和这份忧伤。",
		"old_fashioned": "好的，一杯古典鸡尾酒。苦尽甘来，忧伤总会过去的。",
	},
	"愉悦": {
		"whiskey_neat":  "好的，一杯纯威士忌。看你这么开心，这杯酒都变得更醇厚了。",
		"whiskey_sour":  "好的，一杯威士忌酸。愉悦的心情配酸甜的酒，恰到好处。",
		"martini":       "好的，一杯马天尼。今晚值得庆祝！",
		"beer":          "好的，一杯啤酒！朋友，干了这杯快乐！",
		"mojito":        "好的，一杯莫吉托！清新甜蜜，就像你的笑容。",
		"old_fashioned": "好的，一杯古典鸡尾酒。经典的快乐配方。",
	},
	"沉思": {
		"whiskey_neat":  "好的，一杯纯威士忌。安静的酒，适合安静的思考。",
		"whiskey_sour":  "好的，一杯威士忌酸。在思考中品味，在品味中思考。",
		"martini":       "好的，一杯马天尼。有时候答案就藏在这杯酒里。",
		"beer":          "好的，一杯啤酒。慢慢喝，让思绪沉淀。",
		"mojito":        "好的，一杯莫吉托。清新的味道，让思绪更加清晰。",
		"old_fashioned": "好的，一杯古典鸡尾酒。过去的智慧，往往能照亮前路。",
	},
}

func getBartenderLine(drink, mood string) string {
	if moodLines, ok := bartenderLines[mood]; ok {
		if line, ok := moodLines[drink]; ok {
			return line
		}
	}
	return fmt.Sprintf("好的，一杯 %s。这杯酒，敬这个深夜。", drinkMenu[drink])
}

func OrderDrinkHandler(ctx *app.Context, resp response.Response) error {
	var req OrderDrinkReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	// 获取酒馆角色出场次数并+1
	db := ctx.GetGormDB()
	var character PubCharacter
	if err := db.Where("character_name = ?", req.CharacterName).First(&character).Error; err == nil {
		db.Model(&character).Update("appear_count", character.AppearCount+1)
	}

	// 记录点单
	orderRecord := OrderRecord{
		CharacterName: req.CharacterName,
		DrinkName:     req.DrinkName,
		Mood:          req.Mood,
		Message:       req.Message,
	}
	if err := db.Create(&orderRecord).Error; err != nil {
		logger.Errorf(ctx, "OrderDrink Create OrderRecord err: %v", err)
	}

	// 生成调酒师台词
	drinkCn := drinkMenu[req.DrinkName]
	if drinkCn == "" {
		drinkCn = req.DrinkName
	}

	mood := req.Mood
	if mood == "" {
		mood = "放松"
	}
	bartenderLine := getBartenderLine(req.DrinkName, mood)

	if req.Message != "" {
		bartenderLine += fmt.Sprintf("\n\n%s 说：%s", req.CharacterName, req.Message)
	}

	return resp.Form(&OrderDrinkResp{
		Bartender:   "调酒师",
		DrinkServed: drinkCn,
		DrinkLine:   bartenderLine,
	}).Build()
}

func init() {
	packageContext.POST("order_drink.form", OrderDrinkHandler, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "点一杯酒",
			Request:  &OrderDrinkReq{},
			Response: &OrderDrinkResp{},
		},
	})
}
