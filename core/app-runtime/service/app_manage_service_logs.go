package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	sharedDto "github.com/kageos/kageos/dto"

	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/gitx"
	"github.com/kageos/kageos/pkg/logger"
)

func (s *AppManageService) appStartupNotificationTimeout() time.Duration {
	if s.runtimeConfig == nil {
		return time.Duration((&appconfig.AppRuntimeConfig{}).GetAppStartupNotificationTimeout()) * time.Second
	}
	return time.Duration(s.runtimeConfig.GetAppStartupNotificationTimeout()) * time.Second
}

type lineRange struct {
	Start int
	End   int
}

// ReadAppLog 读取应用版本日志（支持 tail 和关键词检索）
func (s *AppManageService) ReadAppLog(ctx context.Context, req *sharedDto.ReadAppLogRuntimeReq) (*sharedDto.ReadAppLogRuntimeResp, error) {
	lines := req.Lines
	if lines <= 0 {
		lines = 200
	}
	if lines > 1000 {
		lines = 1000
	}
	contextLines := req.ContextLines
	if contextLines < 0 {
		contextLines = 0
	}
	if contextLines == 0 {
		contextLines = 2
	}
	if contextLines > 5 {
		contextLines = 5
	}
	maxMatches := req.MaxMatches
	if maxMatches <= 0 {
		maxMatches = 50
	}
	if maxMatches > 200 {
		maxMatches = 200
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		currentVersion, err := s.getCurrentVersion(ctx, req.User, req.App)
		if err != nil {
			return nil, fmt.Errorf("读取当前版本失败: %w", err)
		}
		if strings.TrimSpace(currentVersion) == "" {
			return nil, fmt.Errorf("当前版本为空，无法定位日志文件")
		}
		version = strings.TrimSpace(currentVersion)
	}

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), req.User, req.App)
	logFileName := appPaths.LogFileName(version)
	logFilePath := appPaths.LogFile(version)

	f, err := os.Open(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("日志文件不存在: %s", logFileName)
		}
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer f.Close()

	allLines := make([]string, 0, 1024)
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}
	totalLines := len(allLines)

	resp := &sharedDto.ReadAppLogRuntimeResp{
		Success:         true,
		Message:         "读取成功",
		ResolvedVersion: version,
		LogFile:         logFileName,
		TotalLines:      totalLines,
	}
	if totalLines == 0 {
		resp.Message = "日志为空"
		return resp, nil
	}

	keyword := req.Keyword
	if strings.TrimSpace(keyword) == "" {
		start := totalLines - lines
		if start < 0 {
			start = 0
		}
		out := allLines[start:totalLines]
		resp.ReturnedLines = len(out)
		resp.Truncated = start > 0
		resp.Content = strings.Join(out, "\n")
		return resp, nil
	}

	matchRanges := make([]lineRange, 0, maxMatches)
	matchCount := 0
	needle := keyword
	if req.IgnoreCase {
		needle = strings.ToLower(needle)
	}
	for i, line := range allLines {
		hay := line
		if req.IgnoreCase {
			hay = strings.ToLower(hay)
		}
		if strings.Contains(hay, needle) {
			matchCount++
			if len(matchRanges) < maxMatches {
				start := i - contextLines
				if start < 0 {
					start = 0
				}
				end := i + contextLines
				if end >= totalLines {
					end = totalLines - 1
				}
				matchRanges = append(matchRanges, lineRange{Start: start, End: end})
			}
		}
	}
	resp.MatchCount = matchCount
	if len(matchRanges) == 0 {
		resp.Message = "未匹配到关键词"
		return resp, nil
	}

	merged := mergeLineRanges(matchRanges)
	result := make([]string, 0, lines)
	for _, rg := range merged {
		for i := rg.Start; i <= rg.End; i++ {
			if len(result) >= lines {
				resp.Truncated = true
				break
			}
			result = append(result, fmt.Sprintf("%d|%s", i+1, allLines[i]))
		}
		if resp.Truncated {
			break
		}
	}
	if matchCount > maxMatches {
		resp.Truncated = true
	}
	resp.ReturnedLines = len(result)
	resp.Content = strings.Join(result, "\n")
	return resp, nil
}

func mergeLineRanges(ranges []lineRange) []lineRange {
	if len(ranges) == 0 {
		return nil
	}
	merged := make([]lineRange, 0, len(ranges))
	current := ranges[0]
	for i := 1; i < len(ranges); i++ {
		r := ranges[i]
		if r.Start <= current.End+1 {
			if r.End > current.End {
				current.End = r.End
			}
			continue
		}
		merged = append(merged, current)
		current = r
	}
	merged = append(merged, current)
	return merged
}

// GitCommitMessage Git 提交消息结构体
type GitCommitMessage struct {
	AppVersion        string `json:"app_version"`        // 应用版本号
	Requirement       string `json:"requirement"`        // 变更需求
	ChangeDescription string `json:"change_description"` // 变更描述
	Summary           string `json:"summary"`            // 变更摘要
	Timestamp         string `json:"timestamp"`          // 时间戳
}

// commitToGit 提交代码到 Git，返回 commit hash
func (s *AppManageService) commitToGit(
	ctx context.Context,
	user, app, version string,
	requirement, changeDescription string,
) (string, error) {
	// 1. 获取应用代码目录
	appCodeDir := newRuntimeAppPaths(s.config.GetBasePath(), user, app).APIDir()

	// 2. 从 ctx 获取用户名称
	authorName := contextx.GetRequestUser(ctx)
	if authorName == "" {
		authorName = user // 如果 ctx 中没有用户信息，使用 user 参数
	}

	// 3. 获取邮箱后缀（从配置读取）
	emailSuffix := s.config.GetGitEmailSuffix()
	if emailSuffix == "" {
		emailSuffix = "kageos.ai" // 默认后缀
	}

	// 4. 构建邮箱：{user}@{email_suffix}
	if authorName == "" || authorName == "system" {
		authorName = "system"
	}
	authorEmail := fmt.Sprintf("%s@%s", authorName, emailSuffix)

	// 4. 初始化或打开 Git 仓库
	gitRepo, err := gitx.InitOrOpen(appCodeDir, authorName, authorEmail)
	if err != nil {
		return "", fmt.Errorf("初始化 Git 仓库失败: %w", err)
	}

	// 5. 构建 commit message（JSON 格式）
	commitMsg := GitCommitMessage{
		AppVersion:        version,
		Requirement:       requirement,
		ChangeDescription: changeDescription,
		Timestamp:         time.Now().Format(time.RFC3339),
	}

	// 构建 summary
	if requirement != "" && changeDescription != "" {
		commitMsg.Summary = fmt.Sprintf("需求：%s\n\n变更描述：%s", requirement, changeDescription)
	} else if requirement != "" {
		commitMsg.Summary = requirement
	} else if changeDescription != "" {
		commitMsg.Summary = changeDescription
	}

	commitJSON, err := json.Marshal(commitMsg)
	if err != nil {
		return "", fmt.Errorf("序列化 commit message 失败: %w", err)
	}

	// 6. 添加所有文件并提交
	commitHash, err := gitRepo.AddAllAndCommit(string(commitJSON))
	if err != nil {
		return "", fmt.Errorf("Git 提交失败: %w", err)
	}

	logger.Infof(ctx, "[commitToGit] Git 提交成功: user=%s, app=%s, version=%s, commitHash=%s",
		user, app, version, commitHash)

	return commitHash, nil
}
