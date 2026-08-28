#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/verify-kageos-release.sh VERSION

Verifies the cloud release result:
  - Kagebase Release and Docker Release completed successfully for vVERSION
  - Docker Hub kageos/kagebase VERSION and latest manifests exist
  - optional Aliyun ACR manifests exist when ALIYUN_REGISTRY and ALIYUN_NAMESPACE are set
  - downloads latest.txt points at VERSION when reachable

Environment:
  ALIYUN_REGISTRY   Optional, for example crpi-xxx.cn-beijing.personal.cr.aliyuncs.com
  ALIYUN_NAMESPACE  Optional, for example qiayanai
  KAGEOS_LATEST_URL Optional, default https://downloads.kageos.com/releases/latest.txt
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
latest_url="${KAGEOS_LATEST_URL:-https://downloads.kageos.com/releases/latest.txt}"

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

if command -v gh >/dev/null 2>&1; then
  command -v jq >/dev/null 2>&1 || die "jq is required to parse gh output"
  for workflow in "Kagebase Release" "Docker Release"; do
    log "Checking GitHub Actions: ${workflow}"
    run_json="$(gh run list --workflow "$workflow" --branch "$tag" --limit 1 --json databaseId,status,conclusion,url)"
    [[ "$run_json" != "[]" ]] || die "no ${workflow} run found for ${tag}"
    status="$(printf '%s' "$run_json" | jq -r '.[0].status')"
    conclusion="$(printf '%s' "$run_json" | jq -r '.[0].conclusion')"
    url="$(printf '%s' "$run_json" | jq -r '.[0].url')"
    printf '%s: status=%s conclusion=%s %s\n' "$workflow" "$status" "$conclusion" "$url"
    [[ "$status" == "completed" && "$conclusion" == "success" ]] || die "${workflow} is not successful yet"
  done
else
  log "Skipping GitHub Actions check because gh is not installed"
fi

inspect_manifest() {
  local image_ref="$1"
  if command -v docker >/dev/null 2>&1; then
    docker buildx imagetools inspect "$image_ref"
    return
  fi
  if command -v podman >/dev/null 2>&1; then
    podman manifest inspect "$image_ref"
    return
  fi
  if command -v skopeo >/dev/null 2>&1; then
    skopeo inspect --raw "docker://${image_ref}"
    return
  fi
  die "docker, podman, or skopeo is required to inspect image manifests"
}

for image in kagebase kageos; do
  for ref in "$version" latest; do
    log "Inspecting docker.io/qiayanai/${image}:${ref}"
    manifest="$(inspect_manifest "docker.io/qiayanai/${image}:${ref}")"
    grep -Eq 'linux/amd64|"architecture"[[:space:]]*:[[:space:]]*"amd64"' <<<"$manifest" || die "${image}:${ref} missing linux/amd64"
    grep -Eq 'linux/arm64|"architecture"[[:space:]]*:[[:space:]]*"arm64"' <<<"$manifest" || die "${image}:${ref} missing linux/arm64"
  done
done

if [[ -n "${ALIYUN_REGISTRY:-}" && -n "${ALIYUN_NAMESPACE:-}" ]]; then
  command -v skopeo >/dev/null 2>&1 || die "skopeo is required for Aliyun ACR verification"
  for image in kagebase kageos; do
    for ref in "$version" latest; do
      log "Inspecting ${ALIYUN_REGISTRY}/${ALIYUN_NAMESPACE}/${image}:${ref}"
      skopeo inspect "docker://${ALIYUN_REGISTRY}/${ALIYUN_NAMESPACE}/${image}:${ref}" >/dev/null
    done
  done
else
  log "Skipping Aliyun ACR check; ALIYUN_REGISTRY and ALIYUN_NAMESPACE are not set"
fi

if command -v curl >/dev/null 2>&1; then
  log "Checking ${latest_url}"
  latest="$(curl -fsSL --retry 3 --retry-delay 2 --connect-timeout 20 "$latest_url" | tr -d '[:space:]')"
  latest="${latest#v}"
  [[ "$latest" == "$version" ]] || die "${latest_url} points at ${latest}, want ${version}"
fi

log "Release ${tag} verification passed"
