package message

import (
	"net/http"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
)

type sendMessageEnvelope struct {
	Meta    sendMessageMeta    `json:"meta"`
	Message sendMessagePayload `json:"message"`
}

type sendMessageMeta struct {
	FullCodePath string `json:"full_code_path"`
}

type sendMessagePayload struct {
	ToUsers       string `json:"to_users"`
	ToDepartments string `json:"to_departments"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	ContentType   string `json:"content_type"`
}

type sendMessageResult struct {
	Message       string `json:"message"`
	From          string `json:"from"`
	FullCodePath  string `json:"full_code_path"`
	ToUsers       string `json:"to_users"`
	ToDepartments string `json:"to_departments"`
	ContentType   string `json:"content_type"`
}

func sendMessage(ctx *app.Context, payload sendMessagePayload) (*sendMessageResult, error) {
	req := &sendMessageEnvelope{
		Meta: sendMessageMeta{
			FullCodePath: strings.TrimSpace(ctx.GetFullCodePath()),
		},
		Message: sendMessagePayload{
			ToUsers:       strings.TrimSpace(payload.ToUsers),
			ToDepartments: strings.TrimSpace(payload.ToDepartments),
			Title:         strings.TrimSpace(payload.Title),
			Content:       strings.TrimSpace(payload.Content),
			ContentType:   strings.TrimSpace(payload.ContentType),
		},
	}
	var result sendMessageResult
	if err := ctx.APICall(http.MethodPost, "/message/api/v1/send", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
