# 将公钥编译到可执行文件中

## 概述

为了简化部署，可以将 RSA 公钥直接编译到可执行文件中。这样：
- ✅ **公钥公开**：公钥在代码中，任何人都可以看到
- ✅ **私钥保密**：只有拥有私钥的人才能生成有效的 License
- ✅ **激活简单**：激活企业版只需要用私钥签名的 License 文件

## 实现方式

### 方式一：使用 build tag（推荐）

#### 1. 创建公钥文件

将公钥内容保存为字符串：

```go
// pkg/license/embedded_public_key.go
// +build embed_public_key

package license

func init() {
	embeddedPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
（这里是公钥内容）
-----END PUBLIC KEY-----`
}
```

#### 2. 编译时启用

```bash
go build -tags embed_public_key -o app-server ./core/app-server/cmd/app
```

### 方式二：使用 embed（Go 1.16+）

#### 1. 创建公钥文件

将公钥保存为 `pkg/license/public_key.pem`

#### 2. 使用 embed 嵌入

```go
// pkg/license/embedded_public_key.go
package license

import _ "embed"

//go:embed public_key.pem
var embeddedPublicKeyData string

func init() {
	embeddedPublicKey = embeddedPublicKeyData
}
```

#### 3. 编译

```bash
go build -o app-server ./core/app-server/cmd/app
```

### 方式三：直接设置（开发/测试）

```go
// pkg/license/embedded_public_key.go
package license

func init() {
	// 开发/测试环境可以直接设置
	embeddedPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----`
}
```

## 工作流程

### 1. 生成密钥对

```bash
# 生成私钥
openssl genrsa -out private_key.pem 2048

# 生成公钥
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

### 2. 将公钥编译到程序中

使用上述方式之一将公钥嵌入到程序中。

### 3. 使用私钥签名 License

```bash
# 使用私钥签名 License 文件
# （需要 License 签名工具）
```

### 4. 分发 License 文件

将签名的 License 文件分发给客户，客户只需要：
1. 将 License 文件放到指定位置
2. 启动服务，自动激活企业版

## 安全说明

### ✅ 安全的设计

1. **公钥可以公开**：公钥用于验证签名，公开不影响安全
2. **私钥必须保密**：只有拥有私钥的人才能生成有效的 License
3. **签名验证**：程序使用公钥验证 License 签名，确保 License 未被篡改

### ⚠️ 注意事项

1. **私钥安全**：私钥必须严格保密，不能泄露
2. **密钥轮换**：如果需要更换密钥，需要重新编译程序并分发
3. **向后兼容**：如果程序中没有嵌入公钥，会从文件加载（向后兼容）

## 示例

### 完整示例（使用 embed）

```go
// pkg/license/embedded_public_key.go
package license

import _ "embed"

//go:embed public_key.pem
var embeddedPublicKeyData string

func init() {
	if embeddedPublicKeyData != "" {
		embeddedPublicKey = embeddedPublicKeyData
	}
}
```

### 编译

```bash
# 确保 public_key.pem 文件存在
# 编译时会自动嵌入
go build -o app-server ./core/app-server/cmd/app
```

## 优势

1. **简化部署**：不需要单独分发公钥文件
2. **防止篡改**：公钥在程序中，无法被替换
3. **激活简单**：客户只需要 License 文件即可激活

## 参考

- [License 系统设计](./README.md)
- [License 使用说明](./USAGE.md)
