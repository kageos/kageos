#!/usr/bin/env bash
set -u

repo=""
failures=0
warnings=0

usage() {
  echo "Usage: contributor_doctor.sh [--repo /path/to/kageos]"
}

pass() { printf '[PASS] %s\n' "$1"; }
warn() { printf '[WARN] %s\n' "$1"; warnings=$((warnings + 1)); }
fail() { printf '[FAIL] %s\n' "$1"; failures=$((failures + 1)); }

version_at_least() {
  local actual="$1" required_major="$2" required_minor="$3"
  local major minor
  major="${actual%%.*}"
  minor="${actual#*.}"
  minor="${minor%%.*}"
  [[ "$major" =~ ^[0-9]+$ && "$minor" =~ ^[0-9]+$ ]] || return 1
  (( major > required_major || (major == required_major && minor >= required_minor) ))
}

run_quiet_timeout() {
  local seconds="$1"
  shift
  "$@" >/dev/null 2>&1 &
  local command_pid=$!
  local ticks=$((seconds * 10))
  local tick
  for ((tick = 0; tick < ticks; tick++)); do
    if ! kill -0 "$command_pid" 2>/dev/null; then
      wait "$command_pid"
      return $?
    fi
    sleep 0.1
  done
  kill "$command_pid" 2>/dev/null || true
  wait "$command_pid" 2>/dev/null || true
  return 124
}

while (($#)); do
  case "$1" in
    --repo)
      [[ $# -ge 2 ]] || { usage >&2; exit 64; }
      repo="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 64
      ;;
  esac
done

printf 'kageos contributor doctor\n'
printf 'OS: %s %s\n' "$(uname -s)" "$(uname -m)"

if command -v git >/dev/null 2>&1; then
  pass "Git available: $(git --version)"
else
  fail "Git is missing"
fi

if command -v go >/dev/null 2>&1; then
  go_version="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
  if version_at_least "$go_version" 1 25; then
    pass "Go $go_version satisfies 1.25+"
  else
    fail "Go $go_version does not satisfy 1.25+"
  fi
else
  fail "Go is missing (requires 1.25+)"
fi

if command -v node >/dev/null 2>&1; then
  node_version="$(node --version | sed 's/^v//')"
  node_major="${node_version%%.*}"
  if version_at_least "$node_version" 22 12 || { [[ "$node_major" == "20" ]] && version_at_least "$node_version" 20 19; } || [[ "$node_major" =~ ^[0-9]+$ && "$node_major" -gt 22 ]]; then
    pass "Node.js $node_version is supported"
  else
    fail "Node.js $node_version is unsupported (requires 20.19+, 22.12+, or newer major)"
  fi
else
  fail "Node.js is missing"
fi

if command -v npm >/dev/null 2>&1; then
  pass "npm available: $(npm --version)"
else
  fail "npm is missing"
fi

docker_ready=0
podman_ready=0

if command -v docker >/dev/null 2>&1; then
  if run_quiet_timeout 10 docker compose version; then
    if run_quiet_timeout 10 docker info; then
      pass "Docker and Docker Compose are ready"
      docker_ready=1
    else
      warn "Docker CLI/Compose exists, but the daemon is not ready"
    fi
  else
    warn "Docker exists, but Docker Compose is unavailable"
  fi
else
  warn "Docker is not installed"
fi

if command -v podman >/dev/null 2>&1; then
  if run_quiet_timeout 10 podman compose version; then
    if run_quiet_timeout 10 podman info; then
      pass "Podman and Podman Compose are ready"
      podman_ready=1
    else
      warn "Podman CLI/Compose exists, but the runtime is not ready; inspect machine and connections before any init/reset"
    fi
  else
    warn "Podman exists, but Podman Compose is unavailable"
  fi
else
  warn "Podman is not installed"
fi

if ((docker_ready == 0 && podman_ready == 0)); then
  fail "No healthy Docker Compose or Podman Compose runtime is available"
fi

if [[ -n "$repo" ]]; then
  if [[ ! -d "$repo" ]]; then
    fail "Repository path does not exist: $repo"
  elif [[ -f "$repo/.kageos-root" && -f "$repo/go.mod" && -d "$repo/cmd/kagectl" ]]; then
    pass "kageos source checkout detected: $repo"
    if git -C "$repo" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      branch="$(git -C "$repo" branch --show-current 2>/dev/null || true)"
      [[ -n "$branch" ]] && pass "Git branch: $branch" || warn "Git checkout is detached"
      if [[ -n "$(git -C "$repo" status --short 2>/dev/null)" ]]; then
        warn "Working tree has local changes; preserve them before contribution work"
      else
        pass "Working tree is clean"
      fi
    else
      warn "Source directory is not a Git worktree"
    fi
  else
    fail "Path is not a confirmed kageos source checkout: $repo"
  fi
fi

printf 'Summary: %d failure(s), %d warning(s)\n' "$failures" "$warnings"
if ((failures > 0)); then
  exit 2
fi
