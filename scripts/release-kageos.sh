#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/release-kageos.sh VERSION [--skip-checks]

Runs guarded local checks, pushes main, creates the vVERSION tag, and pushes
that tag so GitHub Actions performs the cloud release.

This script intentionally does not auto-tag kageos-sdk. Run SDK tests, commit,
tag, and push SDK first when SDK changed, then bump kageos/go.mod to that tag.
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
shift || true
skip_checks=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-checks) skip_checks=1 ;;
    *) die "unknown option: $1" ;;
  esac
  shift
done

version="${version#v}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] || die "VERSION must look like 0.1.67, got: $version"
tag="v${version}"

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || die "run inside the kageos repository"
cd "$repo_root"

scripts/release-preflight.sh "$version"

if [[ "$skip_checks" != "1" ]]; then
  log "Running release guard checks"
  bash scripts/check-sensitive-files.sh
  bash scripts/check-repo-size-guard.sh
  bash scripts/check-doc-links.sh
  bash scripts/check-sdk-boundaries.sh
  go vet -tags exclude_graphdriver_btrfs ./cmd/... ./core/... ./dto/... ./pkg/...
  bash scripts/test-core-go.sh
  npm --prefix web run check:architecture
  npm --prefix web run lint
  npm --prefix web run type-check
  npm --prefix web run test:unit -- --run
  npm --prefix web run build
  git diff --check
fi

log "Pushing main"
git push origin main

log "Creating release tag ${tag}"
git tag -a "$tag" -m "$tag"

log "Pushing release tag ${tag}"
git push origin "$tag"

log "Cloud release triggered"
if command -v gh >/dev/null 2>&1; then
  gh run list --workflow "Kagebase Release" --limit 1
  gh run list --workflow "Docker Release" --limit 1
fi
