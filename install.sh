#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_PATH="${KAGEOS_CONFIG:-.kageos/prod/kage.yaml}"
BASE_URL="${KAGEOS_BASE_URL:-}"
DEPLOY_USER="${KAGEOS_DEPLOY_USER:-}"
SKIP_UP=0
UP_ARGS=()

usage() {
  cat <<'EOF'
Kageos production installer

Usage:
  sudo ./install.sh --base-url http://your-ip-or-domain

Options:
  --base-url URL   Create .kageos/prod/kage.yaml when it does not exist.
  --user USER      Deploy as USER. Defaults to the sudo caller, then current user.
  --skip-up        Prepare the host and config, but do not start deployment.
  --help           Show this help.
  --               Pass remaining arguments to ./prod-up.sh.

Environment:
  KAGEOS_BASE_URL      Same as --base-url.
  KAGEOS_DEPLOY_USER   Same as --user.
  KAGEOS_CONFIG        Config path, default .kageos/prod/kage.yaml.
EOF
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
    echo "  sudo ./install.sh --base-url http://your-ip-or-domain" >&2
    exit 1
  fi
  echo "Creating prod config..."
  run_in_repo go run ./cmd/kagectl init --base-url "$BASE_URL"
else
  echo "config:      exists"
fi

if [[ "$SKIP_UP" == "1" ]]; then
  echo "Host and config are ready. Skipped deployment start (--skip-up)."
  exit 0
fi

echo
echo "Starting production deployment..."
run_in_repo env KAGEOS_CONFIG="$CONFIG_PATH" ./prod-up.sh "${UP_ARGS[@]}"
echo
echo "Follow logs:"
echo "  tail -f .kageos/prod/kagectl-up.log"
