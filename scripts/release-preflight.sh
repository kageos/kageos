#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/release-preflight.sh VERSION

Checks the parts of a kageos release that are easy to miss:
  - kageos main is clean and on main
  - the target platform tag does not already exist
  - the sibling kageos-sdk repo is clean
  - kageos-sdk HEAD is exactly on its newest local tag
  - kageos/go.mod requires that same SDK tag

Environment:
  KAGEOS_SDK_REPO  Path to kageos-sdk. Default: ../kageos-sdk
EOF
}

log() {
  printf '==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

version="${1:-}"
[[ -n "$version" ]] || {
  usage >&2
  exit 1
}
version="${version#v}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] || die "VERSION must look like 0.1.67, got: $version"
tag="v${version}"

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || die "run inside the kageos repository"
cd "$repo_root"
sdk_repo="${KAGEOS_SDK_REPO:-$(cd "$repo_root/.." && pwd)/kageos-sdk}"

[[ -f ".kageos-root" && -f "go.mod" && -d "cmd/kagectl" ]] || die "this does not look like the kageos platform repository"
[[ -d "$sdk_repo/.git" && -f "$sdk_repo/go.mod" ]] || die "kageos-sdk repo not found: $sdk_repo"

log "Checking kageos repository"
current_branch="$(git branch --show-current)"
[[ "$current_branch" == "main" ]] || die "kageos must release from main, current branch is: $current_branch"
[[ -z "$(git status --porcelain)" ]] || die "kageos has uncommitted changes"
git fetch --quiet origin main --tags
local_head="$(git rev-parse HEAD)"
origin_head="$(git rev-parse origin/main)"
[[ "$local_head" == "$origin_head" ]] || die "kageos main is not synced with origin/main"
if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
  die "local tag already exists: $tag"
fi
if git ls-remote --exit-code --tags origin "refs/tags/${tag}" >/dev/null 2>&1; then
  die "remote tag already exists: $tag"
fi

log "Checking kageos-sdk repository"
git -C "$sdk_repo" fetch --quiet origin main --tags
sdk_branch="$(git -C "$sdk_repo" branch --show-current)"
[[ "$sdk_branch" == "main" ]] || die "kageos-sdk must be on main, current branch is: $sdk_branch"
[[ -z "$(git -C "$sdk_repo" status --porcelain)" ]] || die "kageos-sdk has uncommitted changes; publish or commit SDK first"
sdk_head="$(git -C "$sdk_repo" rev-parse HEAD)"
sdk_origin_head="$(git -C "$sdk_repo" rev-parse origin/main)"
[[ "$sdk_head" == "$sdk_origin_head" ]] || die "kageos-sdk main is not synced with origin/main"
sdk_tag="$(git -C "$sdk_repo" tag --points-at HEAD | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+' | sort -V | tail -n 1 || true)"
[[ -n "$sdk_tag" ]] || die "kageos-sdk HEAD is not tagged; tag and push SDK before releasing kageos"

required_sdk="$(go list -m -f '{{.Version}}' github.com/kageos/kageos-sdk)"
[[ "$required_sdk" == "$sdk_tag" ]] || die "kageos/go.mod requires ${required_sdk}, but kageos-sdk HEAD is ${sdk_tag}; run go get github.com/kageos/kageos-sdk@${sdk_tag}"

log "Release preflight passed for ${tag} with SDK ${sdk_tag}"
