package metadata

// Metadata 代码元数据结构体
// 使用 json tag 作为 metadata 的 key
type Metadata struct {
	File          string   `json:"file"`
	DirectoryName string   `json:"directory_name"`
	DirectoryCode string   `json:"directory_code"`
	DirectoryDesc string   `json:"directory_desc"`
	Tags          []string `json:"tags,omitempty"`
	Version       string   `json:"version,omitempty"`
}
