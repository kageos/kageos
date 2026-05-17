#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TOP_N="${TOP_N:-50}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

records="$TMP_DIR/records.tsv"
markers="$TMP_DIR/markers.txt"
: > "$records"
: > "$markers"

is_text_path() {
  case "$1" in
    *.go|*.ts|*.tsx|*.vue|*.js|*.mjs|*.cjs|*.css|*.scss|*.html|*.md|*.json|*.yaml|*.yml|*.toml|*.sql|*.sh|*.txt|*.conf|*.inc|*.te|*.template|*.mod|*.sum) return 0 ;;
    Dockerfile|*/Dockerfile|go.mod|go.sum|.gitignore|*/.gitignore) return 0 ;;
  esac
  return 1
}

is_generated_or_data() {
  case "$1" in
    */docs/docs.go|*/docs/swagger.json|*/docs/swagger.yaml) return 0 ;;
    web/package-lock.json|web/components.d.ts) return 0 ;;
    *.capability-bundle.json) return 0 ;;
    core/agent-server/prompt/system_prompt_source.go|core/agent-server/prompt/embed.go) return 0 ;;
  esac
  return 1
}

is_source_path() {
  case "$1" in
    *.go|*.ts|*.tsx|*.vue|*.js|*.mjs|*.cjs|*.css|*.scss|*.html|*.sql|*.sh) return 0 ;;
    Dockerfile|*/Dockerfile) return 0 ;;
  esac
  return 1
}

should_scan_markers() {
  case "$1" in
    scripts/codebase-health-report.sh) return 1 ;;
  esac
  return 0
}

file_type() {
  case "$1" in
    *.go) echo "Go" ;;
    *.vue) echo "Vue" ;;
    *.ts) echo "TypeScript" ;;
    *.tsx) echo "TSX" ;;
    *.js|*.mjs|*.cjs) echo "JavaScript" ;;
    *.css|*.scss) echo "CSS" ;;
    *.html) echo "HTML" ;;
    *.md) echo "Markdown" ;;
    *.json) echo "JSON" ;;
    *.yaml|*.yml) echo "YAML" ;;
    *.toml) echo "TOML" ;;
    *.sql) echo "SQL" ;;
    *.sh) echo "Shell" ;;
    *.mod|*.sum) echo "Go module" ;;
    Dockerfile|*/Dockerfile) echo "Dockerfile" ;;
    *) echo "Text/Config" ;;
  esac
}

top_bucket() {
  case "$1" in
    web/src/*) echo "web/src" ;;
    core/agent-server/*) echo "core/agent-server" ;;
    core/app-server/*) echo "core/app-server" ;;
    core/app-storage/*) echo "core/app-storage" ;;
    core/app-runtime/*) echo "core/app-runtime" ;;
    core/api-gateway/*) echo "core/api-gateway" ;;
    core/hr-server/*) echo "core/hr-server" ;;
    sdk/agent-app/*) echo "sdk/agent-app" ;;
    pkg/*) echo "pkg" ;;
    dto/*) echo "dto" ;;
    deploy/*) echo "deploy" ;;
    docs/*) echo "docs" ;;
    scripts/*) echo "scripts" ;;
    web/*) echo "web" ;;
    core/*) echo "core" ;;
    *) echo "${1%%/*}" ;;
  esac
}

line_count() {
  wc -l < "$1" | tr -d ' '
}

git ls-files -z --cached --others --exclude-standard | while IFS= read -r -d '' path; do
  if [ ! -f "$path" ]; then
    continue
  fi

  if ! is_text_path "$path"; then
    continue
  fi

  lines="$(line_count "$path")"
  type="$(file_type "$path")"
  bucket="$(top_bucket "$path")"
  scope="other_text"

  if is_generated_or_data "$path"; then
    scope="generated_or_data"
  elif is_source_path "$path"; then
    scope="handwritten_source"
  fi

  printf '%s\t%s\t%s\t%s\t%s\n' "$lines" "$type" "$bucket" "$scope" "$path" >> "$records"

  if [ "$scope" = "handwritten_source" ] && should_scan_markers "$path"; then
    grep -Eio 'TODO|FIXME|legacy|deprecated|兼容|临时' "$path" 2>/dev/null >> "$markers" || true
  fi
done

metric() {
  local scope="$1"
  local field="$2"
  awk -F $'\t' -v scope="$scope" -v field="$field" '
    scope == "all" || $4 == scope {
      files += 1
      lines += $1
    }
    END {
      if (field == "files") print files + 0
      else print lines + 0
    }
  ' "$records"
}

print_group_table() {
  local title="$1"
  local column="$2"
  local limit="${3:-20}"
  local scope="${4:-all}"
  local sorted="$TMP_DIR/group-${title// /-}.tsv"

  printf '\n## %s\n\n' "$title"
  printf '| %s | 文件数 | 行数 |\n' "$column"
  printf '| --- | ---: | ---: |\n'
  awk -F $'\t' -v column="$column" -v scope="$scope" '
    scope == "all" || $4 == scope {
      key = (column == "类型") ? $2 : $3
      files[key] += 1
      lines[key] += $1
    }
    END {
      for (key in files) {
        printf "%d\t%d\t%s\n", lines[key], files[key], key
      }
    }
  ' "$records" | sort -nr > "$sorted"

  sed -n "1,${limit}p" "$sorted" | while IFS=$'\t' read -r lines files key; do
    printf '| `%s` | %s | %s |\n' "$key" "$files" "$lines"
  done
}

print_group_table_by_type() {
  local title="$1"
  local limit="${2:-20}"
  local scope="${3:-all}"
  local sorted="$TMP_DIR/group-${title// /-}.tsv"

  printf '\n## %s\n\n' "$title"
  printf '| 类型 | 文件数 | 行数 |\n'
  printf '| --- | ---: | ---: |\n'
  awk -F $'\t' -v scope="$scope" '
    {
      if (scope != "all" && $4 != scope) {
        next
      }
      key = $2
      files[key] += 1
      lines[key] += $1
    }
    END {
      for (key in files) {
        printf "%d\t%d\t%s\n", lines[key], files[key], key
      }
    }
  ' "$records" | sort -nr > "$sorted"

  sed -n "1,${limit}p" "$sorted" | while IFS=$'\t' read -r lines files key; do
    printf '| `%s` | %s | %s |\n' "$key" "$files" "$lines"
  done
}

print_large_files() {
  local sorted="$TMP_DIR/large-files.tsv"

  printf '\n## 手写源码大文件 Top %s\n\n' "$TOP_N"
  printf '| 行数 | 文件 |\n'
  printf '| ---: | --- |\n'
  awk -F $'\t' '$4 == "handwritten_source" { printf "%d\t%s\n", $1, $5 }' "$records" |
    sort -nr > "$sorted"

  sed -n "1,${TOP_N}p" "$sorted" |
    while IFS=$'\t' read -r lines path; do
      printf '| %s | `%s` |\n' "$lines" "$path"
    done
}

print_large_file_summary() {
  printf '\n## 大文件数量\n\n'
  printf '| 阈值 | 文件数 |\n'
  printf '| ---: | ---: |\n'
  for threshold in 500 800 1200; do
    count="$(awk -F $'\t' -v threshold="$threshold" '$4 == "handwritten_source" && $1 >= threshold { count += 1 } END { print count + 0 }' "$records")"
    printf '| >= %s 行 | %s |\n' "$threshold" "$count"
  done
}

print_marker_summary() {
  printf '\n## 历史痕迹关键词\n\n'
  printf '| 关键词 | 次数 |\n'
  printf '| --- | ---: |\n'
  for marker in TODO FIXME "兼容" "临时" legacy deprecated; do
    count="$(awk -v marker="$marker" 'BEGIN { wanted = tolower(marker) } tolower($0) == wanted { count += 1 } END { print count + 0 }' "$markers")"
    printf '| `%s` | %s |\n' "$marker" "$count"
  done
}

print_marker_top_files() {
  local sorted="$TMP_DIR/marker-top-files.tsv"

  printf '\n## 历史痕迹文件 Top 20\n\n'
  printf '| 次数 | 文件 |\n'
  printf '| ---: | --- |\n'
  while IFS=$'\t' read -r _lines _type _bucket scope path; do
    if [ "$scope" != "handwritten_source" ]; then
      continue
    fi
    if ! should_scan_markers "$path"; then
      continue
    fi
    count="$({ grep -Eio 'TODO|FIXME|legacy|deprecated|兼容|临时' "$path" 2>/dev/null || true; } | wc -l | tr -d ' ')"
    if [ "$count" -gt 0 ]; then
      printf '%s\t%s\n' "$count" "$path"
    fi
  done < "$records" | sort -nr > "$sorted"

  sed -n '1,20p' "$sorted" | while IFS=$'\t' read -r count path; do
    printf '| %s | `%s` |\n' "$count" "$path"
  done
}

cat <<EOF
# Codebase Health Report

Generated at: $(date '+%Y-%m-%d %H:%M:%S %z')

## Summary

| 类别 | 文件数 | 行数 |
| --- | ---: | ---: |
| Tracked/unignored text files | $(metric all files) | $(metric all lines) |
| Handwritten source | $(metric handwritten_source files) | $(metric handwritten_source lines) |
| Generated/data files | $(metric generated_or_data files) | $(metric generated_or_data lines) |
| Other docs/config text | $(metric other_text files) | $(metric other_text lines) |
EOF

print_group_table_by_type "按文件类型统计" 20 "all"
print_group_table "按目录统计" "目录" 20
print_group_table "手写源码按目录统计" "目录" 20 "handwritten_source"
print_large_file_summary
print_large_files
print_marker_summary
print_marker_top_files

cat <<'EOF'

## Notes

- `Handwritten source` means handwritten code-like files and excludes Swagger docs, package-lock, generated component declarations, capability bundles, and prompt embed files.
- Use `TOP_N=100 scripts/codebase-health-report.sh` to expand the large-file list.
- This report is a sizing and refactor compass, not a quality score by itself.
EOF
