package service

import (
	"fmt"
	"strings"
)

type CardRenderer interface {
	Channel() string
	Render(card NotificationCard) (map[string]interface{}, error)
}

type FeishuCardRenderer struct{}

func (FeishuCardRenderer) Channel() string {
	return NotificationChannelFeishu
}

func (FeishuCardRenderer) Render(card NotificationCard) (map[string]interface{}, error) {
	title := firstNonEmptyString(card.Title, "Kageos 通知")
	primary := primaryNotificationAction(card.Actions)
	source := notificationActionByKind(card.Actions, NotificationActionSource)
	session := notificationActionByKind(card.Actions, NotificationActionSession)

	bodyElements := []interface{}{
		map[string]interface{}{
			"tag":        "markdown",
			"content":    fmt.Sprintf("**<font color='blue'>%s</font>**", escapeFeishuCardText(firstNonEmptyString(card.Summary, card.Title))),
			"text_size":  "normal",
			"margin":     "0px 0px 0px 0px",
			"element_id": "summary_display",
		},
		map[string]interface{}{
			"tag":        "markdown",
			"content":    fmt.Sprintf("<font color='grey'>来源目录：</font>%s", escapeFeishuCardText(firstNonEmptyString(card.Source.Title, card.Source.Path, card.Source.Workspace, "Kageos"))),
			"margin":     "0px 0px 0px 0px",
			"element_id": "source_title_display",
		},
	}
	if contextContent := renderFeishuContextMarkdown(card); contextContent != "" {
		bodyElements = append(bodyElements, map[string]interface{}{
			"tag":        "markdown",
			"content":    contextContent,
			"margin":     "0px 0px 0px 0px",
			"element_id": "context_display",
		})
	}
	bodyElements = append(bodyElements, map[string]interface{}{
		"tag":        "hr",
		"margin":     "0px 0px 0px 0px",
		"element_id": "content_divider",
	})
	if content := renderFeishuContentMarkdown(card); content != "" {
		bodyElements = append(bodyElements, map[string]interface{}{
			"tag":        "markdown",
			"content":    content,
			"text_size":  "normal",
			"margin":     "0px 0px 0px 0px",
			"element_id": "content_display",
		})
	}

	if columns := renderFeishuCard2ActionColumns(primary, source, session); len(columns) > 0 {
		bodyElements = append(bodyElements, map[string]interface{}{
			"tag":                "column_set",
			"flex_mode":          "stretch",
			"horizontal_spacing": "8px",
			"horizontal_align":   "left",
			"columns":            columns,
			"margin":             "0px 0px 0px 0px",
		})
	}
	bodyElements = append(bodyElements, map[string]interface{}{
		"tag":        "markdown",
		"content":    "<font color='grey' size='small'>完整内容已保存到 Kageos 站内信，可在工作空间中继续查看。</font>",
		"text_align": "center",
		"margin":     "0px 0px 0px 0px",
		"element_id": "inbox_footer",
	})

	cardPayload := map[string]interface{}{
		"schema": "2.0",
		"config": map[string]interface{}{"update_multi": true},
		"body": map[string]interface{}{
			"direction": "vertical",
			"elements":  bodyElements,
		},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
			"subtitle": map[string]interface{}{
				"tag":     "plain_text",
				"content": "",
			},
			"text_tag_list": []interface{}{
				map[string]interface{}{
					"tag": "text_tag",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": "Kageos 自动通知",
					},
					"color": "blue",
				},
				map[string]interface{}{
					"tag": "text_tag",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": notificationLevelLabel(card.Level),
					},
					"color": feishuLevelColor(card.Level),
				},
			},
			"template": "blue",
			"padding":  "12px 12px 12px 12px",
		},
	}
	if primary.URL != "" {
		cardPayload["card_link"] = map[string]interface{}{"url": primary.URL}
	}

	return map[string]interface{}{
		"msg_type": "interactive",
		"card":     cardPayload,
	}, nil
}

func renderFeishuContextMarkdown(card NotificationCard) string {
	lines := []string{}
	add := func(label, value string) {
		value = escapeFeishuCardText(value)
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("<font color='grey'>%s：</font>%s", label, value))
	}
	add("工作空间", card.Source.Workspace)
	add("目录路径", card.Source.Path)
	add("任务/会话", firstNonEmptyString(card.Task.SessionTitle, card.Task.Title))
	add("发起人", card.FromUser)
	if !card.CreatedAt.IsZero() {
		add("时间", card.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return strings.Join(lines, "\n")
}

func renderFeishuContentMarkdown(card NotificationCard) string {
	content := truncateRunes(stripNotificationMarkup(card.Content), 700)
	if content == "" || content == card.Summary {
		return ""
	}
	return fmt.Sprintf("<font color='grey'>具体内容：</font>\n%s", escapeFeishuCardText(content))
}

type WeComMarkdownRenderer struct{}

func (WeComMarkdownRenderer) Channel() string {
	return NotificationChannelWeCom
}

func (WeComMarkdownRenderer) Render(card NotificationCard) (map[string]interface{}, error) {
	content := renderWeComMarkdown(card)
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("wecom notification content is empty")
	}
	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": content,
		},
	}, nil
}

type WeComTemplateCardRenderer struct{}

func (WeComTemplateCardRenderer) Channel() string {
	return NotificationChannelWeCom
}

func (WeComTemplateCardRenderer) Render(card NotificationCard) (map[string]interface{}, error) {
	primary := primaryNotificationAction(card.Actions)
	if strings.TrimSpace(primary.URL) == "" {
		return WeComMarkdownRenderer{}.Render(card)
	}
	templateCard := map[string]interface{}{
		"card_type": "text_notice",
		"source": map[string]interface{}{
			"desc":       "Kageos 自动通知",
			"desc_color": wecomLevelDescColor(card.Level),
		},
		"main_title": map[string]interface{}{
			"title": firstNonEmptyString(card.Title, "Kageos 通知"),
			"desc":  firstNonEmptyString(card.Summary, "你有一条新的 Kageos 通知"),
		},
		"emphasis_content": map[string]interface{}{
			"title": notificationLevelLabel(card.Level),
			"desc":  "通知级别",
		},
		"horizontal_content_list": renderWeComHorizontalContent(card),
		"jump_list":               renderWeComJumpList(card.Actions),
		"card_action": map[string]interface{}{
			"type": 1,
			"url":  primary.URL,
		},
	}
	if quote := renderWeComQuoteArea(card); quote != nil {
		templateCard["quote_area"] = quote
	}
	return map[string]interface{}{
		"msgtype":       "template_card",
		"template_card": templateCard,
	}, nil
}

type DingTalkActionCardRenderer struct{}

func (DingTalkActionCardRenderer) Channel() string {
	return NotificationChannelDingTalk
}

func (DingTalkActionCardRenderer) Render(card NotificationCard) (map[string]interface{}, error) {
	title := firstNonEmptyString(card.Title, "Kageos 通知")
	content := renderDingTalkActionCardText(card)
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("dingtalk notification content is empty")
	}
	buttons := renderDingTalkActionButtons(card.Actions)
	if len(buttons) == 0 {
		return map[string]interface{}{
			"msgtype": "text",
			"text": map[string]interface{}{
				"content": content,
			},
		}, nil
	}
	return map[string]interface{}{
		"msgtype": "actionCard",
		"actionCard": map[string]interface{}{
			"title":          title,
			"text":           content,
			"btnOrientation": "0",
			"btns":           buttons,
		},
	}, nil
}

func renderFeishuCard2ActionColumns(primary, source, session NotificationAction) []interface{} {
	buttons := []struct {
		label     string
		url       string
		primary   bool
		elementID string
	}{
		{label: "查看详情", url: primary.URL, primary: true, elementID: "view_detail_action"},
		{label: "打开目录", url: source.URL, elementID: "open_source_action"},
		{label: "打开会话", url: session.URL, elementID: "open_session_action"},
	}
	columns := make([]interface{}, 0, len(buttons))
	for _, button := range buttons {
		if strings.TrimSpace(button.url) == "" {
			continue
		}
		buttonType := "default"
		if button.primary {
			buttonType = "primary_filled"
		}
		columns = append(columns, map[string]interface{}{
			"tag":   "column",
			"width": "auto",
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "button",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": button.label,
					},
					"type":  buttonType,
					"width": "fill",
					"behaviors": []interface{}{
						map[string]interface{}{
							"type":        "open_url",
							"default_url": button.url,
							"pc_url":      "",
							"ios_url":     "",
							"android_url": "",
						},
					},
					"margin":     "4px 0px 4px 0px",
					"element_id": button.elementID,
				},
			},
			"vertical_spacing": "8px",
			"horizontal_align": "left",
			"vertical_align":   "top",
		})
	}
	return columns
}

func renderWeComMarkdown(card NotificationCard) string {
	var lines []string
	title := firstNonEmptyString(card.Title, "Kageos 通知")
	lines = append(lines, fmt.Sprintf("**%s**", escapeCardMarkdown(title)))
	lines = append(lines, fmt.Sprintf(">级别：%s", notificationLevelLabel(card.Level)))
	if card.Summary != "" {
		lines = append(lines, fmt.Sprintf(">摘要：%s", escapeCardMarkdown(card.Summary)))
	}
	if card.Source.Workspace != "" {
		lines = append(lines, fmt.Sprintf(">工作空间：%s", escapeCardMarkdown(card.Source.Workspace)))
	}
	if card.Source.Title != "" {
		lines = append(lines, fmt.Sprintf(">来源目录：%s", escapeCardMarkdown(card.Source.Title)))
	}
	if card.Source.Path != "" {
		lines = append(lines, fmt.Sprintf(">目录路径：`%s`", escapeCardMarkdown(card.Source.Path)))
	}
	if taskTitle := firstNonEmptyString(card.Task.SessionTitle, card.Task.Title); taskTitle != "" {
		lines = append(lines, fmt.Sprintf(">任务/会话：%s", escapeCardMarkdown(taskTitle)))
	}
	if card.FromUser != "" {
		lines = append(lines, fmt.Sprintf(">发起人：%s", escapeCardMarkdown(card.FromUser)))
	}
	if !card.CreatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf(">时间：%s", card.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	if content := truncateRunes(stripNotificationMarkup(card.Content), 500); content != "" && content != card.Summary {
		lines = append(lines, "", escapeCardMarkdown(content))
	}
	if len(card.Actions) > 0 {
		lines = append(lines, "")
		for _, action := range card.Actions {
			if strings.TrimSpace(action.URL) == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("[%s](%s)", firstNonEmptyString(action.Label, "打开"), action.URL))
		}
	}
	lines = append(lines, "", "完整内容已保存到 Kageos 站内信。")
	return strings.Join(lines, "\n")
}

func renderWeComHorizontalContent(card NotificationCard) []interface{} {
	items := []interface{}{}
	add := func(key, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		items = append(items, map[string]interface{}{
			"keyname": key,
			"value":   truncateRunes(value, 120),
		})
	}
	add("来源目录", firstNonEmptyString(card.Source.Title, card.Source.Path))
	add("工作空间", card.Source.Workspace)
	add("任务/会话", firstNonEmptyString(card.Task.SessionTitle, card.Task.Title))
	add("发起人", card.FromUser)
	if !card.CreatedAt.IsZero() {
		add("时间", card.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return items
}

func renderWeComQuoteArea(card NotificationCard) map[string]interface{} {
	title := firstNonEmptyString(card.Source.Title, "来源目录")
	quote := firstNonEmptyString(card.Source.Path, card.Source.Workspace, card.Summary)
	if title == "" && quote == "" {
		return nil
	}
	area := map[string]interface{}{
		"type":       0,
		"title":      truncateRunes(title, 80),
		"quote_text": truncateRunes(quote, 180),
	}
	if source := notificationActionByKind(card.Actions, NotificationActionSource); strings.TrimSpace(source.URL) != "" {
		area["type"] = 1
		area["url"] = source.URL
	}
	return area
}

func renderWeComJumpList(actions []NotificationAction) []interface{} {
	list := make([]interface{}, 0, len(actions))
	for _, action := range actions {
		if strings.TrimSpace(action.URL) == "" {
			continue
		}
		list = append(list, map[string]interface{}{
			"type":  1,
			"title": firstNonEmptyString(action.Label, "打开"),
			"url":   strings.TrimSpace(action.URL),
		})
		if len(list) >= 3 {
			break
		}
	}
	return list
}

func wecomLevelDescColor(level string) int {
	switch level {
	case NotificationLevelCritical:
		return 1
	case NotificationLevelWarning:
		return 2
	default:
		return 0
	}
}

func renderDingTalkActionCardText(card NotificationCard) string {
	var lines []string
	title := firstNonEmptyString(card.Title, "Kageos 通知")
	lines = append(lines, "Kageos 自动通知")
	lines = append(lines, "")
	lines = append(lines, "标题："+plainDingTalkText(title))
	lines = append(lines, fmt.Sprintf("级别：%s", notificationLevelLabel(card.Level)))
	if card.Summary != "" {
		lines = append(lines, "摘要："+plainDingTalkText(card.Summary))
	}
	if card.Source.Workspace != "" {
		lines = append(lines, "工作空间："+plainDingTalkText(card.Source.Workspace))
	}
	if card.Source.Title != "" {
		lines = append(lines, "来源目录："+plainDingTalkText(card.Source.Title))
	}
	if card.Source.Path != "" {
		lines = append(lines, "目录路径："+plainDingTalkText(card.Source.Path))
	}
	if taskTitle := firstNonEmptyString(card.Task.SessionTitle, card.Task.Title); taskTitle != "" {
		lines = append(lines, "任务/会话："+plainDingTalkText(taskTitle))
	}
	if card.FromUser != "" {
		lines = append(lines, "发起人："+plainDingTalkText(card.FromUser))
	}
	if !card.CreatedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("时间：%s", card.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	if content := truncateRunes(stripNotificationMarkup(card.Content), 500); content != "" && content != card.Summary {
		lines = append(lines, "", "内容：", plainDingTalkText(content))
	}
	lines = append(lines, "", "完整内容已保存到 Kageos 站内信。")
	return strings.Join(lines, "\n\n")
}

func renderDingTalkActionButtons(actions []NotificationAction) []interface{} {
	buttons := make([]interface{}, 0, len(actions))
	for _, action := range actions {
		if strings.TrimSpace(action.URL) == "" {
			continue
		}
		buttons = append(buttons, map[string]interface{}{
			"title":     firstNonEmptyString(action.Label, "打开"),
			"actionURL": strings.TrimSpace(action.URL),
		})
		if len(buttons) >= 3 {
			break
		}
	}
	return buttons
}

func plainDingTalkText(s string) string {
	s = stripNotificationMarkup(s)
	replacer := strings.NewReplacer(
		"[", "",
		"]", "",
		"(", "（",
		")", "）",
		"---", "",
	)
	return strings.TrimSpace(replacer.Replace(s))
}

func feishuLevelColor(level string) string {
	switch level {
	case NotificationLevelCritical:
		return "red"
	case NotificationLevelWarning:
		return "orange"
	default:
		return "blue"
	}
}

func notificationLevelLabel(level string) string {
	switch level {
	case NotificationLevelCritical:
		return "高优先级"
	case NotificationLevelWarning:
		return "提醒"
	default:
		return "普通"
	}
}

func escapeCardMarkdown(s string) string {
	return strings.TrimSpace(s)
}

func escapeFeishuCardText(s string) string {
	s = strings.TrimSpace(s)
	replacer := strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(s)
}

func primaryNotificationAction(actions []NotificationAction) NotificationAction {
	for _, action := range actions {
		if action.Kind == NotificationActionDetail && strings.TrimSpace(action.URL) != "" {
			return action
		}
	}
	for _, action := range actions {
		if strings.TrimSpace(action.URL) != "" {
			return action
		}
	}
	return NotificationAction{}
}

func notificationActionByKind(actions []NotificationAction, kind string) NotificationAction {
	for _, action := range actions {
		if action.Kind == kind && strings.TrimSpace(action.URL) != "" {
			return action
		}
	}
	return NotificationAction{}
}
