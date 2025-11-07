package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Files struct {
	Files      []*File                `json:"files"`
	UploadUser string                 `json:"upload_user"`
	Remark     string                 `json:"remark"`
	Metadata   map[string]interface{} `json:"metadata"`
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
	Url         string `json:"url"`                  // ✨ 外部访问地址（前端下载使用）
	ServerUrl   string `json:"server_url"`           // ✨ 内部访问地址（服务端下载使用）
	Downloaded  bool   `json:"downloaded,omitempty"` //是否已经下载到本地
}

func (f *File) GetLocalPath() string {
	return f.LocalPath
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
func (fc *Files) Scan(value interface{}) error {
	if value == nil {
		*fc = Files{}
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("cannot scan %T into Files", value)
	}

	return json.Unmarshal(data, fc)
}

// Value 实现 driver.Valuer 接口，用于存储到数据库
func (fc Files) Value() (driver.Value, error) {
	if len(fc.Files) == 0 {
		return nil, nil
	}
	return json.Marshal(fc)
}
