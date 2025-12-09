package gofmt

import "golang.org/x/tools/imports"

func FixGoImport(filename string, src []byte) (out string, err error) {
	// 对于大多数项目，使用这个配置
	opt := &imports.Options{
		Comments:  true, // 保留注释很重要
		TabIndent: true, // Go 标准使用 tab
		TabWidth:  8,    // 标准宽度
	}
	process, err := imports.Process(filename, src, opt)
	return string(process), err
}
