#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/release-docker.sh VERSION

Examples:
  scripts/release-docker.sh 0.1.0
  KAGEOS_DOCKER_IMAGE=your-dockerhub-user/kageos scripts/release-docker.sh 0.1.0

Environment:
  KAGEOS_DOCKER_IMAGE       Image name. Default: qiayanai/kageos
  KAGEOS_DOCKER_PLATFORMS   Platforms. Default: linux/amd64,linux/arm64
  KAGEOS_DOCKER_LATEST      Also push :latest. Default: 1

This script publishes a multi-architecture manifest, so users do not choose
amd64 or arm64 manually. Docker selects the matching image automatically.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

version="${1:-}"
if [[ -z "$version" ]]; then
  usage >&2
  exit 1
fi
version="${version#v}"
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "ERROR: VERSION must look like 0.1.0, got: $version" >&2
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$repo_root"

image="${KAGEOS_DOCKER_IMAGE:-qiayanai/kageos}"
platforms="${KAGEOS_DOCKER_PLATFORMS:-linux/amd64,linux/arm64}"
push_latest="${KAGEOS_DOCKER_LATEST:-1}"

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is required for local multi-arch publishing." >&2
  echo "       Prefer GitHub Actions Docker Release if this machine only has Podman." >&2
  exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
  echo "ERROR: docker buildx is required." >&2
  exit 1
fi

if ! docker buildx inspect kageos-release >/dev/null 2>&1; then
  docker buildx create --name kageos-release --use >/dev/null
else
  docker buildx use kageos-release >/dev/null
fi

tags=(-t "${image}:${version}")
if [[ "$push_latest" == "1" || "$push_latest" == "true" ]]; then
  tags+=(-t "${image}:latest")
fi

echo "==> Building and pushing ${image}:${version}"
echo "==> Platforms: ${platforms}"
docker buildx build \
  --platform "$platforms" \
  -f deploy/aio/Dockerfile \
  "${tags[@]}" \
  --build-arg APT_USE_MIRROR=0 \
  --build-arg GOPROXY=https://proxy.golang.org,direct \
  --build-arg GOSUMDB=sum.golang.org \
  --build-arg NPM_REGISTRY=https://registry.npmjs.org \
  --push \
  .
