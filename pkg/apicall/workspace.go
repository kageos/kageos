package apicall

import (
	"github.com/ai-agent-os/ai-agent-os/dto"
	"net/http"
)

// ServiceTreeAddFunctions 向服务目录添加函数（agent-server -> workspace）
// 将生成的代码写入到工作空间对应的目录下，并更新工作空间
// async: true 表示异步处理（通过回调通知），false 表示同步处理（直接返回结果）
func ServiceTreeAddFunctions(header *Header, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	result, err := callAPI[dto.AddFunctionsResp](
		http.MethodPost,
		"/workspace/api/v1/service_tree/add_functions",
		header,
		req,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
