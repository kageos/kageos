#!/usr/bin/env bash
set -euo pipefail

limit="${1:-80}"

set +o pipefail
git rev-list --objects --all |
  git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(objectsize:disk) %(rest)' |
  awk '$1=="blob" {printf "%12d %12d %s %s\n", $3, $4, $2, $5}' |
  sort -nr |
  head -n "$limit"
