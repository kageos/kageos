package service

import (
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/functionschema"
)

func TestCollectFunctionSensitiveFields(t *testing.T) {
	schema := functionschema.NewForm(
		[]*widget.Field{
			{
				Code:      "member",
				Name:      "会员",
				Sensitive: true,
			},
			{
				Code: "items",
				Name: "明细",
				Children: []*widget.Field{
					{Code: "serial", Name: "序列号", Sensitive: true},
				},
			},
		},
		[]*widget.Field{
			{
				Code: "payment",
				Name: "支付",
				Children: []*widget.Field{
					{Code: "card_number", Name: "卡号", Sensitive: true},
				},
			},
		},
		nil,
	)

	fields := collectFunctionSensitiveFields("alice", "cashier", "/alice/cashier/pay.form", 98, schema)
	got := make(map[string]string)
	for _, field := range fields {
		got[field.Section+":"+field.FieldPath] = field.Source
	}

	for _, key := range []string{
		"request:member",
		"request:items.serial",
		"response:payment.card_number",
	} {
		if got[key] != "schema" {
			t.Fatalf("missing sensitive field %s in %+v", key, got)
		}
	}
}
