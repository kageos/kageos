package jsonx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
)

// DeepEqual 通过JSON序列化/反序列化比较两个interface{}是否相等
// 这个方法可以比较不同类型但内容相同的值，比如：
// - map[string]interface{}{"prefix": ""} 和 &struct{Prefix string}{Prefix: ""}
// - 不同字段顺序的JSON等
func DeepEqual(a1, a2 interface{}) bool {
	// 处理nil值
	if a1 == nil && a2 == nil {
		return true
	}
	if a1 == nil || a2 == nil {
		return false
	}

	// 将两个值都序列化为JSON
	j1, err1 := json.Marshal(a1)
	j2, err2 := json.Marshal(a2)

	if err1 != nil || err2 != nil {
		// 如果序列化失败，回退到标准的reflect.DeepEqual
		return reflect.DeepEqual(a1, a2)
	}

	// 如果JSON字符串直接相等，那肯定相等
	if string(j1) == string(j2) {
		return true
	}

	// 将JSON反序列化为统一的interface{}（通常是map[string]interface{}）
	var parsed1, parsed2 interface{}
	if err := json.Unmarshal(j1, &parsed1); err != nil {
		return false
	}
	if err := json.Unmarshal(j2, &parsed2); err != nil {
		return false
	}

	// 使用reflect.DeepEqual比较反序列化后的值
	return reflect.DeepEqual(parsed1, parsed2)
}

// NormalizeToMap 将interface{}通过JSON序列化/反序列化规范化为map[string]interface{}
// 这样可以消除不同数据类型之间的差异
func NormalizeToMap(value interface{}) (map[string]interface{}, error) {
	if value == nil {
		return nil, nil
	}

	// 序列化为JSON
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	// 反序列化为map
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// EqualJSON 比较两个值的JSON表示是否相等
func EqualJSON(a1, a2 interface{}) (bool, error) {
	j1, err1 := json.Marshal(a1)
	if err1 != nil {
		return false, err1
	}

	j2, err2 := json.Marshal(a2)
	if err2 != nil {
		return false, err2
	}

	return string(j1) == string(j2), nil
}

// Canonicalize 规范化值为标准的JSON结构
func Canonicalize(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	// 序列化为JSON
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	// 反序列化为interface{}
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func Convert(src, dest interface{}) error {
	jsonBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, dest)
	if err != nil {
		return err
	}
	return nil
}

func SaveFile(file string, data interface{}) error {
	os.MkdirAll(filepath.Dir(file), 0777)
	create, err := os.Create(file)
	if err != nil {
		return err
	}
	marshal, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = create.Write(marshal)
	if err != nil {
		return err
	}
	return nil
}
