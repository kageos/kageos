package widget

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ValidateContext struct {
	Field       *FieldTags
	FieldCode   string
	FieldName   string
	GoType      reflect.Type
	WidgetType  string
	SiblingCode map[string]struct{}
	Callbacks   map[string][]string
}

// WidgetValidator 是每个 widget 组件的启动期契约校验函数。
//
// 校验发生在 DecodeForm/DecodeTable 解析 Go struct tag 之后、schema 输出之前：
// 1. 先做字段级通用校验，例如 widget type 是否支持、depend_on 是否指向同级字段、字段筛选配置是否匹配 Go 类型；
// 2. 再调用具体组件注册的 validator，校验该组件自己的 Go 类型、必填配置、数值范围等；
// 3. 所有错误会通过 errors.Join 聚合返回，避免开发者一次只修一个字段。
//
// 组件文件里通常通过 init 注册：
//
//	func init() {
//	    RegisterWidgetValidator(TypeSelect, validateSelectWidget)
//	}
//
// 如果新增组件，必须同时：
// - 在 widget.go 的 supportedWidgetTypes/NewWidget 中登记；
// - 在组件源码里注册 WidgetValidator；
// - 在 validator_test.go 覆盖错误场景，否则 ValidateWidgetValidatorRegistry 会失败。
type WidgetValidator func(ctx ValidateContext) error

var widgetValidators = map[string]WidgetValidator{}

func RegisterWidgetValidator(widgetType string, validator WidgetValidator) {
	if !IsSupportedType(widgetType) {
		panic(fmt.Sprintf("cannot register validator for unsupported widget type %q", widgetType))
	}
	if validator == nil {
		panic(fmt.Sprintf("cannot register nil validator for widget type %q", widgetType))
	}
	if _, exists := widgetValidators[widgetType]; exists {
		panic(fmt.Sprintf("duplicate validator registration for widget type %q", widgetType))
	}
	widgetValidators[widgetType] = validator
}

func ValidateWidgetValidatorRegistry() error {
	var errs []error
	for _, widgetType := range SupportedTypes() {
		if widgetValidators[widgetType] == nil {
			errs = append(errs, fmt.Errorf("missing widget validator for supported widget type %q", widgetType))
		}
	}
	return errors.Join(errs...)
}

func ValidateFieldTags(tags []*FieldTags, fieldsCallback map[string][]string) error {
	if len(tags) == 0 {
		return nil
	}
	if err := ValidateWidgetValidatorRegistry(); err != nil {
		return err
	}
	return validateFieldTagLevel(tags, fieldsCallback)
}

func ValidateFieldCallbackTargets(tags []*FieldTags, fieldsCallback map[string][]string) error {
	if len(fieldsCallback) == 0 {
		return nil
	}
	byCode := collectFieldTagsByCode(tags)
	var errs []error
	for code, callbacks := range fieldsCallback {
		if err := validateCallbackMapCallbacks(code, callbacks); err != nil {
			errs = append(errs, err)
		}
		targets := byCode[code]
		if len(targets) == 0 {
			errs = append(errs, fmt.Errorf("OnSelectFuzzyMap references unknown field %q", code))
			continue
		}
		if !hasDynamicChoiceTarget(targets) {
			errs = append(errs, fmt.Errorf("OnSelectFuzzyMap field %q must use select or multiselect widget", code))
		}
	}
	return errors.Join(errs...)
}

func validateFieldTagLevel(tags []*FieldTags, fieldsCallback map[string][]string) error {
	siblingCodes := make(map[string]struct{}, len(tags))
	codeFields := make(map[string]string, len(tags))
	var errs []error
	for _, tags := range tags {
		if tags == nil {
			continue
		}
		if code := tags.GetCode(); code != "" {
			if firstFieldName, exists := codeFields[code]; exists {
				errs = append(errs, fmt.Errorf("duplicate field code %q in same level: %s and %s", code, firstFieldName, tags.FieldName))
			}
			codeFields[code] = tags.FieldName
			siblingCodes[code] = struct{}{}
		}
	}

	for _, tags := range tags {
		if tags == nil {
			continue
		}
		ctx := ValidateContext{
			Field:       tags,
			FieldCode:   tags.GetCode(),
			FieldName:   tags.FieldName,
			GoType:      tags.Type,
			WidgetType:  strings.TrimSpace(tags.WidgetParsed["type"]),
			SiblingCode: siblingCodes,
			Callbacks:   fieldsCallback,
		}
		if err := validateFieldTag(ctx); err != nil {
			errs = append(errs, err)
		}
		if len(tags.Children) > 0 {
			if err := validateFieldTagLevel(tags.Children, fieldsCallback); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func validateFieldTag(ctx ValidateContext) error {
	var errs []error
	if err := validateAuditFieldConvention(ctx); err != nil {
		errs = append(errs, err)
	}
	if ctx.WidgetType == "" {
		if strings.TrimSpace(ctx.Field.Widget) != "" {
			errs = append(errs, fieldError(ctx, "widget tag must include type"))
		}
		return errors.Join(errs...)
	}
	if !IsSupportedType(ctx.WidgetType) {
		errs = append(errs, fieldError(ctx, "unsupported widget type %q", ctx.WidgetType))
		return errors.Join(errs...)
	}
	if err := validateWidgetTagKeys(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateDataTagKeys(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateHideTag(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateDependOn(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateFieldCallbackTag(ctx); err != nil {
		errs = append(errs, err)
	}
	validator := widgetValidators[ctx.WidgetType]
	if validator == nil {
		errs = append(errs, fieldError(ctx, "missing validator for supported widget type %q", ctx.WidgetType))
	} else if err := validator(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateAuditFieldConvention(ctx ValidateContext) error {
	switch ctx.FieldCode {
	case "id":
		if !isGormPrimaryID(ctx.Field.Gorm) {
			return nil
		}
		return validateAuditIDField(ctx)
	case "created_at":
		return validateAuditTimeField(ctx, "created_at", "autoCreateTime")
	case "updated_at":
		return validateAuditTimeField(ctx, "updated_at", "autoUpdateTime")
	case "create_by", "created_by":
		return validateAuditUserField(ctx, ctx.FieldCode)
	case "update_by", "updated_by":
		return validateAuditUserField(ctx, ctx.FieldCode)
	case "deleted_at":
		return fieldError(ctx, `audit field "deleted_at" must be hidden with widget:"-" or json:"-"`)
	default:
		return nil
	}
}

func validateAuditIDField(ctx ValidateContext) error {
	var errs []error
	if ctx.WidgetType != TypeID {
		errs = append(errs, fieldError(ctx, `audit field "id" must use widget type %q`, TypeID))
	}
	if err := requireHiddenInCreateUpdate(ctx, `audit field "id"`); err != nil {
		errs = append(errs, err)
	}
	if !hasGormFlag(ctx.Field.Gorm, "autoIncrement") {
		errs = append(errs, fieldError(ctx, `audit field "id" gorm tag must include autoIncrement`))
	}
	if column := gormTagValue(ctx.Field.Gorm, "column"); column != "id" {
		errs = append(errs, fieldError(ctx, `audit field "id" gorm column must be "id", got %q`, column))
	}
	return errors.Join(errs...)
}

func validateAuditTimeField(ctx ValidateContext, code string, autoFlag string) error {
	var errs []error
	if ctx.WidgetType != TypeDatetime {
		errs = append(errs, fieldError(ctx, `audit field %q must use widget type %q`, code, TypeDatetime))
	}
	if format := strings.TrimSpace(ctx.Field.WidgetParsed["format"]); format != "YYYY-MM-DD HH:mm:ss" {
		errs = append(errs, fieldError(ctx, `audit field %q datetime format must be "YYYY-MM-DD HH:mm:ss", got %q`, code, format))
	}
	if err := requireHiddenInCreateUpdate(ctx, fmt.Sprintf("audit field %q", code)); err != nil {
		errs = append(errs, err)
	}
	if column := gormTagValue(ctx.Field.Gorm, "column"); column != code {
		errs = append(errs, fieldError(ctx, `audit field %q gorm column must be %q, got %q`, code, code, column))
	}
	if !hasGormFlag(ctx.Field.Gorm, autoFlag) {
		errs = append(errs, fieldError(ctx, `audit field %q gorm tag must include %s`, code, autoFlag))
	}
	return errors.Join(errs...)
}

func validateAuditUserField(ctx ValidateContext, code string) error {
	var errs []error
	if ctx.WidgetType != TypeUser {
		errs = append(errs, fieldError(ctx, `audit field %q must use widget type %q`, code, TypeUser))
	}
	if err := requireHiddenInCreateUpdate(ctx, fmt.Sprintf("audit field %q", code)); err != nil {
		errs = append(errs, err)
	}
	if column := gormTagValue(ctx.Field.Gorm, "column"); column != code {
		errs = append(errs, fieldError(ctx, `audit field %q gorm column must be %q, got %q`, code, code, column))
	}
	return errors.Join(errs...)
}

func requireHiddenInCreateUpdate(ctx ValidateContext, label string) error {
	scenes := parseSceneList(ctx.Field.Hide)
	if len(scenes) == 2 && hasStringItem(scenes, "create") && hasStringItem(scenes, "update") {
		return nil
	}
	return fieldError(ctx, `%s hide tag must be "create,update", got %q`, label, ctx.Field.Hide)
}

func validateDataTagKeys(ctx ValidateContext) error {
	if strings.TrimSpace(ctx.Field.Data) == "" {
		return nil
	}
	allowed := map[string]struct{}{
		"format":  {},
		"example": {},
	}
	var errs []error
	for key := range ctx.Field.DataParsed {
		if _, ok := allowed[key]; ok {
			continue
		}
		errs = append(errs, fieldError(ctx, "unsupported data tag %q", key))
	}
	return errors.Join(errs...)
}

func validateHideTag(ctx ValidateContext) error {
	if !ctx.Field.HideSet {
		return nil
	}
	var errs []error
	scenes := parseSceneList(ctx.Field.Hide)
	if len(scenes) == 0 {
		errs = append(errs, fieldError(ctx, `hide tag must not be empty`))
		return errors.Join(errs...)
	}
	seen := make(map[string]struct{}, len(scenes))
	for _, scene := range scenes {
		switch scene {
		case "list", "create", "update":
		default:
			errs = append(errs, fieldError(ctx, `hide scene must be one of list,create,update, got %q`, scene))
			continue
		}
		if _, exists := seen[scene]; exists {
			errs = append(errs, fieldError(ctx, `hide scene %q is duplicated`, scene))
			continue
		}
		seen[scene] = struct{}{}
	}
	return errors.Join(errs...)
}

func isGormPrimaryID(raw string) bool {
	return hasGormFlag(raw, "primaryKey") || hasGormFlag(raw, "primary_key")
}

func hasGormFlag(raw string, flag string) bool {
	for _, part := range strings.Split(raw, ";") {
		key, _ := splitGormTagPart(part)
		if key == flag {
			return true
		}
	}
	return false
}

func gormTagValue(raw string, key string) string {
	for _, part := range strings.Split(raw, ";") {
		partKey, value := splitGormTagPart(part)
		if partKey == key {
			return value
		}
	}
	return ""
}

func splitGormTagPart(part string) (string, string) {
	part = strings.TrimSpace(part)
	if part == "" {
		return "", ""
	}
	key, value, ok := strings.Cut(part, ":")
	if !ok {
		return part, ""
	}
	return strings.TrimSpace(key), strings.TrimSpace(value)
}

func validateDependOn(ctx ValidateContext) error {
	dependOn := strings.TrimSpace(ctx.Field.WidgetParsed["depend_on"])
	if dependOn == "" {
		return nil
	}
	if _, ok := ctx.SiblingCode[dependOn]; !ok {
		return fieldError(ctx, "depend_on references unknown sibling field %q", dependOn)
	}
	return nil
}

func validateWidgetTagKeys(ctx ValidateContext) error {
	allowed := allowedWidgetTagKeys(ctx.WidgetType)
	if len(allowed) == 0 {
		return fieldError(ctx, "missing widget tag allowlist for widget %q", ctx.WidgetType)
	}
	var errs []error
	for key := range ctx.Field.WidgetParsed {
		if _, ok := allowed[key]; ok {
			continue
		}
		errs = append(errs, fieldError(ctx, "unsupported widget tag %q for widget %q", key, ctx.WidgetType))
	}
	return errors.Join(errs...)
}

func allowedWidgetTagKeys(widgetType string) map[string]struct{} {
	common := []string{"name", "type", "desc", "depend_on"}
	byWidget := map[string][]string{
		TypeInput:       {"placeholder", "password", "prepend", "append", renderDefaultTagKey},
		TypeText:        {"format"},
		TypeTextArea:    {"placeholder", renderDefaultTagKey},
		TypeSelect:      {"options", "options_colors", "placeholder", renderDefaultTagKey, "creatable"},
		TypeSwitch:      {renderDefaultTagKey},
		TypeDatetime:    {"format", "disabled", renderDefaultTagKey},
		TypeUser:        {renderDefaultTagKey, "disabled"},
		TypeUsers:       {renderDefaultTagKey, "max_count"},
		TypeDepartment:  {renderDefaultTagKey},
		TypeDepartments: {renderDefaultTagKey, "max_count"},
		TypeID:          {},
		TypeNumber:      {"placeholder", "min", "max", "step", renderDefaultTagKey, "unit"},
		TypeFloat:       {"placeholder", "min", "max", "precision", "step", renderDefaultTagKey, "unit"},
		TypeFiles:       {"accept", "max_size", "max_count"},
		TypeCheckbox:    {"options", renderDefaultTagKey},
		TypeRadio:       {"options", renderDefaultTagKey},
		TypeMultiSelect: {"options", "options_colors", "placeholder", renderDefaultTagKey, "max_count", "creatable"},
		TypeSlider:      {"min", "max", "step", renderDefaultTagKey, "unit"},
		TypeRate:        {"max", "allow_half", renderDefaultTagKey, "texts"},
		TypeColor:       {"format", renderDefaultTagKey, "show_alpha"},
		TypeRichText:    {"height"},
		TypeTable:       {},
		TypeForm:        {},
		TypeLink:        {"text", "target", "link_type", "icon"},
		TypeProgress:    {"min", "max", "unit"},
		TypeList:        {"item_type", "separator", "placeholder", renderDefaultTagKey, "unique", "max_count"},
	}
	specific, ok := byWidget[widgetType]
	if !ok {
		return nil
	}
	allowed := make(map[string]struct{}, len(common)+len(specific))
	for _, key := range common {
		allowed[key] = struct{}{}
	}
	for _, key := range specific {
		allowed[key] = struct{}{}
	}
	return allowed
}

// AllowedTagKeys returns the widget tag keys accepted by the SDK validator for a widget type.
// It is intended for prompt/example linting so docs stay aligned with the runtime contract.
func AllowedTagKeys(widgetType string) []string {
	allowed := allowedWidgetTagKeys(widgetType)
	keys := make([]string, 0, len(allowed))
	for key := range allowed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateFieldCallbackTag(ctx ValidateContext) error {
	callbacks := parseCallbackTag(ctx.Field.Callback)
	if len(callbacks) == 0 {
		return nil
	}
	var errs []error
	for _, callback := range callbacks {
		switch callback {
		case "OnSelectFuzzy":
			if !supportsDynamicChoiceCallback(ctx.WidgetType) {
				errs = append(errs, fieldError(ctx, "callback %q requires select or multiselect widget, got %q", callback, ctx.WidgetType))
				continue
			}
			if !hasChoiceCallback(ctx) {
				errs = append(errs, fieldError(ctx, "callback %q requires matching OnSelectFuzzyMap entry", callback))
			}
		default:
			errs = append(errs, fieldError(ctx, "unsupported field callback %q", callback))
		}
	}
	return errors.Join(errs...)
}

func validateDatetimeRenderDefault(ctx ValidateContext) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[renderDefaultTagKey])
	if raw == "" {
		return nil
	}
	if isStaticDatetimeLiteral(raw) || isValidDatetimeDynamicDefault(raw) {
		return nil
	}
	return fieldError(ctx, "datetime widget render_default must be a datetime literal or one of CURRENT_TIMESTAMP, CURRENT_DATE, DATE_ADD/DATE_SUB with INTERVAL, got %q", raw)
}

// requireStringLikeGoType 是 input/text/text_area/richtext/color/link/user/department 等组件复用的底层 helper。
//
// 这些组件的协议值都是单个字符串：
// - input/text_area/richtext/color/link 是普通文本值；
// - user/department 存储用户或部门标识；
// - datetime 不复用该函数，因为 datetime 允许 string/time.Time/sdk types.Time。
//
// 注意：这里校验的是 Go 字段类型，不校验业务 validate tag。
// 例如 validate:"required,min=2" 仍由前端/validator 协议处理，这里只保证 widget 和 Go 类型不漂移。
func requireStringLikeGoType(ctx ValidateContext) error {
	if !isStringLikeType(ctx.GoType) {
		return fieldError(ctx, "widget %q requires string Go type, got %s", ctx.WidgetType, typeName(ctx.GoType))
	}
	return nil
}

// requireStringOrStringSliceGoType 是 users/departments 多选标识类组件复用的底层 helper。
//
// 合法 Go 类型：
// - string：推荐落库格式，使用逗号分隔多个标识，前端会做字符串/数组转换；
// - []string 或 [N]string：适合纯接口结构，不直接落单列。
//
// 不允许 []int、[]struct 等类型，因为这些组件的真实值是用户名/full_code_path 这类字符串标识。
func requireStringOrStringSliceGoType(ctx ValidateContext) error {
	if isStringLikeType(ctx.GoType) {
		return nil
	}
	elem := derefType(ctx.GoType)
	if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Array {
		return validateScalarElement(ctx, elem.Elem(), "string")
	}
	return fieldError(ctx, "widget %q requires string or []string Go type, got %s", ctx.WidgetType, typeName(ctx.GoType))
}

// requireNumericGoType 是 slider/rate/progress 等数值展示或输入组件复用的底层 helper。
//
// 合法 Go 类型包括整数和浮点数。它只校验“字段是不是数值”，不解析 min/max/step；
// 如果某个组件需要更严格的 tag 校验，应在对应组件文件的 validateXxxWidget 中追加规则。
func requireNumericGoType(ctx ValidateContext) error {
	if !isNumericType(ctx.GoType) {
		return fieldError(ctx, "widget %q requires numeric Go type, got %s", ctx.WidgetType, typeName(ctx.GoType))
	}
	return nil
}

// requireChoiceSource 是 select/multiselect 复用的选项来源 helper。
//
// 这条规则很重要：下拉类组件必须有“可供选择的数据来源”，来源只能是：
// - 静态 options，例如 widget:"type:select;options:启用,禁用"；
// - 动态 OnSelectFuzzyMap 回调，例如 app 模板里把字段 code 注册到 OnSelectFuzzy。
//
// creatable:true 不算选项来源。它只表示用户可以在已有来源基础上创建新值；
// 如果没有 options 或回调，前端没有初始候选集，也无法知道如何搜索/回填显示值。
func requireChoiceSource(ctx ValidateContext) error {
	if len(parseOptions(ctx.Field.WidgetParsed["options"])) > 0 {
		return nil
	}
	if hasChoiceCallback(ctx) {
		return nil
	}
	return fieldError(ctx, "widget %q requires options or OnSelectFuzzyMap entry", ctx.WidgetType)
}

func validateNonNegativeIntTag(ctx ValidateContext, key string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fieldError(ctx, "widget tag %q must be an integer, got %q", key, raw)
	}
	if value < 0 {
		return fieldError(ctx, "widget tag %q must be >= 0, got %d", key, value)
	}
	return nil
}

func validatePositiveIntTag(ctx ValidateContext, key string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fieldError(ctx, "widget tag %q must be an integer, got %q", key, raw)
	}
	if value <= 0 {
		return fieldError(ctx, "widget tag %q must be > 0, got %d", key, value)
	}
	return nil
}

func validateIntTag(ctx ValidateContext, key string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	if _, err := strconv.Atoi(raw); err != nil {
		return fieldError(ctx, "widget tag %q must be an integer, got %q", key, raw)
	}
	return nil
}

func validateFloatTag(ctx ValidateContext, key string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	if _, err := strconv.ParseFloat(raw, 64); err != nil {
		return fieldError(ctx, "widget tag %q must be a number, got %q", key, raw)
	}
	return nil
}

func validatePositiveFloatTag(ctx ValidateContext, key string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fieldError(ctx, "widget tag %q must be a number, got %q", key, raw)
	}
	if value <= 0 {
		return fieldError(ctx, "widget tag %q must be > 0, got %s", key, raw)
	}
	return nil
}

func validateBoolTag(ctx ValidateContext, key string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	if raw != "true" && raw != "false" {
		return fieldError(ctx, "widget tag %q must be true or false, got %q", key, raw)
	}
	return nil
}

func validateEnumTag(ctx ValidateContext, key string, allowed ...string) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed[key])
	if raw == "" {
		return nil
	}
	for _, item := range allowed {
		if raw == item {
			return nil
		}
	}
	return fieldError(ctx, "widget tag %q must be one of %s, got %q", key, strings.Join(allowed, ","), raw)
}

func validateNumberRange(ctx ValidateContext) error {
	rawMin := strings.TrimSpace(ctx.Field.WidgetParsed["min"])
	rawMax := strings.TrimSpace(ctx.Field.WidgetParsed["max"])
	if rawMin == "" || rawMax == "" {
		return nil
	}
	minValue, minErr := strconv.ParseFloat(rawMin, 64)
	maxValue, maxErr := strconv.ParseFloat(rawMax, 64)
	if minErr != nil || maxErr != nil {
		return nil
	}
	if minValue > maxValue {
		return fieldError(ctx, "widget min must be <= max, got min=%s max=%s", rawMin, rawMax)
	}
	return nil
}

func validateRenderDefaultNumberRange(ctx ValidateContext, fallbackMin, fallbackMax *float64) error {
	rawDefault := strings.TrimSpace(ctx.Field.WidgetParsed[renderDefaultTagKey])
	if rawDefault == "" {
		return nil
	}
	defaultValue, err := strconv.ParseFloat(rawDefault, 64)
	if err != nil {
		return nil
	}
	minValue, minLabel, hasMin := resolveNumericBound(ctx.Field.WidgetParsed["min"], fallbackMin)
	if hasMin && defaultValue < minValue {
		return fieldError(ctx, "widget render_default must be >= min, got render_default=%s min=%s", rawDefault, minLabel)
	}
	maxValue, maxLabel, hasMax := resolveNumericBound(ctx.Field.WidgetParsed["max"], fallbackMax)
	if hasMax && defaultValue > maxValue {
		return fieldError(ctx, "widget render_default must be <= max, got render_default=%s max=%s", rawDefault, maxLabel)
	}
	return nil
}

func resolveNumericBound(raw string, fallback *float64) (float64, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return 0, "", false
		}
		return value, raw, true
	}
	if fallback == nil {
		return 0, "", false
	}
	return *fallback, strconv.FormatFloat(*fallback, 'f', -1, 64), true
}

func validateUniqueOptions(ctx ValidateContext) error {
	options := parseOptions(ctx.Field.WidgetParsed["options"])
	if len(options) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		if _, ok := seen[option]; ok {
			return fieldError(ctx, "widget options contains duplicate value %q", option)
		}
		seen[option] = struct{}{}
	}
	return nil
}

func validateOptionsColors(ctx ValidateContext) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed["options_colors"])
	if raw == "" {
		return nil
	}
	options := parseOptions(ctx.Field.WidgetParsed["options"])
	colors := parseOptionsColors(raw)
	if len(colors) == 0 {
		return nil
	}
	if len(options) == 0 {
		return fieldError(ctx, "widget tag %q requires static options", "options_colors")
	}
	if len(colors) != len(options) {
		return fieldError(ctx, "widget tag %q length must match options length, got colors=%d options=%d", "options_colors", len(colors), len(options))
	}
	for _, color := range colors {
		if !isValidOptionColor(color) {
			return fieldError(ctx, "widget tag %q contains invalid color %q", "options_colors", color)
		}
	}
	return nil
}

func validateChoiceDefaultsInOptions(ctx ValidateContext, values []string, allowExternal bool) error {
	if allowExternal || len(values) == 0 {
		return nil
	}
	options := parseOptions(ctx.Field.WidgetParsed["options"])
	if len(options) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(options))
	for _, option := range options {
		allowed[option] = struct{}{}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := allowed[value]; !ok {
			return fieldError(ctx, "widget render_default value %q must be one of options", value)
		}
	}
	return nil
}

func validateOptionsMatchOneOf(ctx ValidateContext) error {
	options := parseOptions(ctx.Field.WidgetParsed["options"])
	if len(options) == 0 {
		return nil
	}
	oneOfValues, ok := parseValidationOneOf(ctx.Field.Validate)
	if !ok {
		return nil
	}
	if !sameStringSet(options, oneOfValues) {
		return fieldError(ctx, "widget options must match validate oneof, options=%s oneof=%s", strings.Join(options, "|"), strings.Join(oneOfValues, "|"))
	}
	return nil
}

func hasChoiceCallback(ctx ValidateContext) bool {
	for _, callback := range normalizeCallbackList(ctx.Callbacks[ctx.FieldCode]) {
		if callback == "OnSelectFuzzy" {
			return true
		}
	}
	return false
}

func isValidOptionColor(value string) bool {
	return isValidRRGGBBColor(value)
}

func parseOptionsColors(raw string) []string {
	return parseOptions(raw)
}

func isValidRRGGBBColor(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 6 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func validateScalarElement(ctx ValidateContext, elem reflect.Type, expected string) error {
	elem = derefType(elem)
	if expected == "string" {
		if elem.Kind() != reflect.String {
			return fieldError(ctx, "widget %q requires []string element type, got %s", ctx.WidgetType, typeName(elem))
		}
		return nil
	}
	if !isScalarKind(elem.Kind()) {
		return fieldError(ctx, "widget %q requires scalar element type, got %s", ctx.WidgetType, typeName(elem))
	}
	return nil
}

func collectFieldTagsByCode(tags []*FieldTags) map[string][]*FieldTags {
	result := make(map[string][]*FieldTags)
	var walk func(items []*FieldTags)
	walk = func(items []*FieldTags) {
		for _, tags := range items {
			if tags == nil {
				continue
			}
			if code := tags.GetCode(); code != "" {
				result[code] = append(result[code], tags)
			}
			walk(tags.Children)
		}
	}
	walk(tags)
	return result
}

func hasDynamicChoiceTarget(tags []*FieldTags) bool {
	for _, item := range tags {
		if item == nil {
			continue
		}
		if supportsDynamicChoiceCallback(strings.TrimSpace(item.WidgetParsed["type"])) {
			return true
		}
	}
	return false
}

func supportsDynamicChoiceCallback(widgetType string) bool {
	return widgetType == TypeSelect || widgetType == TypeMultiSelect
}

func validateCallbackMapCallbacks(code string, callbacks []string) error {
	var errs []error
	hasOnSelectFuzzy := false
	normalizedCallbacks := normalizeCallbackList(callbacks)
	for _, callback := range callbacks {
		if strings.TrimSpace(callback) == "" {
			errs = append(errs, fmt.Errorf("OnSelectFuzzyMap field %q contains empty callback", code))
		}
	}
	for _, callback := range normalizedCallbacks {
		switch callback {
		case "OnSelectFuzzy":
			hasOnSelectFuzzy = true
		default:
			errs = append(errs, fmt.Errorf("OnSelectFuzzyMap field %q contains unsupported callback %q", code, callback))
		}
	}
	if !hasOnSelectFuzzy {
		errs = append(errs, fmt.Errorf("OnSelectFuzzyMap field %q must include OnSelectFuzzy callback", code))
	}
	return errors.Join(errs...)
}

func normalizeCallbackList(callbacks []string) []string {
	if len(callbacks) == 0 {
		return nil
	}
	return parseCallbackTag(strings.Join(callbacks, ","))
}

func parseCallbackTag(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		callback := strings.TrimSpace(part)
		if callback == "" {
			continue
		}
		if _, ok := seen[callback]; ok {
			continue
		}
		seen[callback] = struct{}{}
		result = append(result, callback)
	}
	return result
}

func parseValidationOneOf(validation string) ([]string, bool) {
	for _, rule := range strings.Split(validation, ",") {
		rule = strings.TrimSpace(rule)
		if strings.HasPrefix(rule, "oneof=") {
			return parseOneOfValues(strings.TrimPrefix(rule, "oneof=")), true
		}
	}
	return nil, false
}

func parseOneOfValues(raw string) []string {
	values := make([]string, 0)
	source := strings.TrimSpace(raw)
	for len(source) > 0 {
		source = strings.TrimLeft(source, " ")
		if source == "" {
			break
		}
		if source[0] == '\'' {
			end := strings.Index(source[1:], "'")
			if end < 0 {
				value := strings.TrimSpace(source[1:])
				if value != "" {
					values = append(values, value)
				}
				break
			}
			value := strings.TrimSpace(source[1 : end+1])
			if value != "" {
				values = append(values, value)
			}
			source = source[end+2:]
			continue
		}
		next := strings.Index(source, " ")
		if next < 0 {
			value := strings.TrimSpace(source)
			if value != "" {
				values = append(values, value)
			}
			break
		}
		value := strings.TrimSpace(source[:next])
		if value != "" {
			values = append(values, value)
		}
		source = source[next+1:]
	}
	return values
}

func sameStringSet(left, right []string) bool {
	leftSet := make(map[string]struct{}, len(left))
	for _, item := range left {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		leftSet[item] = struct{}{}
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		rightSet[item] = struct{}{}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for item := range leftSet {
		if _, ok := rightSet[item]; !ok {
			return false
		}
	}
	return true
}

func hasStringItem(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func isStringLikeType(typ reflect.Type) bool {
	return derefType(typ).Kind() == reflect.String
}

func isNumericType(typ reflect.Type) bool {
	kind := derefType(typ).Kind()
	return isIntegerKind(kind) || kind == reflect.Float32 || kind == reflect.Float64
}

func isIntegerType(typ reflect.Type) bool {
	return isIntegerKind(derefType(typ).Kind())
}

func isFloatType(typ reflect.Type) bool {
	kind := derefType(typ).Kind()
	return kind == reflect.Float32 || kind == reflect.Float64
}

func isIntegerKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

func isScalarType(typ reflect.Type) bool {
	return isScalarKind(derefType(typ).Kind())
}

func isScalarKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isSliceOrArray(typ reflect.Type) bool {
	kind := derefType(typ).Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func isDatetimeCompatibleType(typ reflect.Type) bool {
	typ = derefType(typ)
	if typ.Kind() == reflect.String {
		return true
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	if typ.PkgPath() == "time" && typ.Name() == "Time" {
		return true
	}
	return strings.HasSuffix(typ.PkgPath(), "/sdk/agent-app/types") && typ.Name() == "Time"
}

func isStaticDatetimeLiteral(raw string) bool {
	if _, err := time.Parse("2006-01-02 15:04:05", raw); err == nil {
		return true
	}
	if _, err := time.Parse("2006-01-02", raw); err == nil {
		return true
	}
	return false
}

func isValidDatetimeDynamicDefault(raw string) bool {
	keyword := strings.ToUpper(strings.TrimSpace(raw))
	switch keyword {
	case "CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP()", "CURRENT_DATE", "CURRENT_DATE()":
		return true
	}

	name, args, ok := parseFunctionCall(raw)
	if !ok {
		return false
	}
	name = strings.ToUpper(name)
	if name != "DATE_ADD" && name != "DATE_SUB" {
		return false
	}
	if len(args) != 2 {
		return false
	}
	return isValidDatetimeBase(args[0]) && isValidSQLInterval(args[1])
}

func parseFunctionCall(raw string) (string, []string, bool) {
	raw = strings.TrimSpace(raw)
	openIndex := strings.Index(raw, "(")
	if openIndex <= 0 || !strings.HasSuffix(raw, ")") {
		return "", nil, false
	}
	name := strings.TrimSpace(raw[:openIndex])
	if name == "" {
		return "", nil, false
	}
	argString := strings.TrimSpace(raw[openIndex+1 : len(raw)-1])
	if argString == "" {
		return name, nil, true
	}
	parts := strings.Split(argString, ",")
	args := make([]string, 0, len(parts))
	for _, part := range parts {
		arg := strings.TrimSpace(part)
		if arg != "" {
			args = append(args, arg)
		}
	}
	return name, args, true
}

func isValidDatetimeBase(raw string) bool {
	keyword := strings.ToUpper(strings.TrimSpace(raw))
	return keyword == "CURRENT_TIMESTAMP" ||
		keyword == "CURRENT_TIMESTAMP()" ||
		keyword == "CURRENT_DATE" ||
		keyword == "CURRENT_DATE()"
}

func isValidSQLInterval(raw string) bool {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 3 || strings.ToUpper(parts[0]) != "INTERVAL" {
		return false
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return false
	}
	unit := strings.TrimSuffix(strings.ToUpper(parts[2]), "S")
	switch unit {
	case "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "YEAR":
		return true
	default:
		return false
	}
}

func derefType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

func typeName(typ reflect.Type) string {
	if typ == nil {
		return "<nil>"
	}
	return typ.String()
}

func fieldError(ctx ValidateContext, format string, args ...interface{}) error {
	code := ctx.FieldCode
	if code == "" {
		code = ctx.FieldName
	}
	return fmt.Errorf("field %s (%s): %s", ctx.FieldName, code, fmt.Sprintf(format, args...))
}
