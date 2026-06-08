package widget

const (
	TypeInput       = "input"
	TypeText        = "text"
	TypeTextArea    = "text_area"
	TypeSelect      = "select"
	TypeSwitch      = "switch"
	TypeDatetime    = "datetime"
	TypeUser        = "user"
	TypeUsers       = "users"
	TypeDepartment  = "department"
	TypeDepartments = "departments"
	TypeID          = "ID"
	TypeInteger     = "integer"
	TypeFloat       = "float"
	TypeFiles       = "files"
	TypeCheckbox    = "checkbox"
	TypeRadio       = "radio"
	TypeMultiSelect = "multiselect"
	TypeSlider      = "slider"
	TypeRate        = "rate"
	TypeColor       = "color"
	TypeRichText    = "richtext"
	TypeTable       = "table"
	TypeForm        = "form"
	TypeLink        = "link"
	TypeProgress    = "progress"
	TypeList        = "list"
)

// 数据类型
const (
	// DataTypeString 字符串类型
	DataTypeString = "string"
	// DataTypeInt 数字类型
	DataTypeInt = "int"
	// DataTypeBool 布尔类型
	DataTypeBool = "bool"

	DataTypeStrings = "[]string"
	DataTypeInts    = "[]int"
	DataTypeFloats  = "[]float"
	// DataTypeFloat 浮点数类型
	DataTypeFloat = "float"
	// DataTypeStruct 结构体类型
	DataTypeStruct = "struct"
	// DataTypeStructs 结构体数组类型
	DataTypeStructs = "[]struct"
)

// Widget 是所有组件配置对象的最小接口。
//
// 每个组件源码需要提供三件事：
// - 一个配置 struct，例如 Select/DateTime/Files；
// - 一个 newXxx(widgetParsed) 工厂，把 widget tag 解析成配置 struct；
// - 一个 WidgetValidator 注册到 validator.go，用于启动期校验 Go 类型和关键 tag。
//
// Config 返回的对象会进入 function schema 的 widget.config；
// Type 返回的字符串必须存在于 supportedWidgetTypes，并且前端也要有同名组件注册。
type Widget interface {
	Config() interface{}
	Type() string
}

// supportedWidgetTypes 是 SDK 允许输出到 schema 的组件白名单。
//
// 新增组件时必须同步：
// - 在这里添加 TypeXxx；
// - 在 NewWidget 中添加工厂分支；
// - 在组件文件 init 中 RegisterWidgetValidator；
// - 在前端 widget registry/types 中添加渲染支持。
var supportedWidgetTypes = []string{
	TypeInput,
	TypeText,
	TypeTextArea,
	TypeSelect,
	TypeSwitch,
	TypeDatetime,
	TypeUser,
	TypeUsers,
	TypeDepartment,
	TypeDepartments,
	TypeID,
	TypeInteger,
	TypeFloat,
	TypeFiles,
	TypeCheckbox,
	TypeRadio,
	TypeMultiSelect,
	TypeSlider,
	TypeRate,
	TypeColor,
	TypeRichText,
	TypeTable,
	TypeForm,
	TypeLink,
	TypeProgress,
	TypeList,
}

var supportedWidgetTypeSet = buildSupportedWidgetTypeSet()

func buildSupportedWidgetTypeSet() map[string]struct{} {
	result := make(map[string]struct{}, len(supportedWidgetTypes))
	for _, widgetType := range supportedWidgetTypes {
		result[widgetType] = struct{}{}
	}
	return result
}

func SupportedTypes() []string {
	result := make([]string, len(supportedWidgetTypes))
	copy(result, supportedWidgetTypes)
	return result
}

func IsSupportedType(widgetType string) bool {
	_, ok := supportedWidgetTypeSet[widgetType]
	return ok
}

// NewWidget 根据 widget tag 中的 type 创建具体配置。
//
// 注意：这里的 default 分支只用于兜底旧数据或上层未校验调用；
// 正常 DecodeForm/DecodeTable 会先走 ValidateFieldTags，未知 widget type 会启动失败。
func NewWidget(widgetType string, widgetParsed map[string]string) Widget {
	switch widgetType {
	case TypeFiles:
		return newFiles(widgetParsed)
	case TypeInput:
		return newInput(widgetParsed)
	case TypeTextArea:
		return newTextArea(widgetParsed)
	case TypeSelect:
		return newSelect(widgetParsed)
	case TypeMultiSelect:
		return newMultiSelect(widgetParsed)
	case TypeSwitch:
		return newSwitch(widgetParsed)
	case TypeDatetime:
		return newDateTime(widgetParsed)
	case TypeUser:
		return newUser(widgetParsed)
	case TypeUsers:
		return newUsers(widgetParsed)
	case TypeDepartment:
		return newDepartment(widgetParsed)
	case TypeDepartments:
		return newDepartments(widgetParsed)
	case TypeID:
		return newID(widgetParsed)
	case TypeInteger:
		return newInteger(widgetParsed)
	case TypeFloat:
		return newFloat(widgetParsed)
	case TypeCheckbox:
		return newCheckbox(widgetParsed)
	case TypeRadio:
		return newRadio(widgetParsed)
	case TypeText:
		return newText(widgetParsed)
	case TypeSlider:
		return newSlider(widgetParsed)
	case TypeRate:
		return newRate(widgetParsed)
	case TypeColor:
		return newColor(widgetParsed)
	case TypeRichText:
		return newRichText(widgetParsed)
	case TypeLink:
		return newLink(widgetParsed)
	case TypeProgress:
		return newProgress(widgetParsed)
	case TypeList:
		return newList(widgetParsed)
	default:
		// 默认返回Input组件，确保兜底
		return newInput(widgetParsed)
	}
}
