package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code     int                    `json:"code"`
	Data     interface{}            `json:"data"`
	Msg      string                 `json:"msg"`
	Metadata map[string]interface{} `json:"metadata"`
}

const (
	ERROR   = 7
	SUCCESS = 0
)

func Result(code int, data interface{}, msg string, c *gin.Context, metadata ...map[string]interface{}) {

	if len(metadata) > 0 {
		c.JSON(http.StatusOK, Response{
			Code:     code,
			Data:     data,
			Msg:      msg,
			Metadata: metadata[0],
		})
	} else {
		// 开始时间
		c.JSON(http.StatusOK, Response{
			Code: code,
			Data: data,
			Msg:  msg,
		})
	}
}

func Ok(c *gin.Context) {
	Result(SUCCESS, map[string]interface{}{}, "操作成功", c)
}

func OkWithMessage(c *gin.Context, message string) {
	Result(SUCCESS, map[string]interface{}{}, message, c)
}

func OkWithData(c *gin.Context, data interface{}, metadata ...map[string]interface{}) {
	Result(SUCCESS, data, "成功", c, metadata...)
}

func OkWithDetailed(c *gin.Context, data interface{}, message string) {
	Result(SUCCESS, data, message, c)
}

func Fail(c *gin.Context) {
	Result(ERROR, map[string]interface{}{}, "操作失败", c)
}

func FailWithMessage(c *gin.Context, message string, metadata ...map[string]interface{}) {
	Result(ERROR, map[string]interface{}{}, message, c, metadata...)
}

func NoAuth(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: 7,
		Data: nil,
		Msg:  message,
	})
}

func FailWithDetailed(c *gin.Context, data interface{}, message string) {
	Result(ERROR, data, message, c)
}

// PermissionDenied 权限不足响应（返回 403 状态码）
// 参数：
//   - c: Gin 上下文
//   - message: 错误消息
//   - permissionInfo: 权限详细信息，包含资源路径、操作类型、权限点等，方便前端构造申请权限的提示
//
// 返回格式：
//   {
//     "code": 7,
//     "data": {
//       "resource_path": "/luobei/operations/tools/pdftools/to_images",
//       "action": "table:search",
//       "action_display": "表格查询",
//       "apply_url": "/permissions/apply?resource=/luobei/operations/tools/pdftools/to_images&action=table:search"
//     },
//     "msg": "无权限查询该表格"
//   }
func PermissionDenied(c *gin.Context, message string, permissionInfo map[string]interface{}) {
	c.JSON(http.StatusForbidden, Response{
		Code: ERROR,
		Data: permissionInfo,
		Msg:  message,
	})
	c.Abort()
}
