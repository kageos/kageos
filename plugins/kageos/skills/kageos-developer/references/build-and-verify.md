# 构建与验证

## 本地模块验证

进入工作空间应用模块：

```bash
cd <kageos_repo>/namespace/<user>/<app>
```

执行：

```bash
gofmt -w code/api/<package>/*.go code/cmd/app/main.go
go test ./...
go build ./code/cmd/app
```

`go build ./code/cmd/app` 可能在当前目录生成本地二进制 `app`。如果只是验证，可以删除它。

## 脚本验证

```bash
~/.codex/skills/kageos-developer/scripts/verify_workspace_app.sh /<user>/<app>/<package> <kageos_repo_root>
```

脚本会：

- 解析 full_code_path
- 定位 `namespace/<user>/<app>`
- 跑 `gofmt -w`
- 跑 `go test ./...`
- 跑 `go build ./code/cmd/app`
- 删除验证生成的 `app` 二进制

## 平台生效

本地验证只说明代码能编译。要让平台目录和函数刷新，需要在 kageos 工作台或 agent 工具里执行真正的 build/update，例如 `build_workspace`。

只执行写文件、search replace、write-only update，不会让平台 service_tree 自动刷新。

## 排错顺序

1. 先看 Go 编译错误，定位文件、结构体 tag、缺 import。
2. 再看 SDK schema 生成错误，检查 widget、Response、TemplateType。
3. 再看 runtime update callback，确认新版本是否启动。
4. 最后看 app-server warnings，确认目录对账或函数元数据同步是否失败。
