# 工作台路径映射

用户常说“帮我在某个工作台目录下面创建 xxx”。这类路径不是磁盘路径，而是 kageos full_code_path。

## 解析规则

```text
/<user>/<app>/<package...>
```

抽象结构：

```text
/<user>/<app>/<package>
│       │      └─ package path
│       └─ app
└─ user
```

对应本地位置：

```text
<kageos_repo_root>/namespace/<user>/<app>/
├── go.mod
├── code/
│   ├── api/
│   │   └── <package>/
│   │       ├── init_.go
│   │       └── *.go
│   └── cmd/app/main.go
```

嵌套目录：

```text
/<user>/<app>/<package>/<child>
=> namespace/<user>/<app>/code/api/<package>/<child>
=> RouterGroup: "/<package>/<child>"
=> import path: github.com/kageos/kageos/namespace/<user>/<app>/code/api/<package>/<child>
```

应用根目录：

```text
/<user>/<app>
=> namespace/<user>/<app>
=> 新增业务包时根据需求选择 code/api/<package>
```

## 写代码位置硬规则

只把工作空间业务代码写在应用模块的 `code/api/<package...>/` 下。

```text
full_code_path: /system/promotion
app_root:       <kageos_repo_root>/namespace/system/promotion
业务代码:       <kageos_repo_root>/namespace/system/promotion/code/api/<package>/*.go
入口文件:       <kageos_repo_root>/namespace/system/promotion/code/cmd/app/main.go
```

如果用户只给到应用根路径，例如 `/system/promotion`：

1. 先检查 `namespace/system/promotion/code/api/` 是否已有符合需求的 package。
2. 有合适 package 就改这个 package，例如 `code/api/distribution/`。
3. 没有合适 package 才新建 `code/api/<业务目录名>/`。
4. 不要把 `init_.go`、`tables.go`、`forms.go` 写到 `namespace/system/promotion/` 根目录。

如果用户给到 package 路径，例如 `/system/promotion/distribution`：

```text
写代码到: namespace/system/promotion/code/api/distribution/
RouterGroup: "/distribution"
blank import: _ "github.com/kageos/kageos/namespace/system/promotion/code/api/distribution"
```

如果用户给到嵌套 package，例如 `/system/promotion/douyin/publish`：

```text
写代码到: namespace/system/promotion/code/api/douyin/publish/
RouterGroup: "/douyin/publish"
blank import: _ "github.com/kageos/kageos/namespace/system/promotion/code/api/douyin/publish"
```

允许编辑的位置：

- `namespace/<user>/<app>/code/api/<package...>/*.go`
- `namespace/<user>/<app>/code/cmd/app/main.go`
- `namespace/<user>/<app>/go.mod` / `go.sum`

禁止默认编辑的位置：

- 真实磁盘 `/system/...`
- `namespace/<user>/<app>/` 根目录下的随意 Go 文件
- kageos 主仓库根目录
- `core/...`
- `kageos-sdk/...`
- `.codex/skills/...`

只有当用户明确要求修改平台核心、SDK 或 skill 本身时，才编辑这些禁止默认编辑的位置。

## 定位仓库

优先从当前目录向上找 kageos 主仓库特征：

- `go.mod` 里有 `module github.com/kageos/kageos`
- 存在 `namespace/`
- 存在 `core/app-server` 和 `core/app-runtime`

如果当前 workspace 是外层集合目录，kageos 主仓库通常在 `<cwd>/kageos`。

## 操作前检查

在改文件前确认：

- `namespace/<user>/<app>/go.mod` 存在。
- `namespace/<user>/<app>/code/cmd/app/main.go` 存在。
- 目标 package 目录是否已有 `init_.go`。
- 目标 package 是否已被 `main.go` blank import。

可以运行：

```bash
~/.codex/skills/kageos-developer/scripts/inspect_workspace_app.sh /<user>/<app>/<package> <kageos_repo_root>
```
