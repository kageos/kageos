#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
ENGINE="${DEV_RUNTIME:-auto}"

pick_engine() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    echo docker
    return 0
  fi
  if command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then
    echo podman
    return 0
  fi
  echo "ERROR: 未找到 docker compose 或 podman compose，请先安装。" >&2
  exit 1
}

resolve_engine() {
  case "$1" in
    auto)
      pick_engine
      ;;
    docker)
      command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 || {
        echo "ERROR: docker compose 不可用。" >&2
        exit 1
      }
      echo docker
      ;;
    podman)
      command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1 || {
        echo "ERROR: podman compose 不可用。" >&2
        exit 1
      }
      echo podman
      ;;
    *)
      echo "ERROR: 不支持的引擎: $1（仅支持 docker / podman / auto）" >&2
      exit 1
      ;;
  esac
}

if [[ "${1:-}" == "docker" || "${1:-}" == "podman" || "${1:-}" == "auto" ]]; then
  ENGINE="$1"
  shift
fi

ENGINE="$(resolve_engine "$ENGINE")"

if [[ "$ENGINE" == "docker" ]]; then
  COMPOSE_FILE="$ROOT/deploy/dev/compose/docker-compose.dev.yml"
  COMPOSE_CMD=(docker compose -f "$COMPOSE_FILE")
else
  COMPOSE_FILE="$ROOT/deploy/dev/compose/docker-compose.infra.yml"
  COMPOSE_CMD=(podman compose -f "$COMPOSE_FILE")
fi

if [[ "$#" -eq 0 ]]; then
  set -- up -d
fi

echo "==> dev infra engine: $ENGINE"
echo "==> compose file: $COMPOSE_FILE"

cd "$ROOT"
"${COMPOSE_CMD[@]}" "$@"
