package widget

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/gormx/query"
	apptypes "github.com/kageos/kageos/sdk/agent-app/types"
)

type validatorBadFilesReq struct {
	InputFiles []string `json:"input_files" widget:"name:输入文件;type:files"`
}

type validatorBadTableReq struct {
	Rows string `json:"rows" widget:"name:明细;type:table"`
}

type validatorBadDependOnReq struct {
	City string `json:"city" widget:"name:城市;type:select;options:北京;depend_on:province"`
}

type validatorDynamicSelectReq struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select"`
}

type validatorDynamicNestedItem struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select"`
	Quantity  int `json:"quantity" widget:"name:数量;type:number"`
}

type validatorDynamicNestedReq struct {
	ProductID string                       `json:"product_id" widget:"name:外部商品;type:input"`
	Items     []validatorDynamicNestedItem `json:"items" widget:"name:明细;type:table"`
}

type validatorDynamicTableReq struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select"`
}

type validatorCreatableOnlySelectReq struct {
	Status string `json:"status" widget:"name:状态;type:select;creatable:true"`
}

type validatorCreatableOnlyMultiSelectReq struct {
	Tags []string `json:"tags" widget:"name:标签;type:multiselect;creatable:true"`
}

type validatorOptionsSelectReq struct {
	Status string `json:"status" widget:"name:状态;type:select;options:启用,禁用"`
}

type validatorAggregateReq struct {
	InputFiles []string `json:"input_files" widget:"name:输入文件;type:files;max_count:-1"`
	Age        int      `json:"age" widget:"name:年龄;type:number"`
}

type validatorAggregateResp struct {
	Status int `json:"status" widget:"name:状态;type:select"`
}

type validatorListReq struct {
	Numbers []int    `json:"numbers" widget:"name:数字列表;type:list;item_type:number;max_count:5"`
	Names   []string `json:"names" widget:"name:文本列表;type:list;item_type:text"`
}

type validatorBadListRenderDefaultReq struct {
	Numbers []int    `json:"numbers" widget:"name:数字列表;type:list;item_type:number;render_default:1,a,3"`
	Names   []string `json:"names" widget:"name:文本列表;type:list;item_type:text;max_count:2;render_default:a,b,c"`
}

type validatorBadListReq struct {
	Numbers []int    `json:"numbers" widget:"name:数字列表;type:list;item_type:text;max_count:-1"`
	Names   []string `json:"names" widget:"name:文本列表;type:list"`
}

type validatorBadNumericReq struct {
	Amount float64 `json:"amount" widget:"name:金额;type:number"`
	Count  int     `json:"count" widget:"name:数量;type:float"`
	Score  int     `json:"score" widget:"name:评分;type:number;min:10;max:1"`
}

type validatorBadUnknownWidgetTagReq struct {
	Title string `json:"title" widget:"name:标题;type:input;placehoder:请输入标题"`
	Count int    `json:"count" widget:"name:数量;type:number;maxcount:10"`
}

type validatorTextAreaRowsReq struct {
	Content string `json:"content" widget:"name:内容;type:text_area;rows:8"`
}

type validatorBadTextAreaRowsReq struct {
	Content string `json:"content" widget:"name:内容;type:text_area;rows:0"`
}

type validatorBadNumericDefaultRangeReq struct {
	Count  int     `json:"count" widget:"name:数量;type:number;min:1;max:5;render_default:6"`
	Amount float64 `json:"amount" widget:"name:金额;type:float;min:0;max:10;render_default:-1"`
	Level  int     `json:"level" widget:"name:等级;type:slider;render_default:120"`
	Score  float64 `json:"score" widget:"name:评分;type:rate;render_default:6"`
}

type validatorBadChoiceDetailReq struct {
	DuplicateStatus string   `json:"duplicate_status" widget:"name:重复状态;type:select;options:启用,启用"`
	ColorStatus     string   `json:"color_status" widget:"name:颜色状态;type:select;options:启用,禁用;options_colors:F56C6C"`
	BadColorStatus  string   `json:"bad_color_status" widget:"name:坏颜色;type:select;options:启用,禁用;options_colors:F56C6C,GGGGGG"`
	TagDefaults     []string `json:"tag_defaults" widget:"name:标签默认值;type:multiselect;options:A,B;render_default:A,C"`
	RadioDefault    string   `json:"radio_default" widget:"name:单选默认值;type:radio;options:A,B;render_default:C"`
	CheckboxDefault []string `json:"checkbox_default" widget:"name:复选默认值;type:checkbox;options:A,B;render_default:A,C"`
}

type validatorBadStaticMultiSelectMaxCountReq struct {
	Tags []string `json:"tags" widget:"name:标签;type:multiselect;options:A,B;max_count:3"`
}

type validatorBadColorRenderDefaultReq struct {
	Theme string `json:"theme" widget:"name:主题色;type:color;render_default:redred"`
}

type validatorBadTagSyntaxReq struct {
	Secret    string  `json:"secret" widget:"name:密文;type:input;password:yes"`
	Enabled   bool    `json:"enabled" widget:"name:启用;type:switch;render_default:maybe"`
	Level     int     `json:"level" widget:"name:等级;type:slider;min:x;step:0"`
	Progress  float64 `json:"progress" widget:"name:进度;type:progress;min:10;max:1"`
	Rating    float64 `json:"rating" widget:"name:评分;type:rate;max:0;allow_half:maybe;render_default:6"`
	Theme     string  `json:"theme" widget:"name:主题色;type:color;format:hsl;show_alpha:yes"`
	Homepage  string  `json:"homepage" widget:"name:官网;type:link;target:popup;link_type:normal"`
	Files     string  `json:"files" widget:"name:附件;type:files;accept:pdf;max_size:10Q"`
	RichIntro string  `json:"rich_intro" widget:"name:富文本;type:richtext;height:0"`
}

type validatorMaxCountReq struct {
	Tags        []string `json:"tags" widget:"name:标签;type:multiselect;options:A,B,C;max_count:2"`
	Reviewers   string   `json:"reviewers" widget:"name:审核人;type:users;max_count:3"`
	Departments string   `json:"departments" widget:"name:部门;type:departments;max_count:4"`
}

type validatorBadOneOfReq struct {
	Status string   `json:"status" widget:"name:状态;type:select;options:启用,禁用" validate:"required,oneof=启用 停用"`
	Mode   string   `json:"mode" widget:"name:模式;type:radio;options:自动,手动" validate:"oneof=自动 半自动"`
	Tags   []string `json:"tags" widget:"name:标签;type:multiselect;options:A,B" validate:"dive,oneof=A C"`
	Flags  []string `json:"flags" widget:"name:标记;type:checkbox;options:X,Y" validate:"dive,oneof=X Z"`
}

type validatorGoodOneOfReq struct {
	Status string   `json:"status" widget:"name:状态;type:select;options:启用,禁用" validate:"required,oneof=启用 禁用"`
	Tags   []string `json:"tags" widget:"name:标签;type:multiselect;options:A,B" validate:"dive,oneof=A B"`
	Size   string   `json:"size" widget:"name:尺码;type:radio;options:small size,medium size" validate:"oneof='small size' 'medium size'"`
}

type validatorBadCallbackInputReq struct {
	Name string `json:"name" widget:"name:名称;type:input" callback:"OnSelectFuzzy"`
}

type validatorBadCallbackUnknownReq struct {
	Status string `json:"status" widget:"name:状态;type:select;options:启用,禁用" callback:"OnUnknown"`
}

type validatorCallbackTagWithoutMapReq struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select;options:A,B" callback:"OnSelectFuzzy"`
}

type validatorDynamicSelectWithCallbackTagReq struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select" callback:"OnSelectFuzzy"`
}

type validatorCallbackNormalizeReq struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select" callback:"OnSelectFuzzy, OnSelectFuzzy"`
}

type validatorBadCallbackMapSelectReq struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select"`
}

type validatorAuditGoodReq struct {
	ID        int           `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt apptypes.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt apptypes.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CreatedBy string        `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

type validatorBadAuditReq struct {
	ID        int           `json:"id" gorm:"primaryKey;column:id" widget:"name:ID;type:number" hide:"list,update"`
	CreatedAt apptypes.Time `json:"created_at" gorm:"column:created_on;type:datetime" widget:"name:创建时间;type:input;format:YYYY-MM-DD" hide:"list,update"`
	CreatedBy string        `json:"created_by" gorm:"column:creator" widget:"name:创建人;type:input"`
	DeletedAt string        `json:"deleted_at" gorm:"column:deleted_at" widget:"name:删除时间;type:datetime"`
}

type validatorFieldTagItem struct {
	Name string `json:"name" widget:"name:名称;type:input"`
}

type validatorBadFieldTagReq struct {
	NameA         string                  `json:"name" widget:"name:名称A;type:input"`
	NameB         string                  `json:"name" widget:"name:名称B;type:input"`
	MissingType   string                  `json:"missing_type" widget:"name:缺少类型"`
	BadHide       string                  `json:"bad_hide" widget:"name:展示;type:input" hide:"detail"`
	EmptyHide     string                  `json:"empty_hide" widget:"name:展示;type:input" hide:""`
	BadSensitive  string                  `json:"bad_sensitive" widget:"name:敏感;type:input" sensitive:"yes"`
	DuplicateHide string                  `json:"duplicate_hide" widget:"name:展示;type:input" hide:"create,create"`
	BadData       string                  `json:"bad_data" widget:"name:数据;type:input" data:"fmt:json"`
	Items         []validatorFieldTagItem `json:"items" widget:"name:明细;type:table" hide:"create,update"`
}

type validatorSensitiveReq struct {
	Token string `json:"token" widget:"name:令牌;type:input" sensitive:"true"`
	Name  string `json:"name" widget:"name:名称;type:input" sensitive:"false"`
}

type validatorUnsupportedPasswordInputReq struct {
	Password string `json:"password" widget:"name:密码;type:input;password:true"`
}

type validatorBadDatetimeDefaultReq struct {
	Deadline apptypes.Time `json:"deadline" widget:"name:截止时间;type:datetime;render_default:NOW()"`
	PlanAt   apptypes.Time `json:"plan_at" widget:"name:计划时间;type:datetime;render_default:DATE_ADD(CURRENT_TIMESTAMP, 1 DAY)"`
}

type validatorGoodDatetimeDefaultReq struct {
	Deadline apptypes.Time `json:"deadline" widget:"name:截止时间;type:datetime;render_default:DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 DAY)"`
	PlanAt   apptypes.Time `json:"plan_at" widget:"name:计划时间;type:datetime;render_default:2026-05-01 10:30:00"`
}

type validatorTableConflictReq struct {
	Status string `json:"status" widget:"name:状态入参;type:input"`
}

type validatorPageSortReqTableConflictReq struct {
	Status string `json:"status" widget:"name:状态入参;type:input"`
	query.PageSortReq
}

type validatorTableConflictModel struct {
	Status string `json:"status" widget:"name:状态;type:select;options:A,B"`
	Title  string `json:"title" widget:"name:标题;type:input"`
}

func TestWidgetValidatorRejectsInvalidComponentParams(t *testing.T) {
	tests := []struct {
		name      string
		model     interface{}
		callbacks map[string][]string
		want      string
	}{
		{
			name:  "files requires string",
			model: &validatorBadFilesReq{},
			want:  "files widget uses comma-separated file refs and requires string Go type",
		},
		{
			name:  "table requires slice",
			model: &validatorBadTableReq{},
			want:  "table widget requires slice/array Go type",
		},
		{
			name:  "depend_on target must exist",
			model: &validatorBadDependOnReq{},
			want:  `depend_on references unknown sibling field "province"`,
		},
		{
			name:  "select needs options or callback",
			model: &validatorDynamicSelectReq{},
			want:  `widget "select" requires options or OnSelectFuzzyMap entry`,
		},
		{
			name:  "select creatable is not a choice source",
			model: &validatorCreatableOnlySelectReq{},
			want:  `widget "select" requires options or OnSelectFuzzyMap entry`,
		},
		{
			name:  "multiselect creatable is not a choice source",
			model: &validatorCreatableOnlyMultiSelectReq{},
			want:  `widget "multiselect" requires options or OnSelectFuzzyMap entry`,
		},
		{
			name:  "number and float widgets must match numeric kind",
			model: &validatorBadNumericReq{},
			want:  "number widget requires integer Go type",
		},
		{
			name:      "unknown callback target",
			model:     &validatorOptionsSelectReq{},
			callbacks: map[string][]string{"missing": []string{"OnSelectFuzzy"}},
			want:      `OnSelectFuzzyMap references unknown field "missing"`,
		},
		{
			name:      "callback target must be dynamic choice widget",
			model:     &validatorBadCallbackInputReq{},
			callbacks: map[string][]string{"name": []string{"OnSelectFuzzy"}},
			want:      `OnSelectFuzzyMap field "name" must use select or multiselect widget`,
		},
		{
			name:      "callback map value must be OnSelectFuzzy",
			model:     &validatorBadCallbackMapSelectReq{},
			callbacks: map[string][]string{"product_id": []string{"OnOther"}},
			want:      `OnSelectFuzzyMap field "product_id" contains unsupported callback "OnOther"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecodeForm(tt.callbacks, tt.model, nil)
			if err == nil {
				t.Fatal("DecodeForm() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeForm() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestWidgetValidatorRejectsInvalidChoiceDetails(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadChoiceDetailReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`widget options contains duplicate value "启用"`,
		`widget tag "options_colors" length must match options length`,
		`widget tag "options_colors" contains invalid color "GGGGGG"`,
		`widget render_default value "C" must be one of options`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorRejectsStaticMultiSelectMaxCountDrift(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadStaticMultiSelectMaxCountReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	want := `multiselect widget max_count must be <= options length for static options`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
	}
}

func TestWidgetValidatorRejectsBadColorRenderDefault(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadColorRenderDefaultReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	want := `widget render_default must be a valid color value`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
	}
}

func TestWidgetValidatorEnforcesAuditFieldConventions(t *testing.T) {
	_, _, err := DecodeTable(nil, nil, &validatorAuditGoodReq{})
	if err != nil {
		t.Fatalf("DecodeTable() error = %v, want nil", err)
	}

	_, _, err = DecodeTable(nil, nil, &validatorBadAuditReq{})
	if err == nil {
		t.Fatal("DecodeTable() error = nil, want error")
	}
	for _, want := range []string{
		`audit field "id" must use widget type "ID"`,
		`audit field "id" hide tag must be "create,update"`,
		`audit field "id" gorm tag must include autoIncrement`,
		`audit field "created_at" must use widget type "datetime"`,
		`audit field "created_at" datetime format must be "YYYY-MM-DD HH:mm:ss"`,
		`audit field "created_at" gorm column must be "created_at"`,
		`audit field "created_at" gorm tag must include autoCreateTime`,
		`audit field "created_by" must use widget type "user"`,
		`audit field "created_by" hide tag must be "create,update"`,
		`audit field "created_by" gorm column must be "created_by"`,
		`audit field "deleted_at" must be hidden with widget:"-" or json:"-"`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeTable() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorRejectsInvalidFieldLevelTags(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadFieldTagReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`duplicate field code "name" in same level`,
		`widget tag must include type`,
		`hide scene must be one of list,create,update`,
		`hide tag must not be empty`,
		`sensitive tag must be true or false`,
		`hide scene "create" is duplicated`,
		`unsupported data tag "fmt"`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestDecodeFormMarksSensitiveFields(t *testing.T) {
	fields, _, err := DecodeForm(nil, &validatorSensitiveReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(fields))
	}
	if !fields[0].Sensitive {
		t.Fatalf("sensitive tag should mark field sensitive")
	}
	if fields[1].Sensitive {
		t.Fatalf("sensitive:false should not mark field sensitive")
	}
}

func TestDecodeFormRejectsPasswordInput(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorUnsupportedPasswordInputReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "input widget does not support password") {
		t.Fatalf("DecodeForm() error = %v, want password unsupported error", err)
	}
}

func TestWidgetValidatorRejectsInvalidDatetimeDefaults(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadDatetimeDefaultReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	want := `datetime widget render_default must be a datetime literal or one of CURRENT_TIMESTAMP, CURRENT_DATE, DATE_ADD/DATE_SUB with INTERVAL`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
	}

	if _, _, err := DecodeForm(nil, &validatorGoodDatetimeDefaultReq{}, nil); err != nil {
		t.Fatalf("DecodeForm() good datetime defaults error = %v, want nil", err)
	}
}

func TestDecodeTableRejectsRequestAndTableFieldCodeConflicts(t *testing.T) {
	_, _, err := DecodeTable(nil, &validatorTableConflictReq{}, &validatorTableConflictModel{})
	if err == nil {
		t.Fatal("DecodeTable() without PageSortReq error = nil, want error")
	}
}

func TestDecodeTableAllowsPageSortReqRequestAndTableFieldCodeOverlap(t *testing.T) {
	_, _, err := DecodeTable(nil, &validatorPageSortReqTableConflictReq{}, &validatorTableConflictModel{})
	if err != nil {
		t.Fatalf("DecodeTable() error = %v, want nil", err)
	}
}

func TestWidgetValidatorRejectsUnsupportedWidgetTags(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadUnknownWidgetTagReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`unsupported widget tag "placehoder" for widget "input"`,
		`unsupported widget tag "maxcount" for widget "number"`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorAllowsTextAreaRows(t *testing.T) {
	fields, _, err := DecodeForm(nil, &validatorTextAreaRowsReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 1 {
		t.Fatalf("len(fields) = %d, want 1", len(fields))
	}
	cfg, ok := fields[0].Widget.Config.(*TextArea)
	if !ok {
		t.Fatalf("text_area config = %T, want *TextArea", fields[0].Widget.Config)
	}
	if cfg.Rows != 8 {
		t.Fatalf("Rows = %d, want 8", cfg.Rows)
	}
}

func TestWidgetValidatorRejectsInvalidTextAreaRows(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadTextAreaRowsReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `widget tag "rows" must be > 0`) {
		t.Fatalf("DecodeForm() error = %v, want rows validation error", err)
	}
}

func TestWidgetValidatorRejectsNumericDefaultRangeDrift(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadNumericDefaultRangeReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`field Count (count): widget render_default must be <= max`,
		`field Amount (amount): widget render_default must be >= min`,
		`field Level (level): widget render_default must be <= max`,
		`field Score (score): widget render_default must be <= max`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorRejectsOptionsAndOneOfDrift(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadOneOfReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`field Status (status): widget options must match validate oneof`,
		`field Mode (mode): widget options must match validate oneof`,
		`field Tags (tags): widget options must match validate oneof`,
		`field Flags (flags): widget options must match validate oneof`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorAllowsMatchingOptionsAndOneOf(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorGoodOneOfReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
}

func TestWidgetValidatorRejectsInvalidCallbackTag(t *testing.T) {
	for _, tt := range []struct {
		name  string
		model interface{}
		want  string
	}{
		{
			name:  "OnSelectFuzzy requires select or multiselect",
			model: &validatorBadCallbackInputReq{},
			want:  `callback "OnSelectFuzzy" requires select or multiselect widget`,
		},
		{
			name:  "unsupported callback",
			model: &validatorBadCallbackUnknownReq{},
			want:  `unsupported field callback "OnUnknown"`,
		},
		{
			name:  "OnSelectFuzzy tag requires map entry",
			model: &validatorCallbackTagWithoutMapReq{},
			want:  `callback "OnSelectFuzzy" requires matching OnSelectFuzzyMap entry`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecodeForm(nil, tt.model, nil)
			if err == nil {
				t.Fatal("DecodeForm() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeForm() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestWidgetValidatorRejectsInvalidTagSyntax(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadTagSyntaxReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`input widget does not support password`,
		`widget tag "render_default" must be true or false`,
		`widget tag "min" must be a number`,
		`widget tag "step" must be > 0`,
		`widget min must be <= max`,
		`widget tag "max" must be > 0`,
		`widget tag "allow_half" must be true or false`,
		`widget tag "format" must be one of hex,rgb,rgba`,
		`widget tag "target" must be one of _self,_blank`,
		`widget tag "link_type" must be one of primary,success,warning,danger,info`,
		`widget tag "accept" contains invalid accept item "pdf"`,
		`widget tag "max_size" must use unit B, KB, MB, or GB`,
		`widget tag "height" must be > 0`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorRejectsNumericKindAndRangeDrift(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadNumericReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		"number widget requires integer Go type",
		"float widget requires float32/float64 Go type",
		"widget min must be <= max",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetParserPreservesMaxCount(t *testing.T) {
	fields, _, err := DecodeForm(nil, &validatorMaxCountReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	configs := map[string]int{}
	for _, field := range fields {
		switch cfg := field.Widget.Config.(type) {
		case *MultiSelect:
			configs[field.Code] = cfg.MaxCount
		case *Users:
			configs[field.Code] = cfg.MaxCount
		case *Departments:
			configs[field.Code] = cfg.MaxCount
		}
	}
	for code, want := range map[string]int{
		"tags":        2,
		"reviewers":   3,
		"departments": 4,
	} {
		if configs[code] != want {
			t.Fatalf("%s MaxCount = %d, want %d", code, configs[code], want)
		}
	}
}

func TestWidgetValidatorAllowsListTypes(t *testing.T) {
	fields, _, err := DecodeForm(nil, &validatorListReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 2 {
		t.Fatalf("DecodeForm() fields length = %d, want 2", len(fields))
	}
	if fields[0].Widget.Type != TypeList || fields[0].Widget.Config.(*List).ItemType != ListItemTypeNumber {
		t.Fatalf("unexpected number list field: %+v", fields[0])
	}
}

func TestWidgetValidatorRejectsInvalidListTypes(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadListReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		"item_type:text requires string slice/array element type",
		`widget tag "max_count" must be >= 0`,
		"list widget requires item_type:number or item_type:text",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorRejectsInvalidListRenderDefaults(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorBadListRenderDefaultReq{}, nil)
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		`list widget render_default value "a" must be a number`,
		`list widget render_default count must be <= max_count`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestDecodeFormAggregatesRequestAndResponseErrors(t *testing.T) {
	_, _, err := DecodeForm(nil, &validatorAggregateReq{}, &validatorAggregateResp{})
	if err == nil {
		t.Fatal("DecodeForm() error = nil, want error")
	}
	for _, want := range []string{
		"files widget uses comma-separated file refs and requires string Go type",
		`widget tag "max_count" must be >= 0`,
		`widget "select" requires options or OnSelectFuzzyMap entry`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("DecodeForm() error = %v, want substring %q", err, want)
		}
	}
}

func TestWidgetValidatorRegistryCoversSupportedTypes(t *testing.T) {
	if err := ValidateWidgetValidatorRegistry(); err != nil {
		t.Fatalf("ValidateWidgetValidatorRegistry() error = %v, want nil", err)
	}
}

func TestAllowedTagKeysExposeRuntimeContract(t *testing.T) {
	for _, widgetType := range SupportedTypes() {
		keys := AllowedTagKeys(widgetType)
		if len(keys) == 0 {
			t.Fatalf("AllowedTagKeys(%q) returned no keys", widgetType)
		}
		if !hasStringItem(keys, "type") {
			t.Fatalf("AllowedTagKeys(%q) must include type, got %#v", widgetType, keys)
		}
	}

	inputKeys := AllowedTagKeys(TypeInput)
	if !hasStringItem(inputKeys, "placeholder") {
		t.Fatalf("input keys should include placeholder, got %#v", inputKeys)
	}
	if hasStringItem(inputKeys, "readonly") {
		t.Fatalf("input keys should not include unsupported readonly: %#v", inputKeys)
	}
}

func TestWidgetValidatorRejectsCreatableOnlyChoiceSources(t *testing.T) {
	for _, tt := range []struct {
		name  string
		model interface{}
		want  string
	}{
		{
			name:  "select",
			model: &validatorCreatableOnlySelectReq{},
			want:  `widget "select" requires options or OnSelectFuzzyMap entry`,
		},
		{
			name:  "multiselect",
			model: &validatorCreatableOnlyMultiSelectReq{},
			want:  `widget "multiselect" requires options or OnSelectFuzzyMap entry`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecodeForm(nil, tt.model, nil)
			if err == nil {
				t.Fatal("DecodeForm() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DecodeForm() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestWidgetValidatorAllowsDynamicSelect(t *testing.T) {
	_, fields, err := DecodeForm(map[string][]string{
		"product_id": {"OnSelectFuzzy"},
	}, nil, &validatorDynamicSelectReq{})
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 1 || fields[0].Code != "product_id" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
}

func TestWidgetParserAttachesDynamicCallbacksRecursively(t *testing.T) {
	fields, _, err := DecodeForm(map[string][]string{
		"product_id": {"OnSelectFuzzy"},
	}, &validatorDynamicNestedReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 2 {
		t.Fatalf("DecodeForm() fields length = %d, want 2", len(fields))
	}
	if got := fields[0].Callbacks; len(got) != 0 {
		t.Fatalf("top-level input callbacks = %#v, want empty", got)
	}
	if len(fields[1].Children) != 2 {
		t.Fatalf("nested children length = %d, want 2", len(fields[1].Children))
	}
	if got := fields[1].Children[0].Callbacks; len(got) != 1 || got[0] != "OnSelectFuzzy" {
		t.Fatalf("nested select callbacks = %#v, want %#v", got, []string{"OnSelectFuzzy"})
	}
}

func TestWidgetParserAttachesDynamicCallbacksToTableFields(t *testing.T) {
	requestFields, _, err := DecodeTable(map[string][]string{
		"product_id": {"OnSelectFuzzy"},
	}, &validatorDynamicTableReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeTable() request error = %v, want nil", err)
	}
	if len(requestFields) != 1 {
		t.Fatalf("unexpected request fields: %+v", requestFields)
	}
	if got := requestFields[0].Callbacks; len(got) != 1 || got[0] != "OnSelectFuzzy" {
		t.Fatalf("request callbacks = %#v, want %#v", got, []string{"OnSelectFuzzy"})
	}

	_, responseFields, err := DecodeTable(map[string][]string{
		"product_id": {"OnSelectFuzzy"},
	}, nil, &validatorDynamicTableReq{})
	if err != nil {
		t.Fatalf("DecodeTable() response error = %v, want nil", err)
	}
	if len(responseFields) != 1 {
		t.Fatalf("unexpected response fields: %+v", responseFields)
	}
	if got := responseFields[0].Callbacks; len(got) != 1 || got[0] != "OnSelectFuzzy" {
		t.Fatalf("response callbacks = %#v, want %#v", got, []string{"OnSelectFuzzy"})
	}
}

func TestWidgetValidatorAllowsDynamicSelectWithCallbackTag(t *testing.T) {
	_, fields, err := DecodeForm(map[string][]string{
		"product_id": {"OnSelectFuzzy"},
	}, nil, &validatorDynamicSelectWithCallbackTagReq{})
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 1 || fields[0].Code != "product_id" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
}

func TestWidgetParserNormalizesCallbacks(t *testing.T) {
	fields, _, err := DecodeForm(map[string][]string{
		"product_id": {" OnSelectFuzzy ", "OnSelectFuzzy"},
	}, &validatorCallbackNormalizeReq{}, nil)
	if err != nil {
		t.Fatalf("DecodeForm() error = %v, want nil", err)
	}
	if len(fields) != 1 {
		t.Fatalf("DecodeForm() fields length = %d, want 1", len(fields))
	}
	if got := fields[0].Callbacks; len(got) != 1 || got[0] != "OnSelectFuzzy" {
		t.Fatalf("callbacks = %#v, want %#v", got, []string{"OnSelectFuzzy"})
	}
}
