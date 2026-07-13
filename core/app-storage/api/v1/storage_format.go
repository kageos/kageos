package v1

import (
	"fmt"
	"net"

	"github.com/kageos/kageos/core/app-storage/storage"
)

// trimLeadingSlash 移除前导斜杠（用于 *key 路由参数）
// 注意：此函数与 service/storage_service.go 中的 trimLeadingSlash 功能相同，但保留在各自包中以避免循环依赖
func trimLeadingSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// normalizeIP 规范化IP地址（将IPv6的::1转换为127.0.0.1）
func normalizeIP(ip string) string {
	if ip == storage.IPv6Loopback {
		return storage.IPv4Loopback
	}
	// 尝试解析IP地址，如果是IPv6映射的IPv4地址，转换为IPv4
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		// 如果是IPv6映射的IPv4地址（::ffff:127.0.0.1），转换为IPv4
		if ipv4 := parsedIP.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ip
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	if size < storage.BytesPerKB {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(storage.BytesPerKB), 0
	for n := size / storage.BytesPerKB; n >= storage.BytesPerKB; n /= storage.BytesPerKB {
		div *= storage.BytesPerKB
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
