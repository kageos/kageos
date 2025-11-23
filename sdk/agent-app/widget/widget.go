package widget

const (
	TypeInput       = "input"
	TypeText        = "text"
	TypeTextArea    = "text_area"
	TypeSelect      = "select"
	TypeSwitch      = "switch"
	TypeTimestamp   = "timestamp"
	TypeUser        = "user"
	TypeID          = "ID"
	TypeNumber      = "number"
	TypeFloat       = "float"
	TypeFiles       = "files"
	TypeCheckbox    = "checkbox"
	TypeRadio       = "radio"
	TypeMultiSelect = "multiselect"
	TypeSlider      = "slider"
	TypeTable       = "table"
	TypeForm        = "form"
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
	// DataTypeTimestamp 时间类型
	DataTypeTimestamp = "timestamp"
	// DataTypeFloat 浮点数类型
	DataTypeFloat = "float"
	// DataTypeFiles 文件类型
	DataTypeFiles = "files"
	// DataTypeStruct 结构体类型
	DataTypeStruct = "struct"
	// DataTypeStructs 结构体数组类型
	DataTypeStructs = "[]struct"
)

type Widget interface {
	Config() interface{}
	Type() string
}

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
	case TypeTimestamp:
		return newTimestamp(widgetParsed)
	case TypeUser:
		return newUser(widgetParsed)
	case TypeID:
		return newID(widgetParsed)
	case TypeNumber:
		return newNumber(widgetParsed)
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
	default:
		// 默认返回Input组件，确保兜底
		return newInput(widgetParsed)
	}
}
