package file

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type DownloadURLToFileReq struct {
	URL            string `json:"url" widget:"name:下载地址;type:input;placeholder:https://example.com/report.pdf" validate:"required"`
	FileName       string `json:"file_name" widget:"name:保存文件名;type:input;placeholder:可选，例如 report.pdf"`
	TimeoutSeconds int    `json:"timeout_seconds" widget:"name:超时时间(秒);type:integer;render_default:60;placeholder:请输入超时时间（秒）"`
}

type DownloadURLToFileResp struct {
	OutputFiles  string `json:"output_files" widget:"name:下载文件;type:files"`
	DownloadInfo string `json:"download_info" widget:"name:下载信息;type:text_area"`
}

func DownloadURLToFile(ctx *app.Context, resp response.Response) error {
	var req DownloadURLToFileReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoDownloadURLToFile(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoDownloadURLToFile(ctx *app.Context, req *DownloadURLToFileReq) (*DownloadURLToFileResp, error) {
	rawURL := strings.TrimSpace(req.URL)
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("下载地址不合法: %s", rawURL)
	}

	timeout := 60 * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}
	client := &http.Client{Timeout: timeout}
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造下载请求失败: %w", err)
	}
	request.Header.Set("User-Agent", "kageos-system-tool-downloader/1.0")

	responseHTTP, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer responseHTTP.Body.Close()

	if responseHTTP.StatusCode < 200 || responseHTTP.StatusCode >= 300 {
		return nil, fmt.Errorf("下载失败，HTTP 状态码: %d", responseHTTP.StatusCode)
	}

	// 文件名优先级：用户显式指定 > 响应头 > URL 路径 > Content-Type 推断扩展名。
	fileName := resolveDownloadFileName(req.FileName, parsedURL, responseHTTP)
	outputDir := ctx.GetFS().GetTraceOutputDir()
	outputPath := filepath.Join(outputDir, fileName)
	fileHandle, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer fileHandle.Close()

	written, err := io.Copy(fileHandle, responseHTTP.Body)
	if err != nil {
		return nil, fmt.Errorf("写入下载文件失败: %w", err)
	}
	if written == 0 {
		return nil, fmt.Errorf("下载结果为空文件")
	}

	outputFiles := ctx.GetFS().ResponseFiles([]string{outputPath})

	contentLength := responseHTTP.Header.Get("Content-Length")
	sizeInfo := fmt.Sprintf("%d 字节", written)
	if contentLength != "" {
		sizeInfo = contentLength + " 字节"
	}

	return &DownloadURLToFileResp{
		OutputFiles: outputFiles,
		DownloadInfo: fmt.Sprintf(
			"下载成功\nURL: %s\n文件名: %s\n大小: %s\n内容类型: %s",
			rawURL,
			fileName,
			sizeInfo,
			strings.TrimSpace(responseHTTP.Header.Get("Content-Type")),
		),
	}, nil
}

func resolveDownloadFileName(customName string, parsedURL *url.URL, resp *http.Response) string {
	if strings.TrimSpace(customName) != "" {
		return sanitizeFileName(customName, "downloaded_file")
	}

	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
			if filename := strings.TrimSpace(params["filename"]); filename != "" {
				return sanitizeFileName(filename, "downloaded_file")
			}
		}
	}

	nameFromURL := path.Base(parsedURL.Path)
	if nameFromURL == "." || nameFromURL == "/" || strings.TrimSpace(nameFromURL) == "" {
		nameFromURL = "downloaded_file"
	}
	nameFromURL = sanitizeFileName(nameFromURL, "downloaded_file")

	if filepath.Ext(nameFromURL) == "" {
		if contentType := strings.TrimSpace(resp.Header.Get("Content-Type")); contentType != "" {
			if exts, err := mime.ExtensionsByType(strings.Split(contentType, ";")[0]); err == nil && len(exts) > 0 {
				nameFromURL += exts[0]
			}
		}
	}
	return nameFromURL
}

var DownloadURLToFileTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "下载 URL 到文件",
		Desc:     `把外部下载地址保存为工作台文件，适合先下载远程 PDF、图片、压缩包等，再交给其他系统工具继续处理。`,
		Tags:     []string{"文件", "下载", "URL", "远程文件"},
		Request:  &DownloadURLToFileReq{},
		Response: &DownloadURLToFileResp{},
	},
}

func init() {
	packageContext.POST("download_url_to_file.form", DownloadURLToFile, DownloadURLToFileTemplate)
}
