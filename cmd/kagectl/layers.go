package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type deploymentLayerID string

const (
	layerControl  deploymentLayerID = "L0"
	layerInfra    deploymentLayerID = "L1"
	layerEdge     deploymentLayerID = "L2"
	layerPlatform deploymentLayerID = "L3"
	layerRuntime  deploymentLayerID = "L4"
	layerApps     deploymentLayerID = "L5"
)

type deploymentLayer struct {
	ID             deploymentLayerID
	Name           string
	Responsibility string
}

type deploymentComponent struct {
	Layer deploymentLayerID
	Name  string
	Role  string
}

type layerCheck struct {
	Layer  deploymentLayerID
	Name   string
	Target string
	Fn     func() error
}

type layerComposeServices struct {
	Layer    deploymentLayerID
	Services []string
	Note     string
}

type deploymentReport struct {
	Layers []deploymentLayerReport `json:"layers"`
}

type deploymentLayerReport struct {
	ID              deploymentLayerID          `json:"id"`
	Name            string                     `json:"name"`
	Responsibility  string                     `json:"responsibility"`
	Components      []deploymentComponentEntry `json:"components,omitempty"`
	ComposeServices []string                   `json:"compose_services,omitempty"`
	Note            string                     `json:"note,omitempty"`
}

type deploymentComponentEntry struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type checkReport struct {
	Name     string             `json:"name"`
	OK       bool               `json:"ok"`
	Failures int                `json:"failures"`
	Checks   []layerCheckResult `json:"checks"`
}

type layerCheckResult struct {
	LayerID   deploymentLayerID `json:"layer_id"`
	LayerName string            `json:"layer_name"`
	Name      string            `json:"name"`
	Target    string            `json:"target,omitempty"`
	OK        bool              `json:"ok"`
	Error     string            `json:"error,omitempty"`
}

type statusReport struct {
	Deployment        deploymentReport `json:"deployment"`
	ComposeConfigPath string           `json:"compose_config_path"`
	ComposePS         string           `json:"compose_ps,omitempty"`
}

func deploymentLayers() []deploymentLayer {
	return []deploymentLayer{
		{ID: layerControl, Name: "部署控制层", Responsibility: "kagectl、配置渲染、目录和镜像编排"},
		{ID: layerInfra, Name: "基础设施层", Responsibility: "MySQL、NATS、MinIO、持久化数据目录"},
		{ID: layerEdge, Name: "入口接入层", Responsibility: "Nginx、HTTP/HTTPS、静态前端、API 反代、维护页"},
		{ID: layerPlatform, Name: "平台服务层", Responsibility: "平台业务服务"},
		{ID: layerRuntime, Name: "运行时管理层", Responsibility: "app-runtime、Podman API、app-base 镜像、namespace"},
		{ID: layerApps, Name: "用户应用层", Responsibility: "用户 App 容器、SDK、用户业务代码"},
	}
}

func deploymentComponents(rt RuntimeConfig) []deploymentComponent {
	components := []deploymentComponent{
		{Layer: layerControl, Name: "kagectl", Role: "生成 .generated、调用 Compose、执行 up/status/verify"},
		{Layer: layerControl, Name: "compose", Role: "外层容器执行引擎"},
		{Layer: layerInfra, Name: infraComponentName("mysql", rt.MySQL.Mode), Role: "平台关系型数据"},
		{Layer: layerInfra, Name: infraComponentName("nats", rt.NATS.Mode), Role: "平台和用户 App 消息总线"},
		{Layer: layerInfra, Name: infraComponentName("minio", rt.MinIO.Mode), Role: "对象存储和文件上传下载"},
		{Layer: layerInfra, Name: rt.Storage.Root, Role: "宿主机持久化根目录"},
		{Layer: layerEdge, Name: "nginx", Role: "容器内入口，host 网络监听 80/443"},
		{Layer: layerPlatform, Name: "core-server", Role: "统一承载 gateway/app/agent/hr/storage"},
		{Layer: layerRuntime, Name: "app-runtime", Role: "用户 App 生命周期控制"},
		{Layer: layerRuntime, Name: "podman-api", Role: "main 容器内 Podman socket"},
		{Layer: layerRuntime, Name: rt.Images.AppBase, Role: "用户 App 运行时基础镜像"},
		{Layer: layerApps, Name: "user-app containers", Role: "由 app-runtime 动态创建"},
		{Layer: layerApps, Name: "SDK endpoints", Role: fmt.Sprintf("nats=%s gateway=%s minio=%s", redactURLCredentials(rt.SDKNATSURL), rt.SDKGatewayURL, rt.SDKMinIOEndpoint)},
	}
	return components
}

func infraComponentName(name, mode string) string {
	if mode == "" {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, mode)
}

func layerTitle(id deploymentLayerID) string {
	for _, layer := range deploymentLayers() {
		if layer.ID == id {
			return fmt.Sprintf("%s %s", layer.ID, layer.Name)
		}
	}
	return string(id)
}

func printDeploymentLayers(rt RuntimeConfig) {
	components := deploymentComponents(rt)
	for _, layer := range deploymentLayers() {
		fmt.Printf("%s %s\n", layer.ID, layer.Name)
		fmt.Printf("  职责: %s\n", layer.Responsibility)
		for _, component := range components {
			if component.Layer != layer.ID {
				continue
			}
			fmt.Printf("  - %s: %s\n", component.Name, component.Role)
		}
	}
}

func buildDeploymentReport(rt RuntimeConfig) deploymentReport {
	componentsByLayer := map[deploymentLayerID][]deploymentComponentEntry{}
	for _, component := range deploymentComponents(rt) {
		componentsByLayer[component.Layer] = append(componentsByLayer[component.Layer], deploymentComponentEntry{
			Name: component.Name,
			Role: component.Role,
		})
	}

	servicesByLayer := map[deploymentLayerID]layerComposeServices{}
	for _, group := range composeServicesByLayer(rt) {
		servicesByLayer[group.Layer] = group
	}

	report := deploymentReport{Layers: make([]deploymentLayerReport, 0, len(deploymentLayers()))}
	for _, layer := range deploymentLayers() {
		entry := deploymentLayerReport{
			ID:             layer.ID,
			Name:           layer.Name,
			Responsibility: layer.Responsibility,
			Components:     componentsByLayer[layer.ID],
		}
		if group, ok := servicesByLayer[layer.ID]; ok {
			entry.ComposeServices = append([]string(nil), group.Services...)
			entry.Note = group.Note
		}
		report.Layers = append(report.Layers, entry)
	}
	return report
}

func printComposeServiceOwnership(rt RuntimeConfig) {
	for _, group := range composeServicesByLayer(rt) {
		title := layerTitle(group.Layer)
		services := "none"
		if len(group.Services) > 0 {
			services = strings.Join(group.Services, ", ")
		}
		if group.Note == "" {
			fmt.Printf("%s: %s\n", title, services)
		} else {
			fmt.Printf("%s: %s (%s)\n", title, services, group.Note)
		}
	}
}

func composeServicesByLayer(rt RuntimeConfig) []layerComposeServices {
	infraServices := make([]string, 0, 3)
	infraNotes := make([]string, 0, 3)
	if rt.IncludeMySQL {
		infraServices = append(infraServices, "mysql")
	} else {
		infraNotes = append(infraNotes, "mysql external")
	}
	if rt.IncludeNATS {
		infraServices = append(infraServices, "nats")
	} else {
		infraNotes = append(infraNotes, "nats external")
	}
	if rt.IncludeMinIO {
		infraServices = append(infraServices, "minio")
	} else {
		infraNotes = append(infraNotes, "minio external")
	}

	return []layerComposeServices{
		{Layer: layerControl, Note: "kagectl runs on the host"},
		{Layer: layerInfra, Services: infraServices, Note: strings.Join(infraNotes, ", ")},
		{Layer: layerEdge, Services: []string{"main"}, Note: "nginx runs inside main"},
		{Layer: layerPlatform, Services: []string{"main"}, Note: "core-server runs inside main"},
		{Layer: layerRuntime, Services: []string{"main"}, Note: "app-runtime and Podman API run inside main"},
		{Layer: layerApps, Note: "user App containers are managed by app-runtime, not Compose"},
	}
}

func composeServicesForLayer(rt RuntimeConfig, layer deploymentLayerID) []string {
	for _, group := range composeServicesByLayer(rt) {
		if group.Layer == layer {
			return append([]string(nil), group.Services...)
		}
	}
	return nil
}

func parseDeploymentLayer(value string) (deploymentLayerID, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "l0", "control", "deploy", "deployment", "部署控制层":
		return layerControl, true
	case "l1", "infra", "infrastructure", "基础设施层":
		return layerInfra, true
	case "l2", "edge", "entry", "入口接入层":
		return layerEdge, true
	case "l3", "platform", "平台服务层":
		return layerPlatform, true
	case "l4", "runtime", "运行时管理层":
		return layerRuntime, true
	case "l5", "apps", "app", "user-apps", "用户应用层":
		return layerApps, true
	default:
		return "", false
	}
}

func runLayerChecks(name string, checks []layerCheck) error {
	report := runLayerChecksReport(name, checks)
	printLayerCheckReport(report)
	if !report.OK {
		return fmt.Errorf("%s failed with %d issue(s)", name, report.Failures)
	}
	fmt.Printf("\n%s passed\n", name)
	return nil
}

func printLayerCheckReport(report checkReport) {
	var current deploymentLayerID
	for _, result := range report.Checks {
		if result.LayerID != current {
			current = result.LayerID
			fmt.Printf("\n%s\n", layerTitle(current))
		}
		if !result.OK {
			if result.Target == "" {
				fmt.Printf("  [FAIL] %s: %s\n", result.Name, result.Error)
			} else {
				fmt.Printf("  [FAIL] %s (%s): %s\n", result.Name, result.Target, result.Error)
			}
		} else if result.Target == "" {
			fmt.Printf("  [ OK ] %s\n", result.Name)
		} else {
			fmt.Printf("  [ OK ] %s (%s)\n", result.Name, result.Target)
		}
	}
}

func runLayerChecksReport(name string, checks []layerCheck) checkReport {
	report := checkReport{
		Name:   name,
		OK:     true,
		Checks: make([]layerCheckResult, 0, len(checks)),
	}
	for _, check := range checks {
		result := layerCheckResult{
			LayerID:   check.Layer,
			LayerName: layerName(check.Layer),
			Name:      check.Name,
			Target:    check.Target,
			OK:        true,
		}
		if err := check.Fn(); err != nil {
			result.OK = false
			result.Error = err.Error()
			report.OK = false
			report.Failures++
		}
		report.Checks = append(report.Checks, result)
	}
	return report
}

func layerName(id deploymentLayerID) string {
	for _, layer := range deploymentLayers() {
		if layer.ID == id {
			return layer.Name
		}
	}
	return string(id)
}

func tcpTarget(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func redactURLCredentials(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return raw
	}
	username := parsed.User.Username()
	if username == "" {
		parsed.User = url.User("redacted")
	} else {
		parsed.User = url.UserPassword(username, "redacted")
	}
	return parsed.String()
}
