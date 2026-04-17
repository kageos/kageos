print_success() {
  local transport_mode="HTTP"
  case "${TLS_MODE:-$DEFAULT_TLS_MODE}" in
    https)
      transport_mode="HTTP + HTTPS"
      ;;
    redirect)
      transport_mode="HTTPS（80 -> 443 重定向）"
      ;;
    external)
      transport_mode="HTTP（外部 TLS 终止）"
      ;;
  esac

  echo ""
  echo "=============================="
  echo "  操作完成"
  echo "=============================="
  echo "  访问地址: ${CANONICAL_BASE_URL}"
  echo "  存储根目录: ${FIXED_STORAGE_ROOT}"
  echo "  传输模式: ${transport_mode}"
  echo ""
  echo "  查看日志: bash build.sh logs main"
  echo "  查看状态: bash build.sh status"
  echo "  健康检查: bash build.sh verify"
  echo "  停止服务: bash build.sh down"
  echo "  ⚠ 切勿:  rm -rf ${FIXED_STORAGE_ROOT}"
  echo "=============================="
}

wait_for_exec_health() {
  local service="$1"
  local shell_name="$2"
  local probe="$3"
  local attempts="$4"
  local interval="${5:-5}"

  local i
  for i in $(seq 1 "$attempts"); do
    if compose_run exec -T "$service" "$shell_name" -lc "$probe" >/dev/null 2>&1; then
      echo "==> ${service} 健康检查通过"
      return 0
    fi
    echo "    等待 ${service} 健康检查通过... ($i/$attempts)"
    sleep "$interval"
  done

  echo "ERROR: ${service} 健康检查失败"
  compose_run ps || true
  return 1
}

wait_for_stack_health() {
  echo "==> 执行生产健康检查..."
  wait_for_exec_health mysql sh 'mysqladmin ping -h 127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" --silent' 24 5
  wait_for_exec_health main bash '/app/health/main.sh' 120 5
  wait_for_exec_health scheduler bash '/app/health/scheduler.sh' 24 5
  wait_for_exec_health backup bash '/app/health/backup.sh' 24 5
}

wait_for_main_health() {
  echo "==> 检查 main 健康状态..."
  wait_for_exec_health main bash '/app/health/main.sh' 120 5
}

wait_for_scheduler_health() {
  echo "==> 检查 scheduler 健康状态..."
  wait_for_exec_health scheduler bash '/app/health/scheduler.sh' 24 5
}
