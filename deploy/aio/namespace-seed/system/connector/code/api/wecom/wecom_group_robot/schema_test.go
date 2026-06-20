package wecom_group_robot

import (
	"testing"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/widget"
)

func TestWeComGroupRobotTemplatesDecodeSchema(t *testing.T) {
	formTemplates := map[string]*app.FormTemplate{
		"WeComGroupRobotConfigSaveTemplate":         WeComGroupRobotConfigSaveTemplate,
		"WeComGroupRobotStatusTemplate":             WeComGroupRobotStatusTemplate,
		"WeComGroupRobotSendTextTemplate":           WeComGroupRobotSendTextTemplate,
		"WeComGroupRobotSendMarkdownTemplate":       WeComGroupRobotSendMarkdownTemplate,
		"WeComGroupRobotSendMarkdownV2Template":     WeComGroupRobotSendMarkdownV2Template,
		"WeComGroupRobotSendImageTemplate":          WeComGroupRobotSendImageTemplate,
		"WeComGroupRobotSendNewsTemplate":           WeComGroupRobotSendNewsTemplate,
		"WeComGroupRobotUploadMediaTemplate":        WeComGroupRobotUploadMediaTemplate,
		"WeComGroupRobotSendFileTemplate":           WeComGroupRobotSendFileTemplate,
		"WeComGroupRobotSendVoiceTemplate":          WeComGroupRobotSendVoiceTemplate,
		"WeComGroupRobotSendTextNoticeCardTemplate": WeComGroupRobotSendTextNoticeCardTemplate,
		"WeComGroupRobotSendNewsNoticeCardTemplate": WeComGroupRobotSendNewsNoticeCardTemplate,
		"WeComGroupRobotSendRawJSONTemplate":        WeComGroupRobotSendRawJSONTemplate,
	}
	for name, template := range formTemplates {
		if _, _, err := widget.DecodeForm(groupRobotTemplateCallbacks(template.OnSelectFuzzyMap), template.Request, template.Response); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}

	tableTemplates := map[string]*app.TableTemplate{
		"WeComGroupRobotConfigListTemplate": WeComGroupRobotConfigListTemplate,
	}
	for name, template := range tableTemplates {
		if _, _, err := widget.DecodeTable(groupRobotTemplateCallbacks(template.OnSelectFuzzyMap), template.Request, template.AutoCrudTable); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}
}

func TestGroupRobotSecretEncryptDecrypt(t *testing.T) {
	plain := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-secret"
	cipherText, err := encryptGroupRobotSecret(plain)
	if err != nil {
		t.Fatalf("encryptGroupRobotSecret error: %v", err)
	}
	if cipherText == plain {
		t.Fatalf("webhook should not be stored as plaintext")
	}
	got, err := decryptGroupRobotSecret(cipherText)
	if err != nil {
		t.Fatalf("decryptGroupRobotSecret error: %v", err)
	}
	if got != plain {
		t.Fatalf("decrypted webhook = %q, want %q", got, plain)
	}
}

func groupRobotTemplateCallbacks(fuzzy map[string]app.OnSelectFuzzy) map[string][]string {
	if len(fuzzy) == 0 {
		return nil
	}
	callbacks := make(map[string][]string, len(fuzzy))
	for field := range fuzzy {
		callbacks[field] = []string{app.CallbackTypeOnSelectFuzzy}
	}
	return callbacks
}
