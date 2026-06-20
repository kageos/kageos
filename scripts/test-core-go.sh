#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Keep generated workspace applications out of the core backend test suite.
# `go test ./...` intentionally walks namespace/ and can fail on user apps that
# are mid-generation or contain experimental code.
go test \
  -tags "${GO_TEST_TAGS:-exclude_graphdriver_btrfs}" \
  ./cmd/... \
  ./core/... \
  ./dto/... \
  ./pkg/... \
  ./sdk/... \
  ./scripts/sync-case-catalog/...
