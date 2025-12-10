package gofmt

import "golang.org/x/tools/imports"

// FixGoImport filename 是即将写入的文件路径，code是源代码
func FixGoImport(filename string, code []byte) (out string, err error) {
	// 对于大多数项目，使用这个配置
	opt := &imports.Options{
		Comments:  true, // 保留注释很重要
		TabIndent: true, // Go 标准使用 tab
		TabWidth:  8,    // 标准宽度
	}
	process, err := imports.Process(filename, code, opt)
	return string(process), err
}
