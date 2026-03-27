#!/usr/bin/env bash
# 历史迁移工具：将旧 env.yaml 转成 .env
# 新部署流程以 .env 为唯一配置源，build.sh 不再依赖本脚本
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
SRC="${1:-$ROOT/env.yaml}"
OUT="${2:-$ROOT/.env}"

if [[ ! -f "$SRC" ]]; then
  echo "ERROR: 未找到 $SRC"
  echo "请确认旧 env.yaml 路径是否正确；新部署流程已改为直接维护 .env。"
  exit 1
fi

echo "==> 解析 $SRC → $OUT ..."
awk -v out="$OUT" '
function trim(t) { gsub(/^[ \t]+|[ \t]+$/,"",t); return t }
function strip_inline_comment(t) {
  sub(/[ \t]#.*$/,"",t)
  return trim(t)
}
function unquote(t) {
  t = strip_inline_comment(t)
  t = trim(t)
  if (length(t) >= 2 && substr(t,1,1) == "\"" && substr(t,length(t),1) == "\"")
    return substr(t, 2, length(t)-2)
  return t
}
function esc_env(s, r) {
  r = s
  gsub(/\\/, "\\\\", r)
  gsub(/"/, "\\\"", r)
  return "\"" r "\""
}
/^site:/ { sec="site"; next }
/^secrets:/ { sec="secrets"; next }
/^smtp:/ { sec="smtp"; next }
/^image:/ { sec="image"; next }
/^[a-z]/ { sec=""; next }
sec=="site" && $0 ~ /^[ \t]+canonical_base_url:/ {
  sub(/^[ \t]*canonical_base_url:[ \t]*/, ""); canon=unquote($0); next
}
sec=="secrets" && $0 ~ /^[ \t]+mysql_root_password:/ {
  sub(/^[ \t]*mysql_root_password:[ \t]*/, ""); mp=unquote($0); next
}
sec=="secrets" && $0 ~ /^[ \t]+jwt_secret:/ {
  sub(/^[ \t]*jwt_secret:[ \t]*/, ""); jwt=unquote($0); next
}
sec=="secrets" && $0 ~ /^[ \t]+control_encryption_key:/ {
  sub(/^[ \t]*control_encryption_key:[ \t]*/, ""); ctrl=unquote($0); next
}
sec=="secrets" && $0 ~ /^[ \t]+minio_root_user:/ {
  sub(/^[ \t]*minio_root_user:[ \t]*/, ""); mu=unquote($0); next
}
sec=="secrets" && $0 ~ /^[ \t]+minio_root_password:/ {
  sub(/^[ \t]*minio_root_password:[ \t]*/, ""); mpw=unquote($0); next
}
sec=="smtp" && $0 ~ /^[ \t]+host:/ {
  sub(/^[ \t]*host:[ \t]*/, ""); smtp_host=unquote($0); next
}
sec=="smtp" && $0 ~ /^[ \t]+port:/ {
  sub(/^[ \t]*port:[ \t]*/, ""); smtp_port=unquote($0); next
}
sec=="smtp" && $0 ~ /^[ \t]+username:/ {
  sub(/^[ \t]*username:[ \t]*/, ""); smtp_user=unquote($0); next
}
sec=="smtp" && $0 ~ /^[ \t]+password:/ {
  sub(/^[ \t]*password:[ \t]*/, ""); smtp_pass=unquote($0); next
}
sec=="smtp" && $0 ~ /^[ \t]+from:/ {
  sub(/^[ \t]*from:[ \t]*/, ""); smtp_from=unquote($0); next
}
sec=="smtp" && $0 ~ /^[ \t]+from_name:/ {
  sub(/^[ \t]*from_name:[ \t]*/, ""); smtp_from_name=unquote($0); next
}
sec=="image" && $0 ~ /^[ \t]+main:/ {
  sub(/^[ \t]*main:[ \t]*/, ""); mi=unquote($0); next
}
END {
  errs=0
  if (canon == "")  { print "ERROR: site.canonical_base_url 必填" > "/dev/stderr"; errs++ }
  if (mp == "")     { print "ERROR: secrets.mysql_root_password 必填" > "/dev/stderr"; errs++ }
  if (jwt == "")    { print "ERROR: secrets.jwt_secret 必填" > "/dev/stderr"; errs++ }
  if (ctrl == "")   { print "ERROR: secrets.control_encryption_key 必填" > "/dev/stderr"; errs++ }
  if (mu == "")     { print "ERROR: secrets.minio_root_user 必填" > "/dev/stderr"; errs++ }
  if (mpw == "")    { print "ERROR: secrets.minio_root_password 必填" > "/dev/stderr"; errs++ }
  if (mi == "")     { print "ERROR: image.main 必填" > "/dev/stderr"; errs++ }
  if (errs) exit 1
  if (smtp_host == "")      smtp_host = "smtp.qq.com"
  if (smtp_port == "")      smtp_port = "587"
  if (smtp_from_name == "") smtp_from_name = "AI Agent OS"
  print "CANONICAL_BASE_URL="  esc_env(canon)          > out
  print "MYSQL_ROOT_PASSWORD=" esc_env(mp)             > out
  print "JWT_SECRET="          esc_env(jwt)            > out
  print "CONTROL_ENC_KEY="     esc_env(ctrl)           > out
  print "MINIO_ROOT_USER="     esc_env(mu)             > out
  print "MINIO_ROOT_PASSWORD=" esc_env(mpw)            > out
  print "SMTP_HOST="           esc_env(smtp_host)      > out
  print "SMTP_PORT="           esc_env(smtp_port)      > out
  print "SMTP_USERNAME="       esc_env(smtp_user)      > out
  print "SMTP_PASSWORD="       esc_env(smtp_pass)      > out
  print "SMTP_FROM="           esc_env(smtp_from)      > out
  print "SMTP_FROM_NAME="      esc_env(smtp_from_name) > out
  print "MAIN_IMAGE="          esc_env(mi)             > out
  close(out)
}
' "$SRC"

echo "    .env 已生成"
