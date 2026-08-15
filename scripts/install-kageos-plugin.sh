#!/usr/bin/env bash
set -euo pipefail

base_url="${KAGEOS_PLUGIN_BASE_URL:-https://kageos.ai/downloads/kageos}"
plugin_parent="${KAGEOS_PLUGIN_HOME:-$HOME/plugins}"
plugin_target="$plugin_parent/kageos"
marketplace_file="${KAGEOS_MARKETPLACE_FILE:-$HOME/.agents/plugins/marketplace.json}"
temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT

for command_name in curl python3 unzip; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少必要命令：$command_name" >&2
    exit 1
  fi
done

curl -fsSL "$base_url/latest.json" -o "$temporary_dir/latest.json"
read -r version filename expected_sha <<EOF
$(python3 - "$temporary_dir/latest.json" <<'PY'
import json
import sys
data = json.load(open(sys.argv[1], encoding="utf-8"))
print(data["version"], data["filename"], data["sha256"])
PY
)
EOF

archive="$temporary_dir/$filename"
curl -fsSL "$base_url/$filename" -o "$archive"
actual_sha="$(python3 - "$archive" <<'PY'
import hashlib
import sys
print(hashlib.sha256(open(sys.argv[1], "rb").read()).hexdigest())
PY
)"
if [ "$actual_sha" != "$expected_sha" ]; then
  echo "SHA-256 校验失败，已停止安装。" >&2
  exit 1
fi

unzip -q "$archive" -d "$temporary_dir/unpacked"
python3 - "$temporary_dir/unpacked/kageos" "$version" <<'PY'
import json
import pathlib
import sys
root = pathlib.Path(sys.argv[1])
expected = sys.argv[2]
codex = json.loads((root / ".codex-plugin" / "plugin.json").read_text(encoding="utf-8"))
claude = json.loads((root / ".claude-plugin" / "plugin.json").read_text(encoding="utf-8"))
if codex.get("name") != "kageos" or codex.get("version") != expected or claude.get("version") != expected:
    raise SystemExit("插件名称或版本与 release metadata 不一致")
PY

mkdir -p "$plugin_parent"
if [ -e "$plugin_target" ]; then
  backup_root="$HOME/.local/state/kageos-plugin/backups"
  mkdir -p "$backup_root"
  backup_path="$backup_root/kageos-$(date -u +%Y%m%d%H%M%S)"
  mv "$plugin_target" "$backup_path"
  echo "旧版本已备份到：$backup_path"
fi
mv "$temporary_dir/unpacked/kageos" "$plugin_target"

mkdir -p "$(dirname "$marketplace_file")"
python3 - "$marketplace_file" <<'PY'
import json
import os
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
if path.exists():
    data = json.loads(path.read_text(encoding="utf-8"))
else:
    data = {"name": "personal", "interface": {"displayName": "Personal"}, "plugins": []}
plugins = data.setdefault("plugins", [])
entry = {
    "name": "kageos",
    "source": {"source": "local", "path": "./plugins/kageos"},
    "policy": {"installation": "AVAILABLE", "authentication": "ON_INSTALL"},
    "category": "DeveloperTools",
}
plugins[:] = [item for item in plugins if item.get("name") != "kageos"] + [entry]
temporary = path.with_name(path.name + ".tmp")
temporary.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
os.replace(temporary, path)
PY

echo "kageos 套件 $version 已安装到：$plugin_target"
echo "请刷新 Codex，在 Plugins → Personal 中启用 kageos，然后新建任务。"
echo "Claude Code 可运行：claude --plugin-dir $plugin_target"
