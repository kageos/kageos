package gitx

import (
	"fmt"
	"os"
)

// existsFile 检查文件是否存在
func existsFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("检查文件是否存在失败: %v", err)
}
