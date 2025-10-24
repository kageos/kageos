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

func FailWithMessage(c *gin.Context, message string) {
	Result(ERROR, map[string]interface{}{}, message, c)
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
