#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
ENGINE="${DEV_RUNTIME:-auto}"
ENV_FILE="$ROOT/.kageos/dev/env/kageos.env"

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

mysql_sql_quote() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\'/\'\'}"
  printf "'%s'" "$value"
}

try_migrate_legacy_mysql_root_password() {
  local quoted_password
  quoted_password="$(mysql_sql_quote "$MYSQL_ROOT_PASSWORD")"
  if "$ENGINE" exec "$MYSQL_CONTAINER" mysql -h 127.0.0.1 -uroot -proot -e "SELECT 1" >/dev/null 2>&1; then
    echo "WARN: MySQL volume still uses legacy root/root; migrating root password to .kageos/dev/env/kageos.env ..."
    "$ENGINE" exec "$MYSQL_CONTAINER" mysql -h 127.0.0.1 -uroot -proot -e \
      "ALTER USER 'root'@'%' IDENTIFIED BY ${quoted_password}; ALTER USER 'root'@'localhost' IDENTIFIED BY ${quoted_password}; FLUSH PRIVILEGES;"
    return 0
  fi
  return 1
}

if [[ "${1:-}" == "docker" || "${1:-}" == "podman" || "${1:-}" == "auto" ]]; then
  ENGINE="$1"
  shift
fi

ENGINE="$(resolve_engine "$ENGINE")"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
else
  echo "ERROR: dev env file not found: $ENV_FILE" >&2
  echo "ERROR: please run: go run ./cmd/kagectl init-dev --skip-base" >&2
  exit 1
fi

if [[ "$ENGINE" == "docker" ]]; then
  COMPOSE_FILE="$ROOT/deploy/dev/compose/docker-compose.dev.yml"
  COMPOSE_CMD=(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE")
else
  COMPOSE_FILE="$ROOT/deploy/dev/compose/docker-compose.infra.yml"
  COMPOSE_CMD=(podman compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE")
fi

if [[ "$#" -eq 0 ]]; then
  set -- up -d
fi

ACTION="$1"

echo "==> dev infra engine: $ENGINE"
echo "==> compose file: $COMPOSE_FILE"

cd "$ROOT"
"${COMPOSE_CMD[@]}" "$@"

# up 之后始终执行 init SQL（幂等），确保所有数据库存在
if [[ "$ACTION" == "up" ]]; then
  if [[ "$ENGINE" == "docker" ]]; then
    MYSQL_CONTAINER="kageos-dev-mysql"
  else
    MYSQL_CONTAINER="mysql8"
  fi

  INIT_SQL="$ROOT/deploy/base/infra/mysql/init-db.sql"

  if [[ ! -f "$INIT_SQL" ]]; then
    echo "WARN: init SQL 不存在: $INIT_SQL，跳过数据库初始化" >&2
  else
    echo "==> 等待 MySQL 就绪（最多 60s）..."
    MYSQL_READY=0
    for _ in $(seq 1 30); do
      if "$ENGINE" exec "$MYSQL_CONTAINER" mysql -h 127.0.0.1 -uroot -p"${MYSQL_ROOT_PASSWORD}" -e "SELECT 1" >/dev/null 2>&1; then
        MYSQL_READY=1
        break
      fi
      sleep 2
    done
    if [[ "$MYSQL_READY" != "1" ]]; then
      if try_migrate_legacy_mysql_root_password; then
        if "$ENGINE" exec "$MYSQL_CONTAINER" mysql -h 127.0.0.1 -uroot -p"${MYSQL_ROOT_PASSWORD}" -e "SELECT 1" >/dev/null 2>&1; then
          MYSQL_READY=1
        fi
      fi
    fi
    if [[ "$MYSQL_READY" != "1" ]]; then
      echo "ERROR: MySQL 未能使用 .kageos/dev/env/kageos.env 中的 MYSQL_ROOT_PASSWORD 登录。" >&2
      echo "ERROR: 常见原因是旧 MySQL volume 里保存的是历史密码，但 .kageos/dev/env/kageos.env 已换成新密码。" >&2
      echo "ERROR: 若可清空本地开发数据，请执行：bash deploy/dev/scripts/infra.sh ${ENGINE} down -v && go run ./cmd/kagectl init-dev --engine ${ENGINE} --regen-secrets" >&2
      exit 1
    fi

    echo "==> 执行数据库初始化 SQL（幂等）..."
    "$ENGINE" exec -i "$MYSQL_CONTAINER" mysql -h 127.0.0.1 -uroot -p"${MYSQL_ROOT_PASSWORD}" < "$INIT_SQL"
    echo "==> 数据库初始化完成"
  fi
fi
