package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/hub/backend/service"
	"github.com/gin-gonic/gin"
)

type PubKey struct {
	pubKeyService *service.PubKeyService
}

func NewPubKey(pubKeyService *service.PubKeyService) *PubKey {
	return &PubKey{pubKeyService: pubKeyService}
}

type GeneratePubKeyReq struct {
	Name string `json:"name"`
}

type GeneratePubKeyResp struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	KeyPrefix string `json:"key_prefix"`
}

// Generate 生成新的 pub key
func (p *PubKey) Generate(c *gin.Context) {
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "用户信息不能为空")
		return
	}

	var req GeneratePubKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Name = "默认密钥"
	}
	if req.Name == "" {
		req.Name = "默认密钥"
	}

	pubKey, fullKey, err := p.pubKeyService.Generate(username, req.Name)
	if err != nil {
		response.FailWithMessage(c, "生成密钥失败: "+err.Error())
		return
	}

	response.OkWithData(c, GeneratePubKeyResp{
		ID:        pubKey.ID,
		Name:      pubKey.Name,
		Key:       fullKey,
		KeyPrefix: pubKey.KeyPrefix,
	})
}

// List 列出当前用户的所有 pub key
func (p *PubKey) List(c *gin.Context) {
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "用户信息不能为空")
		return
	}

	keys, err := p.pubKeyService.ListByUsername(username)
	if err != nil {
		response.FailWithMessage(c, "获取密钥列表失败: "+err.Error())
		return
	}

	response.OkWithData(c, keys)
}

// Delete 删除指定的 pub key
func (p *PubKey) Delete(c *gin.Context) {
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "用户信息不能为空")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的密钥 ID")
		return
	}

	if err := p.pubKeyService.Delete(id, username); err != nil {
		response.FailWithMessage(c, "删除密钥失败: "+err.Error())
		return
	}

	response.OkWithMessage(c, "密钥已删除")
}
