
字段详情
```json
{
                "callbacks": null,
                "children": null,
                "code": "attachment",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "struct"
                },
                "desc": "",
                "field_name": "Attachment",
                "name": "附件",
                "search": "",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "accept": "",
                        "max_count": 5,
                        "max_size": ""
                    },
                    "type": "files"
                }
            }
```

字段参数
```go
type FilesParams struct {
	// Accept 文件类型限制，支持多种格式（逗号分隔）：
	// 1. 扩展名：.pdf,.doc,.docx,.jpg,.png
	// 2. MIME类型：application/pdf,image/jpeg
	// 3. MIME通配符：image/*,video/*,audio/*
	// 4. 混合使用：.pdf,image/*,video/*,application/zip
	// 示例：accept:.pdf,.doc,.docx,image/*,video/*
	// 为空或者*则不限制类型
	Accept string `json:"accept"`

	// MaxSize 单个文件最大大小，支持单位：B, KB, MB, GB
	// 示例：max_size:10MB, max_size:1024KB, max_size:1GB
	// 为空则不限制
	MaxSize string `json:"max_size"`

	// MaxCount 最大上传文件数量，默认为 5
	// 示例：max_count:10
	MaxCount int `json:"max_count"`
}

```

对应的后端的go的结构体,上传后需要传递这种字段，输出文件也会输出这种结构的
```go
type Files struct {
	Files    []*File                `json:"files"`
	Remark   string                 `json:"remark"`
	Metadata map[string]interface{} `json:"metadata"`
}
type File struct {
	Name        string `json:"name"`
	SourceName  string `json:"source_name"` //源文件名称
	Storage     string `json:"storage"`     //minio/qiniu/xxxxx 存储引擎
	Description string `json:"description"`
	Hash        string `json:"hash"`
	Size        int64  `json:"size"`
	UploadTs    int64  `json:"upload_ts"`
	LocalPath   string `json:"local_path"`
	IsUploaded  bool   `json:"is_uploaded"`          //是否已经上传到云端
	Url         string `json:"url"`                  // 上传后的地址
	Downloaded  bool   `json:"downloaded,omitempty"` //是否已经下载到本地
}
```


