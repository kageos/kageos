#!/usr/bin/env bash
# 从 env.yaml 生成 .env（需 PyYAML：pip3 install pyyaml）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
SRC="${1:-$ROOT/env.yaml}"
OUT="${2:-$ROOT/.env}"

if [[ ! -f "$SRC" ]]; then
  echo "未找到 $SRC，请 cp env.yaml.example env.yaml 并编辑"
  exit 1
fi

python3 - "$SRC" "$OUT" <<'PY'
import sys
try:
    import yaml
except ImportError:
    print("请先安装: pip3 install pyyaml", file=sys.stderr)
    sys.exit(1)

src, out = sys.argv[1], sys.argv[2]
with open(src, "r", encoding="utf-8") as f:
    d = yaml.safe_load(f) or {}

site = d.get("site") or {}
sec = d.get("secrets") or {}
ports = d.get("ports") or {}
img = d.get("image") or {}

def esc(v: str) -> str:
    return str(v).replace("\\", "\\\\").replace("\n", " ").replace('"', '\\"')

lines = [
    f'CANONICAL_BASE_URL="{esc(site.get("canonical_base_url", "https://geeleo.com"))}"',
    f'MYSQL_ROOT_PASSWORD="{esc(sec.get("mysql_root_password", "changeme"))}"',
    f'JWT_SECRET="{esc(sec.get("jwt_secret", "change-me"))}"',
    f'CONTROL_ENC_KEY="{esc(sec.get("control_encryption_key", "ai-agent-os-license-key-32bytes!"))}"',
    f'MINIO_ROOT_USER="{esc(sec.get("minio_root_user", "minioadmin"))}"',
    f'MINIO_ROOT_PASSWORD="{esc(sec.get("minio_root_password", "minioadmin123"))}"',
    f'SMTP_PASSWORD="{esc(sec.get("smtp_password", ""))}"',
    f'HTTP_PUBLISH_PORT="{esc(str(ports.get("http_publish", 80)))}"',
    f'MAIN_IMAGE="{esc(img.get("main", "ai-agent-os-main:latest"))}"',
]
with open(out, "w", encoding="utf-8") as f:
    f.write("\n".join(lines) + "\n")
print("已写入:", out)
PY
