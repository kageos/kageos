#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_PATH="${KAGEOS_CONFIG:-.kageos/prod/kage.yaml}"
BASE_URL="${KAGEOS_BASE_URL:-}"
TLS_MODE="${KAGEOS_TLS_MODE:-auto}"
HTTP_PORT="${KAGEOS_HTTP_PORT:-}"
HTTPS_PORT="${KAGEOS_HTTPS_PORT:-}"
TIMEZONE="${KAGEOS_TIMEZONE:-}"
DEPLOY_USER="${KAGEOS_DEPLOY_USER:-}"
SKIP_UP=0
UP_ARGS=()

usage() {
  cat <<'EOF'
Kageos production installer

Usage:
  sudo ./install.sh --base-url app.example.com
  sudo ./install.sh --base-url http://your-ip-or-domain
  sudo ./install.sh --base-url http://your-ip-or-domain:8080 --http-port 8080
  sudo ./install.sh --base-url http://your-ip-or-domain --timezone Asia/Shanghai

Options:
  --base-url URL       Create .kageos/prod/kage.yaml when it does not exist.
  --tls-mode MODE      auto, http, https, redirect, or external. Defaults to auto.
  --timezone TZ        Deployment timezone. Defaults to Asia/Shanghai.
  --http-port PORT     HTTP listen port. Defaults to 80, or the port in --base-url.
  --https-port PORT    HTTPS listen port. Defaults to 443, or the port in --base-url.
  --user USER          Deploy as USER. Defaults to the sudo caller, then current user.
  --skip-up            Prepare the host and config, but do not start deployment.
  --help               Show this help.
  --                   Pass remaining arguments to ./prod-up.sh.

Environment:
  KAGEOS_BASE_URL      Same as --base-url.
  KAGEOS_TLS_MODE      Same as --tls-mode.
  KAGEOS_TIMEZONE      Same as --timezone.
  KAGEOS_HTTP_PORT     Same as --http-port.
  KAGEOS_HTTPS_PORT    Same as --https-port.
  KAGEOS_DEPLOY_USER   Same as --user.
  KAGEOS_CONFIG        Config path, default .kageos/prod/kage.yaml.
EOF
}

validate_port() {
  local name="$1"
  local value="$2"
  if [[ ! "$value" =~ ^[0-9]+$ || "$value" -lt 1 || "$value" -gt 65535 ]]; then
    echo "ERROR: $name must be a TCP port between 1 and 65535" >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-url)
      shift
      if [[ $# -eq 0 || -z "$1" ]]; then
        echo "ERROR: --base-url requires a value" >&2
        exit 1
      fi
      BASE_URL="$1"
      ;;
    --http-port)
      shift
      if [[ $# -eq 0 || -z "$1" ]]; then
        echo "ERROR: --http-port requires a value" >&2
        exit 1
      fi
      validate_port "--http-port" "$1"
      HTTP_PORT="$1"
      ;;
    --tls-mode)
      shift
      if [[ $# -eq 0 || -z "$1" ]]; then
        echo "ERROR: --tls-mode requires a value" >&2
        exit 1
      fi
      case "$1" in
        auto|http|https|redirect|external)
          TLS_MODE="$1"
          ;;
        *)
          echo "ERROR: --tls-mode must be auto, http, https, redirect, or external" >&2
          exit 1
          ;;
      esac
      ;;
    --timezone)
      shift
      if [[ $# -eq 0 || -z "$1" ]]; then
        echo "ERROR: --timezone requires a value" >&2
        exit 1
      fi
      TIMEZONE="$1"
      ;;
    --https-port)
      shift
      if [[ $# -eq 0 || -z "$1" ]]; then
        echo "ERROR: --https-port requires a value" >&2
        exit 1
      fi
      validate_port "--https-port" "$1"
      HTTPS_PORT="$1"
      ;;
    --user)
      shift
      if [[ $# -eq 0 || -z "$1" ]]; then
        echo "ERROR: --user requires a value" >&2
        exit 1
      fi
      DEPLOY_USER="$1"
      ;;
    --skip-up)
      SKIP_UP=1
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    --)
      shift
      UP_ARGS+=("$@")
      break
      ;;
    *)
      echo "ERROR: unsupported argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

if [[ -n "$HTTP_PORT" ]]; then
  validate_port "KAGEOS_HTTP_PORT/--http-port" "$HTTP_PORT"
fi
if [[ -n "$HTTPS_PORT" ]]; then
  validate_port "KAGEOS_HTTPS_PORT/--https-port" "$HTTPS_PORT"
fi
current_user() {
  id -un
}

if [[ -z "$DEPLOY_USER" ]]; then
  if [[ "${EUID:-$(id -u)}" -eq 0 && -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
    DEPLOY_USER="$SUDO_USER"
  else
    DEPLOY_USER="$(current_user)"
  fi
fi

if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  echo "ERROR: deploy user does not exist: $DEPLOY_USER" >&2
  echo "Create it first or pass --user with an existing user." >&2
  exit 1
fi

run_privileged() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

run_as_deploy() {
  if [[ "$(current_user)" == "$DEPLOY_USER" ]]; then
    "$@"
  elif command -v sudo >/dev/null 2>&1; then
    sudo -H -u "$DEPLOY_USER" "$@"
  elif [[ "${EUID:-$(id -u)}" -eq 0 ]] && command -v runuser >/dev/null 2>&1; then
    runuser -u "$DEPLOY_USER" -- "$@"
  else
    echo "ERROR: sudo is required to run commands as $DEPLOY_USER" >&2
    exit 1
  fi
}

run_in_repo() {
  run_as_deploy bash -lc 'cd "$1"; shift; exec "$@"' _ "$ROOT_DIR" "$@"
}

resolve_path() {
  local path="$1"
  if [[ "$path" = /* ]]; then
    printf '%s\n' "$path"
  else
    printf '%s/%s\n' "$ROOT_DIR" "$path"
  fi
}

CONFIG_ABS="$(resolve_path "$CONFIG_PATH")"

echo "Kageos production install"
echo "repo:        $ROOT_DIR"
echo "deploy user: $DEPLOY_USER"
echo "config:      $CONFIG_PATH"
if [[ -n "$HTTP_PORT" ]]; then
  echo "http port:   $HTTP_PORT"
fi
if [[ -n "$HTTPS_PORT" ]]; then
  echo "https port:  $HTTPS_PORT"
fi
if [[ -n "$TIMEZONE" ]]; then
  echo "timezone:    $TIMEZONE"
fi
if [[ -n "$TLS_MODE" ]]; then
  echo "tls mode:    $TLS_MODE"
fi
echo

if [[ "$DEPLOY_USER" != "root" ]] && command -v loginctl >/dev/null 2>&1; then
  linger="$(loginctl show-user "$DEPLOY_USER" -p Linger --value 2>/dev/null || true)"
  if [[ "$linger" != "yes" ]]; then
    echo "Enabling systemd linger for $DEPLOY_USER..."
    run_privileged loginctl enable-linger "$DEPLOY_USER"
  fi
  linger="$(loginctl show-user "$DEPLOY_USER" -p Linger --value 2>/dev/null || true)"
  if [[ "$linger" != "yes" ]]; then
    echo "ERROR: failed to enable systemd linger for $DEPLOY_USER" >&2
    echo "Run once: sudo loginctl enable-linger $DEPLOY_USER" >&2
    exit 1
  fi
  echo "linger:      enabled"
elif [[ "$DEPLOY_USER" != "root" ]]; then
  echo "WARN: loginctl not found; skip linger setup."
fi

if ! run_in_repo test -r go.mod; then
  echo "ERROR: deploy user cannot read repo: $ROOT_DIR" >&2
  exit 1
fi
if ! run_in_repo test -w .; then
  echo "ERROR: deploy user cannot write repo: $ROOT_DIR" >&2
  echo "Fix ownership/permissions or run with --user $(current_user)." >&2
  exit 1
fi
run_as_deploy mkdir -p "$(dirname "$CONFIG_ABS")"

if ! run_in_repo bash -lc 'command -v go >/dev/null 2>&1'; then
  echo "ERROR: Go is required for kagectl. Install Go 1.25 or newer, then retry." >&2
  exit 1
fi

if run_in_repo bash -lc 'command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1'; then
  echo "compose:     podman compose"
elif run_in_repo bash -lc 'command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1'; then
  echo "compose:     docker compose"
else
  echo "ERROR: podman compose or docker compose is required." >&2
  exit 1
fi

if [[ ! -f "$CONFIG_ABS" ]]; then
  if [[ -z "$BASE_URL" ]]; then
    echo "ERROR: prod config not found: $CONFIG_PATH" >&2
    echo "Pass --base-url for first install, for example:" >&2
    echo "  sudo ./install.sh --base-url app.example.com" >&2
    exit 1
  fi
  echo "Creating prod config..."
  init_args=(init --base-url "$BASE_URL" --tls-mode "$TLS_MODE")
  if [[ -n "$HTTP_PORT" ]]; then
    init_args+=(--http-port "$HTTP_PORT")
  fi
  if [[ -n "$HTTPS_PORT" ]]; then
    init_args+=(--https-port "$HTTPS_PORT")
  fi
  if [[ -n "$TIMEZONE" ]]; then
    init_args+=(--timezone "$TIMEZONE")
  fi
  run_in_repo go run ./cmd/kagectl "${init_args[@]}"
else
  echo "config:      exists"
fi

if [[ "$SKIP_UP" == "1" ]]; then
  echo "Host and config are ready. Skipped deployment start (--skip-up)."
  exit 0
fi

echo
echo "Starting production deployment..."
deploy_env=(KAGEOS_CONFIG="$CONFIG_PATH")
if [[ -n "$TLS_MODE" && "$TLS_MODE" != "auto" ]]; then
  deploy_env+=(KAGEOS_TLS_MODE="$TLS_MODE")
else
  deploy_env+=(KAGEOS_TLS_MODE="")
fi
if [[ -n "$HTTP_PORT" ]]; then
  deploy_env+=(KAGEOS_HTTP_PORT="$HTTP_PORT")
fi
if [[ -n "$HTTPS_PORT" ]]; then
  deploy_env+=(KAGEOS_HTTPS_PORT="$HTTPS_PORT")
fi
if [[ -n "$TIMEZONE" ]]; then
  deploy_env+=(KAGEOS_TIMEZONE="$TIMEZONE")
fi
run_in_repo env "${deploy_env[@]}" ./prod-up.sh "${UP_ARGS[@]}"
echo
echo "Follow logs:"
echo "  tail -f .kageos/prod/kagectl-up.log"
