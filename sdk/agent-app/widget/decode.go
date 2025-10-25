package widget

// DecodeTable table 类型的只有一个model需要解析，table的增删改查，都是围绕着这个表进行的，无需两个model
func DecodeTable(request, tableModel interface{}) (requestFields []*Field, responseFields []*Field, err error) {

	return nil, nil, nil
}

// DecodeForm form 函数有两个，request是对应前端的提交表单参数，response是提交后后端处理后返回的响应参数
func DecodeForm(request, response interface{}) (requestFields []*Field, responseFields []*Field, err error) {

	return nil, nil, nil
}
