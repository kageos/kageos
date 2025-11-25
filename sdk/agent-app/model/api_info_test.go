package model

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"testing"
)

func TestApiInfoBindingMethods(t *testing.T) {
	// 创建测试用的ApiInfo对象
	apiInfo := &app.ApiInfo{
		Code:         "createUser",
		Name:         "创建用户",
		Desc:         "创建新用户",
		Router:       "/api/user/create",
		Method:       "POST",
		User:         "testuser",
		App:          "testapp",
		FullCodePath: "/testuser/testapp/api/user/create",
	}

	// 测试GetParentFullNamePath
	expectedParentPath := "/testuser/testapp/api/user"
	if parentPath := apiInfo.GetParentFullCodePath(); parentPath != expectedParentPath {
		t.Errorf("Expected GetParentFullCodePath to be %s, got %s", expectedParentPath, parentPath)
	}

	// 测试GetAppPrefix
	expectedAppPrefix := "/testuser/testapp"
	if appPrefix := apiInfo.GetAppPrefix(); appPrefix != expectedAppPrefix {
		t.Errorf("Expected GetAppPrefix to be %s, got %s", expectedAppPrefix, appPrefix)
	}

	// 测试GetRelativePath
	expectedRelativePath := "/api/user/create"
	if relativePath := apiInfo.GetRelativePath(); relativePath != expectedRelativePath {
		t.Errorf("Expected GetRelativePath to be %s, got %s", expectedRelativePath, relativePath)
	}

	// 测试GetFunctionName
	expectedFunctionName := "createUser"
	if functionName := apiInfo.GetFunctionName(); functionName != expectedFunctionName {
		t.Errorf("Expected GetFunctionName to be %s, got %s", expectedFunctionName, functionName)
	}

	// 测试GetPackagePath
	expectedPackagePath := "/testuser/testapp/api/user"
	if packagePath := apiInfo.GetPackagePath(); packagePath != expectedPackagePath {
		t.Errorf("Expected GetPackagePath to be %s, got %s", expectedPackagePath, packagePath)
	}

	// 测试GetPackageChain
	expectedPackageChain := []string{"api", "user"}
	packageChain := apiInfo.GetPackageChain()
	if len(packageChain) != len(expectedPackageChain) {
		t.Errorf("Expected GetPackageChain to have %d elements, got %d", len(expectedPackageChain), len(packageChain))
	} else {
		for i, expected := range expectedPackageChain {
			if packageChain[i] != expected {
				t.Errorf("Expected packageChain[%d] to be %s, got %s", i, expected, packageChain[i])
			}
		}
	}
}

func TestApiInfoEdgeCases(t *testing.T) {
	// 测试空路径
	apiInfo1 := &app.ApiInfo{
		FullCodePath: "",
		User:         "testuser",
		App:          "testapp",
	}

	if parentPath := apiInfo1.GetParentFullCodePath(); parentPath != "" {
		t.Errorf("Expected empty parent path for empty FullCodePath, got %s", parentPath)
	}

	if relativePath := apiInfo1.GetRelativePath(); relativePath != "" {
		t.Errorf("Expected empty relative path for empty FullCodePath, got %s", relativePath)
	}

	if functionName := apiInfo1.GetFunctionName(); functionName != "" {
		t.Errorf("Expected empty function name for empty FullCodePath, got %s", functionName)
	}

	// 测试只有应用级别的路径
	apiInfo2 := &app.ApiInfo{
		FullCodePath: "/testuser/testapp",
		User:         "testuser",
		App:          "testapp",
	}

	if parentPath := apiInfo2.GetParentFullCodePath(); parentPath != "/testuser" {
		t.Errorf("Expected parent path to be '/testuser' for app-level path, got %s", parentPath)
	}

	// 测试GetFunctionName（code字段总是存在）
	if functionName := apiInfo2.GetFunctionName(); functionName != "" {
		t.Errorf("Expected empty function name for ApiInfo without code, got %s", functionName)
	}

	// 测试一级路径
	apiInfo3 := &app.ApiInfo{
		FullCodePath: "/testuser/testapp/api",
		User:         "testuser",
		App:          "testapp",
	}

	expectedParentPath := "/testuser/testapp"
	if parentPath := apiInfo3.GetParentFullCodePath(); parentPath != expectedParentPath {
		t.Errorf("Expected parent path to be %s, got %s", expectedParentPath, parentPath)
	}

	// 测试GetFunctionName（code字段总是存在）
	if functionName := apiInfo3.GetFunctionName(); functionName != "" {
		t.Errorf("Expected empty function name for ApiInfo without code, got %s", functionName)
	}

	// 测试GetPackageChain对于一级路径应该返回空
	packageChain := apiInfo3.GetPackageChain()
	if len(packageChain) != 0 {
		t.Errorf("Expected empty package chain for single-level path, got %v", packageChain)
	}
}

func TestApiInfoBuildFullNamePath(t *testing.T) {
	apiInfo := &app.ApiInfo{
		User:   "testuser",
		App:    "testapp",
		Router: "/api/user/create",
	}

	expectedPath := "/testuser/testapp/api/user/create"
	if path := apiInfo.BuildFullCodePath(); path != expectedPath {
		t.Errorf("Expected BuildFullCodePath to be %s, got %s", expectedPath, path)
	}

	// 测试Router有斜杠的情况
	apiInfo.Router = "api/user/create"
	expectedPath = "/testuser/testapp/api/user/create"
	if path := apiInfo.BuildFullCodePath(); path != expectedPath {
		t.Errorf("Expected BuildFullCodePath to be %s, got %s", expectedPath, path)
	}

	// 测试空Router
	apiInfo.Router = ""
	expectedPath = "/testuser/testapp"
	if path := apiInfo.BuildFullCodePath(); path != expectedPath {
		t.Errorf("Expected BuildFullCodePath to be %s, got %s", expectedPath, path)
	}
}
