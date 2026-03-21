package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// FileCache 文件缓存管理器
// 通过 hash 实现文件去重，避免重复下载相同文件。
// 磁盘文件不在此处删除，由 app-runtime 定时任务在宿主机清空 file-cache/output。
type FileCache struct {
	mu         sync.RWMutex
	cacheDir   string            // 缓存目录：/app/workplace/file-cache
	hashToPath map[string]string // hash -> 缓存文件路径
	refCount   map[string]int    // 缓存文件路径 -> 引用计数
	pathToHash map[string]string // 用户文件路径 -> hash
}

var (
	globalFileCache *FileCache
	cacheOnce       sync.Once
	cacheMu         sync.Mutex // 保护全局缓存的互斥锁
)

// GetFileCache 获取全局文件缓存实例（单例）
func GetFileCache() *FileCache {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	cacheOnce.Do(func() {
		globalFileCache = &FileCache{
			cacheDir:   "/app/workplace/file-cache",
			hashToPath: make(map[string]string),
			refCount:   make(map[string]int),
			pathToHash: make(map[string]string),
		}
		os.MkdirAll(globalFileCache.cacheDir, 0755)
	})
	return globalFileCache
}

// ResetFileCache 重置全局文件缓存（用于测试或应用重启时清理）
// 注意：只有在确保没有其他 goroutine 在使用缓存时才能调用
func ResetFileCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if globalFileCache != nil {
		globalFileCache.resetMaps()
		globalFileCache = nil
		cacheOnce = sync.Once{}
	}
}

// GetOrDownload 获取或下载文件
// 如果缓存中存在相同hash的文件，从缓存的路径复制到目标路径
// 否则直接下载到目标路径，并记录该路径到缓存
// 返回：本地文件路径、是否从缓存获取、错误
func (fc *FileCache) GetOrDownload(ctx context.Context, hash string, downloadURL string, targetPath string) (string, bool, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	var cachedPath string
	var fromCache bool

	// 1. 先检查内存映射
	if existingCachedPath, exists := fc.hashToPath[hash]; exists {
		// 检查缓存文件是否还存在
		if _, err := os.Stat(existingCachedPath); err == nil {
			cachedPath = existingCachedPath
			fromCache = true

			logger.Debugf(ctx, "[FileCache] 从内存映射找到缓存文件: hash=%s, path=%s", hash, cachedPath)
		} else {
			delete(fc.hashToPath, hash)
			delete(fc.refCount, existingCachedPath)
			logger.Debugf(ctx, "[FileCache] 内存映射中的缓存文件已不存在，从映射中移除: hash=%s, path=%s", hash, existingCachedPath)
		}
	}

	// 2. 如果缓存存在，从缓存路径复制到目标路径
	if fromCache {
		if err := copyFile(ctx, cachedPath, targetPath); err != nil {
			return "", false, fmt.Errorf("复制文件失败: %w", err)
		}
		logger.Infof(ctx, "[FileCache] 从缓存复制文件: hash=%s, cachedPath=%s -> targetPath=%s", hash, cachedPath, targetPath)
	} else {
		// 3. 如果缓存不存在，直接下载到目标路径
		logger.Infof(ctx, "[FileCache] 缓存文件不存在，直接下载到目标路径: hash=%s, url=%s, targetPath=%s", hash, downloadURL, targetPath)
		if err := downloadFile(ctx, downloadURL, targetPath); err != nil {
			return "", false, err
		}

		// 记录到缓存（缓存记录的是第一次下载的路径）
		fc.hashToPath[hash] = targetPath
		fc.refCount[targetPath] = 0 // 初始引用计数为0，创建用户文件后会增加
		logger.Infof(ctx, "[FileCache] 下载文件完成并记录到缓存: hash=%s, path=%s", hash, targetPath)
	}

	fc.pathToHash[targetPath] = hash
	cachedPath, exists := fc.hashToPath[hash]
	if exists {
		fc.refCount[cachedPath]++
	}

	return targetPath, fromCache, nil
}

// Release 释放文件引用（仅更新内存映射，不删磁盘；磁盘由 runtime 定时清空）
func (fc *FileCache) Release(filePath string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	hash, exists := fc.pathToHash[filePath]
	if !exists {
		return
	}
	delete(fc.pathToHash, filePath)

	cachedPath, exists := fc.hashToPath[hash]
	if exists {
		fc.refCount[cachedPath]--
		if fc.refCount[cachedPath] <= 0 {
			delete(fc.hashToPath, hash)
			delete(fc.refCount, cachedPath)
		}
	}
}

// copyFile 普通文件复制
func copyFile(ctx context.Context, srcPath, dstPath string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 打开源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制文件内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		os.Remove(dstPath) // 复制失败，删除不完整的文件
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	logger.Infof(ctx, "[FileCache] 普通复制完成: %s -> %s, 大小: %d bytes", srcPath, dstPath, written)
	return nil
}

// DownloadOnly 直接下载到目标路径，不写入缓存（用于无 hash 的文件）
func (fc *FileCache) DownloadOnly(ctx context.Context, downloadURL string, targetPath string) (string, error) {
	if err := downloadFile(ctx, downloadURL, targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
}

// downloadFile 下载文件到指定路径
func downloadFile(ctx context.Context, url string, filePath string) error {

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建下载请求失败: %w", err)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Minute, // 大文件下载需要较长时间
	}

	// 执行下载
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建目标文件
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer outFile.Close()

	// 复制文件内容
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(filePath) // 下载失败，删除不完整的文件
		return fmt.Errorf("写入文件失败: %w", err)
	}

	logger.Infof(ctx, "[downloadFile] 下载完成: %s, 大小: %d bytes", filePath, written)
	return nil
}

// resetMaps 清空所有内存映射（供 ResetFileCache / 测试使用）
func (fc *FileCache) resetMaps() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.hashToPath = make(map[string]string)
	fc.refCount = make(map[string]int)
	fc.pathToHash = make(map[string]string)
}

// CleanupOnShutdown 应用退出时清空内存映射（磁盘由 runtime 定时清空，此处不删文件）
func (fc *FileCache) CleanupOnShutdown() {
	fc.resetMaps()
}
