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
// 通过hash实现文件去重，避免重复下载相同文件
// 支持延迟删除机制：文件删除后延迟一段时间再真正删除，避免用户反复使用同一文件时重复下载
// 使用普通复制创建文件副本
type FileCache struct {
	mu                 sync.RWMutex
	cacheDir           string               // 缓存目录：/app/workplace/file-cache
	hashToPath         map[string]string    // hash -> 缓存文件路径
	refCount           map[string]int       // 缓存文件路径 -> 引用计数（有多少个用户文件在使用）
	pathToHash         map[string]string    // 用户文件路径 -> hash（用于清理时减少引用计数）
	pendingDelete      map[string]time.Time // 文件路径 -> 删除时间（待删除的用户文件）
	pendingCacheDelete map[string]time.Time // hash -> 删除时间（待删除的缓存文件）
	deleteDelay        time.Duration        // 延迟删除时间（默认1分钟，测试用）
	cleanupTicker      *time.Ticker         // 清理任务定时器
	cleanupStop        chan struct{}        // 停止清理任务
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
			cacheDir:           "/app/workplace/file-cache",
			hashToPath:         make(map[string]string),
			refCount:           make(map[string]int),
			pathToHash:         make(map[string]string),
			pendingDelete:      make(map[string]time.Time),
			pendingCacheDelete: make(map[string]time.Time),
			deleteDelay:        1 * time.Minute, // 测试阶段：1分钟（生产环境建议改为2小时）
			cleanupStop:        make(chan struct{}),
		}
		// 确保缓存目录存在（虽然现在不在这里存储文件，但保留目录用于其他用途）
		os.MkdirAll(globalFileCache.cacheDir, 0755)
		// 注意：缓存文件现在分散在各个 traceid 目录中，无法通过扫描恢复
		// 应用重启后，第一次请求会重新下载，然后记录到缓存
		// 启动后台清理任务
		globalFileCache.startCleanupTask()
	})
	return globalFileCache
}

// ResetFileCache 重置全局文件缓存（用于测试或应用重启时清理）
// 注意：只有在确保没有其他 goroutine 在使用缓存时才能调用
func ResetFileCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if globalFileCache != nil {
		// 停止清理任务
		globalFileCache.CleanupOnShutdown()
		globalFileCache = nil
		// 重置 sync.Once，允许重新创建
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

			// 如果缓存文件在待删除列表中，取消删除标记（文件被重新使用，延长缓存时间）
			if _, pending := fc.pendingCacheDelete[hash]; pending {
				delete(fc.pendingCacheDelete, hash)
				logger.Infof(ctx, "[FileCache] 缓存文件被重新使用，取消删除标记: hash=%s", hash)
			}
			logger.Debugf(ctx, "[FileCache] 从内存映射找到缓存文件: hash=%s, path=%s", hash, cachedPath)
		} else {
			// 缓存文件已被删除，从映射中移除
			delete(fc.hashToPath, hash)
			delete(fc.refCount, existingCachedPath)
			delete(fc.pendingCacheDelete, hash)
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

	// 4. 更新映射并增加缓存文件的引用计数
	fc.pathToHash[targetPath] = hash

	// 增加缓存文件的引用计数（每个用户文件都引用缓存文件）
	cachedPath, exists := fc.hashToPath[hash]
	if exists {
		fc.refCount[cachedPath]++
		// 如果缓存文件在待删除列表中，重置删除时间（文件被重新使用）
		if _, pending := fc.pendingCacheDelete[hash]; pending {
			delete(fc.pendingCacheDelete, hash)
			logger.Infof(ctx, "[FileCache] 缓存文件被重新使用，取消删除标记: hash=%s", hash)
		}
	}

	return targetPath, fromCache, nil
}

// Release 释放文件引用（延迟删除）
// 当文件不再使用时调用，标记为延迟删除，不立即删除
// 减少缓存文件的引用计数，如果引用计数为0，标记缓存文件为延迟删除
func (fc *FileCache) Release(filePath string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	hash, exists := fc.pathToHash[filePath]
	if !exists {
		return
	}

	// 标记用户文件为延迟删除（不立即删除）
	fc.pendingDelete[filePath] = time.Now().Add(fc.deleteDelay)
	logger.Debugf(context.Background(), "[FileCache] 标记文件为延迟删除: %s (hash: %s, 延迟时间: %v)", filePath, hash, fc.deleteDelay)

	// 减少缓存文件的引用计数
	cachedPath, exists := fc.hashToPath[hash]
	if exists {
		fc.refCount[cachedPath]--
		if fc.refCount[cachedPath] <= 0 {
			// 引用计数为0，标记缓存文件为延迟删除
			fc.pendingCacheDelete[hash] = time.Now().Add(fc.deleteDelay)
			logger.Infof(context.Background(), "[FileCache] 缓存文件引用计数为0，标记为延迟删除: hash=%s, path=%s", hash, cachedPath)
		}
	}

	// 注意：不立即删除文件，等待清理任务处理
	// 也不立即从映射中删除，清理任务会处理
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

// startCleanupTask 启动后台清理任务
// 定期检查并删除过期的待删除文件
func (fc *FileCache) startCleanupTask() {
	// 测试阶段：每1秒清理一次（便于调试，生产环境建议改为1小时）
	fc.cleanupTicker = time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-fc.cleanupTicker.C:
				fc.cleanupExpiredFiles()
			case <-fc.cleanupStop:
				// 停止清理任务
				fc.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupExpiredFiles 清理过期的待删除文件
func (fc *FileCache) cleanupExpiredFiles() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	now := time.Now()
	var expiredPaths []string
	var expiredCacheHashes []string

	// 1. 找出所有过期的用户文件
	for filePath, deleteTime := range fc.pendingDelete {
		if now.After(deleteTime) {
			// 检查文件是否还在映射中（防止在延迟期间被重新使用）
			if _, exists := fc.pathToHash[filePath]; exists {
				expiredPaths = append(expiredPaths, filePath)
			}
		}
	}

	// 2. 找出所有过期的缓存文件
	for hash, deleteTime := range fc.pendingCacheDelete {
		if now.After(deleteTime) {
			cachedPath, exists := fc.hashToPath[hash]
			if exists && fc.refCount[cachedPath] <= 0 {
				expiredCacheHashes = append(expiredCacheHashes, hash)
			}
		}
	}

	// 3. 删除过期的用户文件
	for _, filePath := range expiredPaths {
		hash, exists := fc.pathToHash[filePath]
		if !exists {
			delete(fc.pendingDelete, filePath)
			continue
		}

		// 删除用户文件
		os.Remove(filePath)
		delete(fc.pathToHash, filePath)
		delete(fc.pendingDelete, filePath)
		logger.Infof(context.Background(), "[FileCache] 清理过期用户文件: path=%s, hash=%s", filePath, hash)

		// 减少缓存文件的引用计数
		cachedPath, exists := fc.hashToPath[hash]
		if exists {
			fc.refCount[cachedPath]--
			if fc.refCount[cachedPath] <= 0 {
				// 引用计数为0，标记缓存文件为延迟删除（如果还没有标记）
				if _, pending := fc.pendingCacheDelete[hash]; !pending {
					fc.pendingCacheDelete[hash] = time.Now().Add(fc.deleteDelay)
				}
			}
		}
	}

	// 4. 删除过期的缓存文件
	for _, hash := range expiredCacheHashes {
		// 再次检查是否还在待删除列表中（可能在检查期间被重新使用，已取消删除标记）
		if _, stillPending := fc.pendingCacheDelete[hash]; !stillPending {
			// 已经被取消删除标记，跳过
			continue
		}

		cachedPath, exists := fc.hashToPath[hash]
		if !exists {
			// 内存映射中没有，但可能磁盘上还有文件，检查一下
			cachedPath = filepath.Join(fc.cacheDir, hash)
			if _, err := os.Stat(cachedPath); err != nil {
				// 磁盘上也不存在，清理标记即可
				delete(fc.pendingCacheDelete, hash)
				continue
			}
			// 磁盘上存在，但内存映射中没有，说明可能被清理任务误删了映射
			// 这种情况下不应该删除文件，而是恢复映射（文件可能还在被使用）
			// 但这里引用计数为0，说明确实没有用户文件在使用，可以删除
		}

		// 再次检查引用计数（双重检查，防止并发问题）
		refCount := 0
		if exists {
			refCount = fc.refCount[cachedPath]
		}
		if refCount <= 0 {
			// 再次检查磁盘文件是否存在（防止并发问题）
			if _, err := os.Stat(cachedPath); err == nil {
				// 真正删除缓存文件
				os.Remove(cachedPath)
				logger.Infof(context.Background(), "[FileCache] 清理过期缓存文件: hash=%s, path=%s", hash, cachedPath)
			}
			// 清理内存映射（无论文件是否存在）
			delete(fc.hashToPath, hash)
			if exists {
				delete(fc.refCount, cachedPath)
			}
			delete(fc.pendingCacheDelete, hash)
		}
	}

	if len(expiredPaths) > 0 || len(expiredCacheHashes) > 0 {
		logger.Infof(context.Background(), "[FileCache] 清理完成: 删除了 %d 个过期用户文件, %d 个过期缓存文件", len(expiredPaths), len(expiredCacheHashes))
	}
}

// CleanupOnShutdown 应用退出时立即清理所有待删除的文件
func (fc *FileCache) CleanupOnShutdown() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// 停止清理任务
	if fc.cleanupTicker != nil {
		fc.cleanupTicker.Stop()
	}
	// 发送停止信号（避免重复关闭channel）
	select {
	case <-fc.cleanupStop:
		// channel已经关闭
	default:
		close(fc.cleanupStop)
	}

	// 立即清理所有待删除的用户文件
	userFileCount := 0
	for filePath := range fc.pendingDelete {
		os.Remove(filePath)
		delete(fc.pathToHash, filePath)
		delete(fc.pendingDelete, filePath)
		userFileCount++
	}

	// 立即清理所有标记删除的缓存文件（不管引用计数，应用退出时全部清理）
	cacheFileCount := 0
	for hash := range fc.pendingCacheDelete {
		cachedPath, exists := fc.hashToPath[hash]
		if exists {
			os.Remove(cachedPath)
			delete(fc.hashToPath, hash)
			delete(fc.refCount, cachedPath)
			delete(fc.pendingCacheDelete, hash)
			cacheFileCount++
		}
	}

	if userFileCount > 0 || cacheFileCount > 0 {
		logger.Infof(context.Background(), "[FileCache] 应用退出时清理了 %d 个待删除用户文件, %d 个待删除缓存文件", userFileCount, cacheFileCount)
	}
}
