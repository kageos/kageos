package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	workspaceFileProfileVersion          = "workspace_file_profile.v1"
	workspaceFileProfileMaxFiles         = 8
	workspaceFileProfileMaxDownloadBytes = 8 * 1024 * 1024
	workspaceFileProfilePreviewRows      = 5
	workspaceFileProfileMaxSheets        = 3
	workspaceFileProfileMaxTextChars     = 8000
)

var (
	workspaceFileProfileResolveFileRefs = apicall.ResolveFileRefs
	workspaceFileProfileHTTPClient      = &http.Client{Timeout: 20 * time.Second}
	workspaceFileProfileCache           sync.Map
)

type workspaceFileProfileData struct {
	ProtocolVersion string                     `json:"protocol_version"`
	Status          string                     `json:"status"`
	Files           []workspaceFileProfileItem `json:"files"`
	Truncated       bool                       `json:"truncated,omitempty"`
	Notes           []string                   `json:"notes,omitempty"`
}

type workspaceFileProfileItem struct {
	Ref              string                      `json:"ref"`
	Name             string                      `json:"name,omitempty"`
	ContentType      string                      `json:"content_type,omitempty"`
	Size             int64                       `json:"size,omitempty"`
	Kind             string                      `json:"kind"`
	Encoding         string                      `json:"encoding,omitempty"`
	Truncated        bool                        `json:"truncated,omitempty"`
	Error            string                      `json:"error,omitempty"`
	Sheets           []workspaceFileProfileSheet `json:"sheets,omitempty"`
	Columns          []string                    `json:"columns,omitempty"`
	RowCount         int                         `json:"row_count,omitempty"`
	RowCountObserved bool                        `json:"row_count_observed,omitempty"`
	SampleRows       []map[string]interface{}    `json:"sample_rows,omitempty"`
	JSONType         string                      `json:"json_type,omitempty"`
	Keys             []string                    `json:"keys,omitempty"`
	LineCount        int                         `json:"line_count,omitempty"`
	TextPreview      string                      `json:"text_preview,omitempty"`
}

type workspaceFileProfileSheet struct {
	Name             string                   `json:"name"`
	Columns          []string                 `json:"columns,omitempty"`
	RowCount         int                      `json:"row_count,omitempty"`
	RowCountObserved bool                     `json:"row_count_observed,omitempty"`
	SampleRows       []map[string]interface{} `json:"sample_rows,omitempty"`
}

func workspaceFileProfileBlockForRefs(ctx context.Context, files string) string {
	refs := splitWorkspaceFileRefs(files)
	if len(refs) == 0 {
		return ""
	}
	cacheKey := strings.Join(refs, ",")
	if cached, ok := workspaceFileProfileCache.Load(cacheKey); ok {
		if block, ok := cached.(string); ok {
			return block
		}
	}
	profile := buildWorkspaceFileProfile(ctx, refs)
	raw, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceFileProfile] marshal failed: %v", err)
		return ""
	}
	block := "<file_profile>\n" + string(raw) + "\n</file_profile>\n\n" + strings.TrimSpace(fileProfileInstruction)
	workspaceFileProfileCache.Store(cacheKey, block)
	return block
}

func splitWorkspaceFileRefs(files string) []string {
	parts := strings.Split(strings.TrimSpace(files), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
		if len(out) >= workspaceFileProfileMaxFiles {
			break
		}
	}
	return out
}

func buildWorkspaceFileProfile(ctx context.Context, refs []string) workspaceFileProfileData {
	profile := workspaceFileProfileData{
		ProtocolVersion: workspaceFileProfileVersion,
		Status:          "ok",
		Files:           make([]workspaceFileProfileItem, 0, len(refs)),
	}
	if len(refs) > workspaceFileProfileMaxFiles {
		profile.Truncated = true
		profile.Notes = append(profile.Notes, fmt.Sprintf("本轮最多自动读取 %d 个文件。", workspaceFileProfileMaxFiles))
		refs = refs[:workspaceFileProfileMaxFiles]
	}
	resp, err := workspaceFileProfileResolveFileRefs(ctx, &dto.ResolveFileRefsReq{Refs: refs, Audience: "server"})
	if err != nil || resp == nil {
		profile.Status = "error"
		for _, ref := range refs {
			profile.Files = append(profile.Files, workspaceFileProfileItem{
				Ref:   ref,
				Name:  filepath.Base(ref),
				Kind:  "unknown",
				Error: fmt.Sprintf("解析文件引用失败: %v", err),
			})
		}
		return profile
	}
	for _, resolved := range resp.Files {
		item := profileResolvedWorkspaceFile(ctx, resolved)
		if item.Error != "" && profile.Status == "ok" {
			profile.Status = "partial"
		}
		if item.Truncated {
			profile.Truncated = true
		}
		profile.Files = append(profile.Files, item)
	}
	if len(profile.Files) == 0 {
		profile.Status = "error"
		profile.Notes = append(profile.Notes, "没有可读取的文件。")
	}
	return profile
}

func profileResolvedWorkspaceFile(ctx context.Context, resolved dto.ResolvedFile) workspaceFileProfileItem {
	item := workspaceFileProfileItem{
		Ref:         firstNonEmptyString(resolved.Ref, resolved.Bucket+"/"+resolved.Key),
		Name:        firstNonEmptyString(resolved.Name, resolved.SourceName, filepath.Base(resolved.Key), filepath.Base(resolved.Ref)),
		ContentType: strings.TrimSpace(resolved.ContentType),
		Size:        resolved.Size,
		Kind:        classifyWorkspaceFileKind(resolved.Name, resolved.Key, resolved.ContentType),
	}
	if strings.TrimSpace(resolved.Error) != "" {
		item.Error = resolved.Error
		return item
	}
	if item.Kind == "unsupported_binary" {
		item.Error = "自动画像暂不读取该二进制文件类型。"
		return item
	}
	if item.Size > workspaceFileProfileMaxDownloadBytes && item.Kind == "unknown" {
		item.Error = fmt.Sprintf("文件超过自动画像大小限制 %dMB，且无法从扩展名判断为可采样文本。", workspaceFileProfileMaxDownloadBytes/1024/1024)
		return item
	}
	raw, truncated, err := downloadWorkspaceProfileFile(ctx, resolved)
	if err != nil {
		item.Error = err.Error()
		return item
	}
	item.Truncated = truncated
	if item.Kind == "unknown" {
		if !looksLikeWorkspaceText(raw) {
			item.Kind = "unsupported_binary"
			item.Error = "自动画像暂不读取该二进制文件类型。"
			return item
		}
		item.Kind = "text"
	}
	if truncated && item.Kind == "excel" {
		item.Error = fmt.Sprintf("Excel 文件超过自动画像大小限制 %dMB，未读取内容。", workspaceFileProfileMaxDownloadBytes/1024/1024)
		return item
	}
	switch item.Kind {
	case "excel":
		applyExcelWorkspaceFileProfile(&item, raw)
	case "csv", "tsv":
		applyCSVWorkspaceFileProfile(&item, raw, item.Kind)
	case "json", "jsonl":
		applyJSONWorkspaceFileProfile(&item, raw)
	default:
		applyTextWorkspaceFileProfile(&item, raw)
	}
	return item
}

func downloadWorkspaceProfileFile(ctx context.Context, resolved dto.ResolvedFile) ([]byte, bool, error) {
	rawURL := firstNonEmptyString(resolved.ServerDownloadURL, resolved.DownloadURL)
	if strings.TrimSpace(rawURL) == "" {
		return nil, false, fmt.Errorf("文件没有可用的服务端下载地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("创建下载请求失败: %w", err)
	}
	resp, err := workspaceFileProfileHTTPClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("下载文件失败: HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, workspaceFileProfileMaxDownloadBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, fmt.Errorf("读取文件失败: %w", err)
	}
	truncated := len(raw) > workspaceFileProfileMaxDownloadBytes
	if truncated {
		raw = raw[:workspaceFileProfileMaxDownloadBytes]
	}
	return raw, truncated, nil
}

func classifyWorkspaceFileKind(name, key, contentType string) string {
	lower := strings.ToLower(firstNonEmptyString(name, key))
	ext := strings.ToLower(filepath.Ext(lower))
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch ext {
	case ".xlsx", ".xlsm", ".xltx", ".xltm", ".xls":
		return "excel"
	case ".csv":
		return "csv"
	case ".tsv":
		return "tsv"
	case ".json":
		return "json"
	case ".jsonl", ".ndjson":
		return "jsonl"
	}
	if strings.Contains(ct, "spreadsheet") || strings.Contains(ct, "excel") {
		return "excel"
	}
	if strings.Contains(ct, "csv") {
		return "csv"
	}
	if strings.Contains(ct, "json") {
		return "json"
	}
	if strings.HasPrefix(ct, "text/") || isWorkspaceTextExtension(ext, lower) {
		return "text"
	}
	if isWorkspaceKnownBinaryExtension(ext) {
		return "unsupported_binary"
	}
	return "unknown"
}

func isWorkspaceTextExtension(ext, lowerName string) bool {
	if strings.HasSuffix(lowerName, "dockerfile") || strings.HasSuffix(lowerName, ".env") {
		return true
	}
	textExts := map[string]struct{}{
		".txt": {}, ".md": {}, ".markdown": {}, ".log": {}, ".go": {}, ".js": {}, ".ts": {}, ".tsx": {}, ".jsx": {},
		".vue": {}, ".py": {}, ".java": {}, ".kt": {}, ".rs": {}, ".c": {}, ".cc": {}, ".cpp": {}, ".h": {}, ".hpp": {},
		".cs": {}, ".php": {}, ".rb": {}, ".swift": {}, ".sql": {}, ".yaml": {}, ".yml": {}, ".toml": {}, ".xml": {},
		".html": {}, ".htm": {}, ".css": {}, ".scss": {}, ".sass": {}, ".less": {}, ".sh": {}, ".bash": {}, ".zsh": {},
		".ini": {}, ".conf": {}, ".cfg": {}, ".properties": {}, ".gitignore": {}, ".dockerignore": {},
	}
	_, ok := textExts[ext]
	return ok
}

func isWorkspaceKnownBinaryExtension(ext string) bool {
	binaryExts := map[string]struct{}{
		".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".bmp": {}, ".ico": {}, ".pdf": {}, ".doc": {},
		".docx": {}, ".ppt": {}, ".pptx": {}, ".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".mp3": {},
		".mp4": {}, ".mov": {}, ".avi": {}, ".mkv": {}, ".wav": {}, ".wasm": {}, ".so": {}, ".dylib": {}, ".exe": {},
	}
	_, ok := binaryExts[ext]
	return ok
}

func applyExcelWorkspaceFileProfile(item *workspaceFileProfileItem, raw []byte) {
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		item.Error = "Excel 读取失败: " + err.Error()
		return
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) > workspaceFileProfileMaxSheets {
		item.Truncated = true
		sheets = sheets[:workspaceFileProfileMaxSheets]
	}
	for _, sheetName := range sheets {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			item.Sheets = append(item.Sheets, workspaceFileProfileSheet{Name: sheetName})
			continue
		}
		rows = trimEmptyWorkspaceRows(rows)
		sheet := workspaceFileProfileSheet{Name: sheetName}
		if len(rows) > 0 {
			sheet.Columns = normalizeWorkspaceHeaders(rows[0])
			sheet.RowCount = maxInt(len(rows)-1, 0)
			sheet.SampleRows = workspaceRowsToObjects(sheet.Columns, rows[1:], workspaceFileProfilePreviewRows)
		}
		item.Sheets = append(item.Sheets, sheet)
	}
}

func applyCSVWorkspaceFileProfile(item *workspaceFileProfileItem, raw []byte, kind string) {
	text, encodingName := decodeWorkspaceText(raw)
	item.Encoding = encodingName
	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	if kind == "tsv" {
		reader.Comma = '\t'
	}
	rows := make([][]string, 0, workspaceFileProfilePreviewRows+1)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			item.Error = "CSV 读取失败: " + err.Error()
			return
		}
		rows = append(rows, row)
	}
	rows = trimEmptyWorkspaceRows(rows)
	if len(rows) == 0 {
		return
	}
	item.Columns = normalizeWorkspaceHeaders(rows[0])
	item.RowCount = maxInt(len(rows)-1, 0)
	item.RowCountObserved = item.Truncated
	item.SampleRows = workspaceRowsToObjects(item.Columns, rows[1:], workspaceFileProfilePreviewRows)
}

func applyJSONWorkspaceFileProfile(item *workspaceFileProfileItem, raw []byte) {
	text, encodingName := decodeWorkspaceText(raw)
	item.Encoding = encodingName
	var value interface{}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		if item.Kind == "jsonl" {
			applyJSONLWorkspaceFileProfile(item, text)
			return
		}
		item.Error = "JSON 读取失败: " + err.Error()
		return
	}
	applyJSONValueWorkspaceFileProfile(item, value)
}

func applyJSONLWorkspaceFileProfile(item *workspaceFileProfileItem, text string) {
	lines := strings.Split(text, "\n")
	rows := make([]map[string]interface{}, 0, workspaceFileProfilePreviewRows)
	keys := map[string]struct{}{}
	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		count++
		var value interface{}
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			item.Error = "JSONL 读取失败: " + err.Error()
			return
		}
		if obj, ok := value.(map[string]interface{}); ok {
			for key := range obj {
				keys[key] = struct{}{}
			}
			if len(rows) < workspaceFileProfilePreviewRows {
				rows = append(rows, obj)
			}
		}
	}
	item.JSONType = "jsonl"
	item.RowCount = count
	item.Columns = sortedWorkspaceKeys(keys)
	item.Keys = item.Columns
	item.SampleRows = rows
}

func applyJSONValueWorkspaceFileProfile(item *workspaceFileProfileItem, value interface{}) {
	switch v := value.(type) {
	case []interface{}:
		item.JSONType = "array"
		item.RowCount = len(v)
		rows, keys := sampleJSONObjects(v)
		item.Columns = sortedWorkspaceKeys(keys)
		item.Keys = item.Columns
		item.SampleRows = rows
	case map[string]interface{}:
		item.JSONType = "object"
		item.Keys = sortedWorkspaceMapKeys(v)
		for _, key := range item.Keys {
			if arr, ok := v[key].([]interface{}); ok {
				rows, keys := sampleJSONObjects(arr)
				if len(rows) > 0 {
					item.JSONType = "object_with_array:" + key
					item.RowCount = len(arr)
					item.Columns = sortedWorkspaceKeys(keys)
					item.SampleRows = rows
					return
				}
			}
		}
		if raw, err := json.MarshalIndent(v, "", "  "); err == nil {
			item.TextPreview = truncateWorkspaceString(string(raw), workspaceFileProfileMaxTextChars)
		}
	default:
		item.JSONType = fmt.Sprintf("%T", value)
		if raw, err := json.MarshalIndent(value, "", "  "); err == nil {
			item.TextPreview = truncateWorkspaceString(string(raw), workspaceFileProfileMaxTextChars)
		}
	}
}

func applyTextWorkspaceFileProfile(item *workspaceFileProfileItem, raw []byte) {
	text, encodingName := decodeWorkspaceText(raw)
	item.Encoding = encodingName
	item.LineCount = strings.Count(text, "\n")
	if text != "" && !strings.HasSuffix(text, "\n") {
		item.LineCount++
	}
	item.TextPreview = truncateWorkspaceString(text, workspaceFileProfileMaxTextChars)
}

func sampleJSONObjects(values []interface{}) ([]map[string]interface{}, map[string]struct{}) {
	rows := make([]map[string]interface{}, 0, workspaceFileProfilePreviewRows)
	keys := map[string]struct{}{}
	for _, value := range values {
		obj, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		for key := range obj {
			keys[key] = struct{}{}
		}
		if len(rows) < workspaceFileProfilePreviewRows {
			rows = append(rows, obj)
		}
	}
	return rows, keys
}

func decodeWorkspaceText(raw []byte) (string, string) {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if len(raw) >= 2 {
		switch {
		case raw[0] == 0xFF && raw[1] == 0xFE:
			return decodeUTF16WorkspaceText(raw[2:], false), "utf-16le"
		case raw[0] == 0xFE && raw[1] == 0xFF:
			return decodeUTF16WorkspaceText(raw[2:], true), "utf-16be"
		}
	}
	if utf8.Valid(raw) {
		return string(raw), "utf-8"
	}
	if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(raw); err == nil && utf8.Valid(decoded) {
		return string(decoded), "gb18030"
	}
	return strings.ToValidUTF8(string(raw), ""), "unknown"
}

func decodeUTF16WorkspaceText(raw []byte, bigEndian bool) string {
	u16 := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		if bigEndian {
			u16 = append(u16, uint16(raw[i])<<8|uint16(raw[i+1]))
		} else {
			u16 = append(u16, uint16(raw[i+1])<<8|uint16(raw[i]))
		}
	}
	return string(utf16.Decode(u16))
}

func looksLikeWorkspaceText(raw []byte) bool {
	if len(raw) == 0 {
		return true
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return false
	}
	if utf8.Valid(raw) {
		return true
	}
	decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(raw)
	return err == nil && utf8.Valid(decoded)
}

func trimEmptyWorkspaceRows(rows [][]string) [][]string {
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		empty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				empty = false
				break
			}
		}
		if !empty {
			out = append(out, row)
		}
	}
	return out
}

func normalizeWorkspaceHeaders(row []string) []string {
	out := make([]string, len(row))
	seen := map[string]int{}
	for i, cell := range row {
		name := strings.TrimSpace(cell)
		if name == "" {
			name = fmt.Sprintf("列%d", i+1)
		}
		seen[name]++
		if seen[name] > 1 {
			name = fmt.Sprintf("%s_%d", name, seen[name])
		}
		out[i] = name
	}
	return out
}

func workspaceRowsToObjects(headers []string, rows [][]string, limit int) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, minInt(len(rows), limit))
	for _, row := range rows {
		if len(out) >= limit {
			break
		}
		obj := make(map[string]interface{}, len(headers))
		for i, header := range headers {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			obj[header] = value
		}
		out = append(out, obj)
	}
	return out
}

func sortedWorkspaceMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedWorkspaceKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func truncateWorkspaceString(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "\n...（已截断）"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
