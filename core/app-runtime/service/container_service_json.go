package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

func decodePodmanJSONObjectList(output []byte) ([]map[string]json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var objects []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &objects); err != nil {
			return nil, err
		}
		return objects, nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &object); err != nil {
			return nil, err
		}
		return []map[string]json.RawMessage{object}, nil
	}

	return nil, fmt.Errorf("unexpected JSON output: %q", trimmed)
}

func podmanJSONField(object map[string]json.RawMessage, names ...string) (json.RawMessage, bool) {
	for _, name := range names {
		if raw, ok := object[name]; ok {
			return raw, true
		}
	}
	for key, raw := range object {
		for _, name := range names {
			if strings.EqualFold(key, name) {
				return raw, true
			}
		}
	}
	return nil, false
}

func podmanJSONString(object map[string]json.RawMessage, names ...string) string {
	raw, ok := podmanJSONField(object, names...)
	if !ok {
		return ""
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	return ""
}

func podmanJSONBool(object map[string]json.RawMessage, names ...string) (bool, bool) {
	raw, ok := podmanJSONField(object, names...)
	if !ok {
		return false, false
	}

	var boolValue bool
	if err := json.Unmarshal(raw, &boolValue); err == nil {
		return boolValue, true
	}

	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		switch strings.ToLower(strings.TrimSpace(stringValue)) {
		case "true", "running":
			return true, true
		case "false", "exited", "stopped", "created":
			return false, true
		}
	}

	return false, false
}

func podmanJSONNames(object map[string]json.RawMessage) []string {
	raw, ok := podmanJSONField(object, "Names", "names")
	if !ok {
		return nil
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		names := make([]string, 0, len(values))
		for _, name := range values {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
		return names
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return splitContainerNames(value)
	}

	return nil
}

func parsePodmanContainerListJSON(output []byte) ([]ContainerInfo, error) {
	objects, err := decodePodmanJSONObjectList(output)
	if err != nil {
		return nil, err
	}

	containers := make([]ContainerInfo, 0, len(objects))
	for _, object := range objects {
		state := strings.ToLower(podmanJSONString(object, "State", "state", "Status", "status"))
		exited := true
		if state != "" {
			exited = state != "running"
		} else if value, ok := podmanJSONBool(object, "Exited", "exited"); ok {
			exited = value
		}

		containers = append(containers, ContainerInfo{
			ID:     podmanJSONString(object, "ID", "Id", "id"),
			Names:  podmanJSONNames(object),
			State:  state,
			Exited: exited,
		})
	}

	return containers, nil
}

func parsePodmanImageListJSON(output []byte) ([]ImageInfo, error) {
	objects, err := decodePodmanJSONObjectList(output)
	if err != nil {
		return nil, err
	}

	images := make([]ImageInfo, 0, len(objects))
	for _, object := range objects {
		repository := podmanJSONString(object, "Repository", "repository")
		tag := podmanJSONString(object, "Tag", "tag")
		if repository == "" && tag == "" {
			repository, tag = splitImageRepositoryTag(firstPodmanJSONName(object))
		}

		images = append(images, ImageInfo{
			ID:         podmanJSONString(object, "ID", "Id", "id"),
			Repository: repository,
			Tag:        tag,
		})
	}

	return images, nil
}

func parsePodmanInspectRunningJSON(output []byte) (bool, error) {
	objects, err := decodePodmanJSONObjectList(output)
	if err != nil {
		return false, err
	}
	if len(objects) == 0 {
		return false, nil
	}

	stateRaw, ok := podmanJSONField(objects[0], "State", "state")
	if !ok {
		return false, nil
	}

	var stateObject map[string]json.RawMessage
	if err := json.Unmarshal(stateRaw, &stateObject); err != nil {
		return false, err
	}
	running, _ := podmanJSONBool(stateObject, "Running", "running")
	return running, nil
}

// 容器管理方法

// ListContainers 列出所有容器（包括已停止的）
func splitContainerNames(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	rawNames := strings.Split(value, ",")
	names := make([]string, 0, len(rawNames))
	for _, name := range rawNames {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func firstPodmanJSONName(object map[string]json.RawMessage) string {
	names := podmanJSONNames(object)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func splitImageRepositoryTag(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}

	lastSlash := strings.LastIndex(name, "/")
	lastColon := strings.LastIndex(name, ":")
	if lastColon > lastSlash {
		return name[:lastColon], name[lastColon+1:]
	}
	return name, ""
}
