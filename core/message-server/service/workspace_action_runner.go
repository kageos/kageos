package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/serviceconfig"
)

const (
	WorkspaceActionClientSource = "mobile_action"
	WorkspaceActionSourceType   = "message_action"
)

type WorkspaceActionRequest struct {
	RecipientUser         string
	UserID                string
	UserEmail             string
	LeaderUsername        string
	DepartmentFullPath    string
	CompanyCode           string
	CompanyName           string
	CompanyLogoURL        string
	Channel               string
	FullCodePath          string
	SessionID             string
	ThreadKey             string
	Content               string
	DisplayContent        string
	Files                 string
	OriginalTitle         string
	TraceID               string
	SourceRef             string
	SourcePath            string
	SourceTitle           string
	SourceParentPath      string
	SourceParentTitle     string
	SourceTemplateType    string
	WorkspaceSessionTitle string
	WorkspaceRole         string
}

type WorkspaceActionSubmitResult struct {
	SessionID string
	Accepted  bool
}

type WorkspaceActionRunner struct {
	baseURL      string
	client       *http.Client
	signer       *controlauth.Signer
	startTimeout time.Duration
	runTimeout   time.Duration
}

func NewWorkspaceActionRunner(baseURL string, signer *controlauth.Signer) *WorkspaceActionRunner {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = serviceconfig.GetGatewayURL()
	}
	return &WorkspaceActionRunner{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 0,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		signer:       signer,
		startTimeout: 8 * time.Second,
		runTimeout:   30 * time.Minute,
	}
}

func (r *WorkspaceActionRunner) Submit(ctx context.Context, req WorkspaceActionRequest) (*WorkspaceActionSubmitResult, error) {
	if r == nil {
		return nil, fmt.Errorf("workspace action runner is nil")
	}
	req.RecipientUser = strings.TrimSpace(req.RecipientUser)
	req.FullCodePath = strings.TrimSpace(req.FullCodePath)
	req.Content = strings.TrimSpace(req.Content)
	if req.RecipientUser == "" {
		return nil, fmt.Errorf("工作台提交缺少用户")
	}
	if req.FullCodePath == "" {
		return nil, fmt.Errorf("工作台提交缺少目录路径")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("工作台提交内容不能为空")
	}

	started := make(chan workspaceActionStartResult, 1)
	go r.run(req, started)

	timeout := r.startTimeout
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-started:
		if result.err != nil {
			return nil, result.err
		}
		return &WorkspaceActionSubmitResult{SessionID: result.sessionID, Accepted: true}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		logger.Warnf(ctx, "[WorkspaceActionRunner] 等待工作台会话启动超时 full_code_path=%s user=%s", req.FullCodePath, req.RecipientUser)
		return &WorkspaceActionSubmitResult{Accepted: true}, nil
	}
}

type workspaceActionStartResult struct {
	sessionID string
	err       error
}

func (r *WorkspaceActionRunner) run(req WorkspaceActionRequest, started chan<- workspaceActionStartResult) {
	runTimeout := r.runTimeout
	if runTimeout <= 0 {
		runTimeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	signalStarted := workspaceActionStartSignaler(started)
	requestBody := dto.WorkspaceChatReq{
		FullCodePath: req.FullCodePath,
		SessionID:    strings.TrimSpace(req.SessionID),
		Message: dto.WorkspaceMsg{
			Content:        req.Content,
			DisplayContent: strings.TrimSpace(req.DisplayContent),
			Files:          strings.TrimSpace(req.Files),
			ContextUsage:   dto.WorkspaceMessageContextCurrentTurn,
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		signalStarted("", fmt.Errorf("序列化工作台请求失败: %w", err))
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.chatStreamURL(), bytes.NewReader(body))
	if err != nil {
		signalStarted("", fmt.Errorf("创建工作台请求失败: %w", err))
		return
	}
	applyWorkspaceActionHeaders(httpReq, req)
	if r.signer == nil {
		signalStarted("", fmt.Errorf("提交工作台失败: internal request signer is not configured"))
		return
	}
	if err := controlauth.SignHTTPRequest(httpReq, body, workspaceActionSignedHeaders(), r.signer); err != nil {
		signalStarted("", fmt.Errorf("提交工作台失败: sign internal request: %w", err))
		return
	}

	client := r.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		signalStarted("", fmt.Errorf("提交工作台失败: %w", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		signalStarted("", fmt.Errorf("提交工作台失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes))))
		return
	}

	if err := readWorkspaceActionStream(ctx, resp.Body, signalStarted); err != nil {
		logger.Errorf(ctx, "[WorkspaceActionRunner] 工作台执行流失败 user=%s full_code_path=%s err=%v", req.RecipientUser, req.FullCodePath, err)
	}
}

func workspaceActionSignedHeaders() []string {
	names := contextx.TrustedIdentityHeaderNames()
	names = append(names, contextx.TraceIdHeader)
	return names
}

func (r *WorkspaceActionRunner) chatStreamURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(r.baseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(serviceconfig.GetGatewayURL(), "/")
	}
	return baseURL + "/agent/api/v1/workspace/chat/stream"
}

func applyWorkspaceActionHeaders(httpReq *http.Request, req WorkspaceActionRequest) {
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(contextx.RequestUserHeader, strings.TrimSpace(req.RecipientUser))
	setHeaderIfNotEmpty(httpReq, contextx.UserIDHeader, req.UserID)
	setHeaderIfNotEmpty(httpReq, contextx.UserEmailHeader, req.UserEmail)
	setHeaderIfNotEmpty(httpReq, contextx.LeaderUsernameHeader, req.LeaderUsername)
	setHeaderIfNotEmpty(httpReq, contextx.DepartmentFullPathHeader, req.DepartmentFullPath)
	setHeaderIfNotEmpty(httpReq, contextx.CompanyCodeHeader, req.CompanyCode)
	setHeaderIfNotEmpty(httpReq, contextx.CompanyNameHeader, req.CompanyName)
	setHeaderIfNotEmpty(httpReq, contextx.CompanyLogoURLHeader, req.CompanyLogoURL)
	httpReq.Header.Set(contextx.ClientSourceHeader, WorkspaceActionClientSource)
	httpReq.Header.Set(contextx.SourceTypeHeader, WorkspaceActionSourceType)
	setHeaderIfNotEmpty(httpReq, contextx.TraceIdHeader, req.TraceID)
	setHeaderIfNotEmpty(httpReq, contextx.SourceRefHeader, req.SourceRef)
	setHeaderIfNotEmpty(httpReq, contextx.SourcePathHeader, req.SourcePath)
	setHeaderIfNotEmpty(httpReq, contextx.SourceTitleHeader, req.SourceTitle)
	setHeaderIfNotEmpty(httpReq, contextx.SourceParentPathHeader, req.SourceParentPath)
	setHeaderIfNotEmpty(httpReq, contextx.SourceParentTitleHeader, req.SourceParentTitle)
	setHeaderIfNotEmpty(httpReq, contextx.SourceTemplateTypeHeader, req.SourceTemplateType)
	setHeaderIfNotEmpty(httpReq, contextx.WorkspaceSessionIDHeader, req.SessionID)
	setHeaderIfNotEmpty(httpReq, contextx.WorkspaceSessionTitleHeader, req.WorkspaceSessionTitle)
	setHeaderIfNotEmpty(httpReq, contextx.WorkspaceRoleHeader, req.WorkspaceRole)
}

func setHeaderIfNotEmpty(httpReq *http.Request, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		httpReq.Header.Set(key, value)
	}
}

func workspaceActionStartSignaler(ch chan<- workspaceActionStartResult) func(string, error) {
	var sent bool
	return func(sessionID string, err error) {
		if sent {
			return
		}
		sent = true
		ch <- workspaceActionStartResult{sessionID: strings.TrimSpace(sessionID), err: err}
	}
}

func readWorkspaceActionStream(ctx context.Context, body io.Reader, signalStarted func(string, error)) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	eventName := ""
	dataLines := make([]string, 0, 4)
	started := false

	flushEvent := func() {
		if eventName == "" && len(dataLines) == 0 {
			return
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		switch eventName {
		case dto.WorkspaceStreamEventSession:
			sessionID := workspaceActionSessionID(data)
			if sessionID != "" && !started {
				started = true
				signalStarted(sessionID, nil)
			}
		case dto.WorkspaceStreamEventError:
			message := workspaceActionErrorMessage(data)
			if message == "" {
				message = "工作台执行失败"
			}
			if !started {
				started = true
				signalStarted("", errors.New(message))
			}
			logger.Warnf(ctx, "[WorkspaceActionRunner] 工作台返回错误: %s", message)
		case dto.WorkspaceStreamEventDone:
			if !started {
				started = true
				signalStarted("", nil)
			}
		}
		eventName = ""
		dataLines = dataLines[:0]
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			flushEvent()
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	flushEvent()
	if err := scanner.Err(); err != nil {
		if !started {
			signalStarted("", err)
		}
		return err
	}
	if !started {
		err := fmt.Errorf("工作台未返回会话事件")
		signalStarted("", err)
		return err
	}
	return nil
}

func workspaceActionSessionID(data string) string {
	var payload struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.SessionID)
}

func workspaceActionErrorMessage(data string) string {
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return strings.TrimSpace(data)
	}
	return strings.TrimSpace(payload.Message)
}
