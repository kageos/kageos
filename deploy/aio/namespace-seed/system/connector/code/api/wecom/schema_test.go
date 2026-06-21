package wecom

import (
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func TestWeComTemplatesDecodeSchema(t *testing.T) {
	formTemplates := map[string]*app.FormTemplate{
		"WeComConfigSaveTemplate":       WeComConfigSaveTemplate,
		"WeComConnectionStatusTemplate": WeComConnectionStatusTemplate,
		"WeComSendTextTemplate":         WeComSendTextTemplate,
		"WeComSendMarkdownTemplate":     WeComSendMarkdownTemplate,
	}
	for name, template := range formTemplates {
		if _, _, err := widget.DecodeForm(wecomTemplateCallbacks(template.OnSelectFuzzyMap), template.Request, template.Response); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}

	tableTemplates := map[string]*app.TableTemplate{
		"WeComConfigListTemplate": WeComConfigListTemplate,
	}
	for name, template := range tableTemplates {
		if _, _, err := widget.DecodeTable(wecomTemplateCallbacks(template.OnSelectFuzzyMap), template.Request, template.AutoCrudTable); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}
}

func TestWeComSecretEncryptDecrypt(t *testing.T) {
	plain := "wecom-secret-for-test"
	cipherText, err := encryptWeComSecret(plain)
	if err != nil {
		t.Fatalf("encryptWeComSecret error: %v", err)
	}
	if cipherText == plain {
		t.Fatalf("secret should not be stored as plaintext")
	}
	got, err := decryptWeComSecret(cipherText)
	if err != nil {
		t.Fatalf("decryptWeComSecret error: %v", err)
	}
	if got != plain {
		t.Fatalf("decrypted secret = %q, want %q", got, plain)
	}
}

func wecomTemplateCallbacks(fuzzy map[string]app.OnSelectFuzzy) map[string][]string {
	if len(fuzzy) == 0 {
		return nil
	}
	callbacks := make(map[string][]string, len(fuzzy))
	for field := range fuzzy {
		callbacks[field] = []string{app.CallbackTypeOnSelectFuzzy}
	}
	return callbacks
}
