package service

import (
	"strings"
	"testing"
)

func TestWorkspaceBuildErrorHints(t *testing.T) {
	errText := `app startup failed: SDK schema compile failed:
router /auction/auction_item_list.table table schema decode failed: field Status widget tag "options_colors" contains invalid color "default"
failed to validate table request fields: table request field "session_id" conflicts with table model field "session_id"
failed to validate table callbacks: OnSelectFuzzyMap field "item_id" must use select or multiselect widget
field CurrentPrice (current_price): unsupported widget tag "readonly" for widget "float"
field Attachments (attachments): unsupported widget type "file"
field AvgScore (avg_score): number widget requires integer Go type, got float64
namespace/auction/evaluation_object_list.go:62:9: req.Status undefined (type EvaluationObjectListReq has no field or method Status)
namespace/auction/auction_statistics.go:94:34: d.After undefined (type func() time.Time has no field or method After)
namespace/auction/auction_item_list.go:115:14: req.GetPage undefined
namespace/auction/auction_statistics.go:49: undefined: chart.ComboChart
namespace/auction/auction_statistics.go:204:20: undefined: types.EmptyRequest
namespace/auction/publish_item.go:81:21: undefined: app.Time
namespace/auction/nps_statistics.go:81:20: cannot use res (variable of type *NpsStatisticsResp) as chart.Charter value in argument to resp.Chart: *NpsStatisticsResp does not implement chart.Charter (missing method GetChartType)
namespace/auction/auction_bid_list.go:15:6: AuctionBidRecord redeclared in this block`

	enriched := enrichWorkspaceBuildError(errText)
	for _, want := range []string{
		"options_colors 只支持不带 # 的 6 位十六进制 RRGGBB",
		"widget 的 type 和配置 key 必须来自 SDK 主文档组件速查和运行时白名单",
		"type:number 只配 int/int64 等整数",
		"Table 列表使用 query.PageSortReq",
		"调整 Table Request 字段后",
		"OnSelectFuzzyMap 的 key 必须对应 schema",
		"分页默认使用 resp.Table(&rows, queryDB",
		"types.Time 做格式化或比较时先调用 Time() 方法",
		"未确认的 SDK API chart.ComboChart",
		"未确认的 SDK API types.EmptyRequest",
		"未确认的 SDK API app.Time",
		"Chart Handler 必须把 SDK chart 包里的具体图表对象传给 resp.Chart",
		"同一个 Go package 里的模型、函数和方法只能定义一次",
	} {
		if !strings.Contains(enriched, want) {
			t.Fatalf("expected build error hint %q in:\n%s", want, enriched)
		}
	}
	if got := strings.Count(enriched, "options_colors 只支持"); got != 1 {
		t.Fatalf("expected options_colors hint once, got %d in:\n%s", got, enriched)
	}
}

func TestWorkspaceBuildErrorHintsNoop(t *testing.T) {
	const errText = "unexpected build failure"
	if got := enrichWorkspaceBuildError(errText); got != errText {
		t.Fatalf("unexpected hint for unknown error: %q", got)
	}
}

func TestWorkspaceBuildErrorMentionsScope(t *testing.T) {
	const errText = "app startup failed: SDK schema compile failed: router /evaluation/evaluation_record_list.table table schema decode failed"
	got := enrichWorkspaceBuildError(errText, "/liubeiluo/test4")
	for _, want := range []string{
		"本次编译范围: /liubeiluo/test4",
		"router /xxx 是该工作空间内相对路由",
		errText,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in:\n%s", want, got)
		}
	}
}
