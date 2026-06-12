package scheduledsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/contextx"
)

var _ Adapter = (*HTTPAdapter)(nil)

type HTTPAdapter struct {
	baseURL string
	client  *http.Client
}

func NewHTTPAdapter(baseURL string, client *http.Client) *HTTPAdapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPAdapter{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client:  client,
	}
}

func (a *HTTPAdapter) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	var out Task
	if err := a.doJSON(ctx, http.MethodPost, "/tasks", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) UpdateTask(ctx context.Context, taskID int64, req UpdateTaskRequest) (*Task, error) {
	var out Task
	if err := a.doJSON(ctx, http.MethodPut, fmt.Sprintf("/tasks/%d", taskID), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) PauseTask(ctx context.Context, taskID int64) error {
	return a.doJSON(ctx, http.MethodPost, fmt.Sprintf("/tasks/%d/pause", taskID), nil, nil, nil)
}

func (a *HTTPAdapter) ResumeTask(ctx context.Context, taskID int64) error {
	return a.doJSON(ctx, http.MethodPost, fmt.Sprintf("/tasks/%d/resume", taskID), nil, nil, nil)
}

func (a *HTTPAdapter) CancelTask(ctx context.Context, taskID int64) error {
	return a.doJSON(ctx, http.MethodPost, fmt.Sprintf("/tasks/%d/cancel", taskID), nil, nil, nil)
}

func (a *HTTPAdapter) DeleteTask(ctx context.Context, taskID int64) error {
	return a.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/tasks/%d", taskID), nil, nil, nil)
}

func (a *HTTPAdapter) RunNow(ctx context.Context, taskID int64) (*Execution, error) {
	var out Execution
	if err := a.doJSON(ctx, http.MethodPost, fmt.Sprintf("/tasks/%d/run_now", taskID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) GetTask(ctx context.Context, taskID int64) (*Task, error) {
	var out Task
	if err := a.doJSON(ctx, http.MethodGet, fmt.Sprintf("/tasks/%d", taskID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) ListTasks(ctx context.Context, req ListTasksRequest) (*ListTasksResponse, error) {
	query := url.Values{}
	setQuery(query, "executor_key", req.ExecutorKey)
	setQuery(query, "status", req.Status)
	setQuery(query, "category", req.Category)
	setQuery(query, "source_type", req.SourceType)
	setQuery(query, "source_ref", req.SourceRef)
	setQuery(query, "resource_scope", req.ResourceScope)
	setQuery(query, "resource_key", req.ResourceKey)
	setQuery(query, "created_by", req.CreatedBy)
	setQueryInt(query, "page", req.Page)
	setQueryInt(query, "page_size", req.PageSize)
	var out ListTasksResponse
	if err := a.doJSON(ctx, http.MethodGet, "/tasks", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) GetExecution(ctx context.Context, taskID, executionID int64) (*Execution, error) {
	var out Execution
	path := fmt.Sprintf("/tasks/%d/executions/%d", taskID, executionID)
	if err := a.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) ListExecutions(ctx context.Context, taskID int64, req ListExecutionsRequest) (*ListExecutionsResponse, error) {
	query := url.Values{}
	setQuery(query, "status", req.Status)
	setQueryInt(query, "page", req.Page)
	setQueryInt(query, "page_size", req.PageSize)
	var out ListExecutionsResponse
	if err := a.doJSON(ctx, http.MethodGet, fmt.Sprintf("/tasks/%d/executions", taskID), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *HTTPAdapter) MarkExecutionStarted(ctx context.Context, req MarkExecutionStartedRequest) error {
	return a.doJSON(ctx, http.MethodPost, "/executions/started", nil, req, nil)
}

func (a *HTTPAdapter) MarkExecutionHeartbeat(ctx context.Context, req MarkExecutionHeartbeatRequest) error {
	return a.doJSON(ctx, http.MethodPost, "/executions/heartbeat", nil, req, nil)
}

func (a *HTTPAdapter) MarkExecutionFinished(ctx context.Context, req MarkExecutionFinishedRequest) error {
	return a.doJSON(ctx, http.MethodPost, "/executions/finished", nil, req, nil)
}

func (a *HTTPAdapter) doJSON(ctx context.Context, method, path string, query url.Values, in interface{}, out interface{}) error {
	if strings.TrimSpace(a.baseURL) == "" {
		return fmt.Errorf("scheduledsdk: base url is required")
	}
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	endpoint := a.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	applyContextHeaders(req, ctx)

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &errResp) == nil && strings.TrimSpace(errResp.Error) != "" {
			return fmt.Errorf("scheduledsdk: %s", errResp.Error)
		}
		return fmt.Errorf("scheduledsdk: http %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func applyContextHeaders(req *http.Request, ctx context.Context) {
	if token := contextx.GetToken(ctx); token != "" {
		req.Header.Set(contextx.TokenHeader, token)
	}
	if traceID := contextx.GetTraceId(ctx); traceID != "" {
		req.Header.Set(contextx.TraceIdHeader, traceID)
	}
	if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
		req.Header.Set(contextx.RequestUserHeader, requestUser)
	}
	if departmentFullPath := contextx.GetRequestDepartmentFullPath(ctx); departmentFullPath != "" {
		req.Header.Set(contextx.DepartmentFullPathHeader, departmentFullPath)
	}
	if companyCode := contextx.GetRequestCompanyCode(ctx); companyCode != "" {
		req.Header.Set(contextx.CompanyCodeHeader, companyCode)
	}
	if companyName := contextx.GetRequestCompanyName(ctx); companyName != "" {
		req.Header.Set(contextx.CompanyNameHeader, companyName)
	}
	if companyLogoURL := contextx.GetRequestCompanyLogoURL(ctx); companyLogoURL != "" {
		req.Header.Set(contextx.CompanyLogoURLHeader, companyLogoURL)
	}
	if clientSource := contextx.GetClientSource(ctx); clientSource != "" {
		req.Header.Set(contextx.ClientSourceHeader, clientSource)
	}
	if sourceType := contextx.GetSourceType(ctx); sourceType != "" {
		req.Header.Set(contextx.SourceTypeHeader, sourceType)
	}
	if sourceRef := contextx.GetSourceRef(ctx); sourceRef != "" {
		req.Header.Set(contextx.SourceRefHeader, sourceRef)
	}
}

func setQuery(query url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		query.Set(key, value)
	}
}

func setQueryInt(query url.Values, key string, value int) {
	if value > 0 {
		query.Set(key, strconv.Itoa(value))
	}
}
