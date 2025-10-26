package widget

const (
	TypeInput     = "input"
	TypeTextArea  = "text_area"
	TypeSelect    = "select"
	TypeSwitch    = "switch"
	TypeTimestamp = "timestamp"
	TypeUser      = "user"
	TypeID        = "ID"
	TypeNumber    = "number"
	TypeFloat     = "float"
	TypeFiles     = "files"
	TypeCheckbox  = "checkbox"
	TypeRadio     = "radio"
)

type Widget interface {
	Config() interface{}
	Type() string
}

func NewWidget(widgetType string, widgetParsed map[string]string) Widget {
	switch widgetType {
	case TypeInput:
		return newInput(widgetParsed)
	case TypeTextArea:
		return newTextArea(widgetParsed)
	case TypeSelect:
		return newSelect(widgetParsed)
	case TypeSwitch:
		return newSwitch(widgetParsed)
	case TypeTimestamp:
		return newTimestamp(widgetParsed)
	case TypeUser:
		return newUser(widgetParsed)
	case TypeID:
		return newID(widgetParsed)
	default:
		// 默认返回Input组件，确保兜底
		return newInput(widgetParsed)
	}
}
