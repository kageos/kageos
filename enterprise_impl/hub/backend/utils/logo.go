package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
)

// GenerateDefaultLogo 生成默认 Logo URL（基于应用名称）
// 使用 DiceBear API 生成随机头像，基于应用名称的哈希值确保相同名称生成相同头像
func GenerateDefaultLogo(appName string) string {
	// 使用 MD5 哈希应用名称，确保相同名称生成相同的头像
	hash := md5.Sum([]byte(appName))
	seed := fmt.Sprintf("%x", hash)[:8] // 取前8位作为种子
	
	// 使用 DiceBear API 生成头像（使用 avataaars 风格）
	// 更多风格可选：avataaars, bottts, identicon, initials, pixel-art 等
	return fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=b6e3f4,c0aede,d1d4f9,ffd5dc,ffdfbf", seed)
}

// HashString 计算字符串的 SHA256 哈希值
func HashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash)
}





