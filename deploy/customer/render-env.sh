#!/usr/bin/env bash
# 从手写 env.yaml 生成 Compose 用的 .env（纯 awk，无 Python）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
SRC="${1:-$ROOT/env.yaml}"
OUT="${2:-$ROOT/.env}"

if [[ ! -f "$SRC" ]]; then
  echo "ERROR: 未找到 $SRC"
  echo "请复制 env.yaml.example 为 env.yaml ，按示例结构手写填写后再执行本脚本。"
  exit 1
fi

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
/^ports:/ { sec="ports"; next }
/^image:/ { sec="image"; next }
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
sec=="secrets" && $0 ~ /^[ \t]+smtp_password:/ {
  sub(/^[ \t]*smtp_password:[ \t]*/, ""); smtp=unquote($0); next
}
sec=="ports" && $0 ~ /^[ \t]+http_publish:/ {
  sub(/^[ \t]*http_publish:[ \t]*/, ""); hp=unquote($0); next
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
  if (mu == "")    { print "ERROR: secrets.minio_root_user 必填" > "/dev/stderr"; errs++ }
  if (mpw == "")   { print "ERROR: secrets.minio_root_password 必填" > "/dev/stderr"; errs++ }
  if (hp == "")    { print "ERROR: ports.http_publish 必填" > "/dev/stderr"; errs++ }
  if (mi == "")    { print "ERROR: image.main 必填" > "/dev/stderr"; errs++ }
  if (errs) exit 1
  print "CANONICAL_BASE_URL="  esc_env(canon) > out
  print "MYSQL_ROOT_PASSWORD="  esc_env(mp)   > out
  print "JWT_SECRET="           esc_env(jwt)  > out
  print "CONTROL_ENC_KEY="      esc_env(ctrl) > out
  print "MINIO_ROOT_USER="      esc_env(mu)   > out
  print "MINIO_ROOT_PASSWORD="  esc_env(mpw)  > out
  print "SMTP_PASSWORD="        esc_env(smtp) > out
  print "HTTP_PUBLISH_PORT="    esc_env(hp)   > out
  print "MAIN_IMAGE="           esc_env(mi)   > out
  close(out)
}
' "$SRC"

printf '%s\n' "已生成 ${OUT}（由 ${SRC} 生成，请勿手改；改配置请编辑 env.yaml 后重新执行本脚本）"
