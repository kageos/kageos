package widget

// 数据类型
const (
	// DataTypeString 字符串类型
	DataTypeString = "string"
	// DataTypeInt 数字类型
	DataTypeInt = "int"
	// DataTypeBool 布尔类型
	DataTypeBool = "bool"

	DataTypeStrings = "[]string"
	DataTypeNumbers = "[]int"
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

type Field struct {
	Code   string     `json:"code"` //从json标签里解析，
	Desc   string     `json:"desc"` //
	Name   string     `json:"name"`
	Search string     `json:"search"`
	Data   *FieldData `json:"data"`
	Widget struct {
		Type   string      `json:"type"`
		Config interface{} `json:"config"`
	} `json:"widget"`
	Callbacks       []string `json:"callbacks"`        //
	TablePermission string   `json:"table_permission"` //read,update,create
	Validation      string   `json:"validation"`       //完全照搬github.com/go-playground/validator/v10
}

// FieldData
type FieldData struct {
	Type    string `json:"type"`    // 这里的type可以自动根据组件类型来推断出来，例如Widget类型是input，那么很显然，FieldData的type是DataTypeString
	Format  string `json:"format"`  //默认不格式化，特殊场景可以格式化成 csv/markdown/json/yaml/html 等等，这个不重要，后面再说吧
	Example string `json:"example"` //示例数据，例如 10，紧急 这种，方便前端展示一些示例数据
}
