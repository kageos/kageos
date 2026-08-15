#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "用法: inspect_workspace_app.sh <full_code_path> [kageos_repo_root]" >&2
  echo "示例: inspect_workspace_app.sh /<user>/<app>/<package> <kageos_repo_root>" >&2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

full_path="$1"
repo_root="${2:-}"

trimmed="${full_path#/}"
trimmed="${trimmed%/}"
IFS='/' read -r -a parts <<< "$trimmed"

if [[ ${#parts[@]} -lt 2 ]]; then
  echo "错误: full_code_path 至少需要 /<user>/<app>" >&2
  exit 1
fi

user="${parts[0]}"
app="${parts[1]}"
package_path=""
if [[ ${#parts[@]} -gt 2 ]]; then
  package_path="$(IFS=/; echo "${parts[*]:2}")"
fi

find_repo_root() {
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/go.mod" ]] && grep -q 'module github.com/kageos/kageos' "$dir/go.mod" && [[ -d "$dir/namespace" ]]; then
      echo "$dir"
      return 0
    fi
    if [[ -d "$dir/kageos/namespace" && -f "$dir/kageos/go.mod" ]] && grep -q 'module github.com/kageos/kageos' "$dir/kageos/go.mod"; then
      echo "$dir/kageos"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

if [[ -z "$repo_root" ]]; then
  if ! repo_root="$(find_repo_root)"; then
    echo "错误: 无法自动定位 kageos 主仓库，请传入 kageos_repo_root" >&2
    exit 1
  fi
fi

app_root="$repo_root/namespace/$user/$app"
api_root="$app_root/code/api"
main_go="$app_root/code/cmd/app/main.go"
package_dir="$api_root"
router_group="/"
import_path=""

if [[ -n "$package_path" ]]; then
  package_dir="$api_root/$package_path"
  router_group="/$package_path"
  import_path="github.com/kageos/kageos/namespace/$user/$app/code/api/$package_path"
fi

echo "kageos 仓库: $repo_root"
echo "full_code_path: /$trimmed"
echo "user: $user"
echo "app: $app"
echo "package_path: ${package_path:-<应用根>}"
echo "app_root: $app_root"
echo "package_dir: $package_dir"
echo "router_group: $router_group"
if [[ -n "$import_path" ]]; then
  echo "blank_import: _ \"$import_path\""
fi
echo

[[ -d "$app_root" ]] && echo "OK: app_root 存在" || echo "缺失: app_root 不存在"
[[ -f "$app_root/go.mod" ]] && echo "OK: go.mod 存在" || echo "缺失: go.mod 不存在"
[[ -f "$main_go" ]] && echo "OK: main.go 存在" || echo "缺失: main.go 不存在"
if [[ -n "$package_path" ]]; then
  [[ -d "$package_dir" ]] && echo "OK: package_dir 存在" || echo "提示: package_dir 尚不存在，可创建"
  [[ -f "$package_dir/init_.go" ]] && echo "OK: init_.go 存在" || echo "提示: init_.go 尚不存在"
  if [[ -f "$main_go" && -n "$import_path" ]] && grep -q "$import_path" "$main_go"; then
    echo "OK: main.go 已 blank import 该 package"
  elif [[ -n "$package_path" ]]; then
    echo "提示: main.go 尚未 blank import 该 package"
  fi
fi
