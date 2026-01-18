package apicall

import (
	"net/http"
)

// CallFormAPI 调用 Form API（泛型版本，支持任意请求和响应类型）
// TReq: 请求类型
// TResp: 响应类型
// formPath: Form 函数路径（full-code-path），例如：/system/official/agent/plugin/table_parse
// header: 请求头信息（包含token、trace_id等）
// req: Form 提交请求（任意类型）
// 返回: Form 提交响应（任意类型）
func CallFormAPI[TReq, TResp any](header *Header, formPath string, req TReq) (*TResp, error) {
	// 构建完整路径
	path := "/workspace/api/v1/form/submit" + formPath

	result, err := callAPI[TResp](
		http.MethodPost,
		path,
		header,
		req,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
