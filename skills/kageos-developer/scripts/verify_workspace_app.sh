#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "用法: verify_workspace_app.sh <full_code_path> [kageos_repo_root]" >&2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
inspect_output="$("$script_dir/inspect_workspace_app.sh" "$@")"
echo "$inspect_output"

app_root="$(printf '%s\n' "$inspect_output" | awk -F': ' '/^app_root:/ {print $2}')"
package_dir="$(printf '%s\n' "$inspect_output" | awk -F': ' '/^package_dir:/ {print $2}')"

if [[ ! -d "$app_root" ]]; then
  echo "错误: app_root 不存在: $app_root" >&2
  exit 1
fi

cd "$app_root"

echo
echo "== gofmt =="
if [[ -d "$package_dir" ]]; then
  while IFS= read -r -d '' go_file; do
    gofmt -w "$go_file"
  done < <(find "$package_dir" -name '*.go' -print0)
fi
if [[ -f code/cmd/app/main.go ]]; then
  gofmt -w code/cmd/app/main.go
fi

echo
echo "== go test ./... =="
if ! go test ./...; then
  echo
  echo "go test 失败。若错误是 missing go.sum entry，先在模块根目录运行 go mod tidy，再重新验证。" >&2
  exit 1
fi

echo
echo "== go build ./code/cmd/app =="
if ! go build ./code/cmd/app; then
  echo
  echo "go build 失败。请先修复编译错误，再重新验证。" >&2
  exit 1
fi

if [[ -f app ]]; then
  rm app
  echo "已删除验证生成的 ./app 二进制"
fi

echo
echo "kageos 工作空间应用验证通过: $app_root"
