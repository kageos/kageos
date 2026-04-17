cmd_init() {
  local force=0
  if [[ -n "${ARG1:-}" ]]; then
    if [[ "$ARG1" == "--force" ]]; then
      force=1
    else
      echo "ERROR: init 仅支持可选参数 --force"
      exit 1
    fi
  fi

  if [[ ! -f "$ENV_FILE" || "$force" == "1" ]]; then
    echo "==> 初始化 .env ..."
    create_env_from_example "$force"
  else
    echo "==> .env 已存在，仅补齐空值 ..."
  fi

  ensure_env_value MYSQL_ROOT_PASSWORD "$(generate_random_hex 32)"
  ensure_env_value JWT_SECRET "$(generate_random_hex 64)"
  ensure_env_value CONTROL_ENC_KEY "$(generate_random_hex 32)"
  ensure_env_value MYSQL_IMAGE "mysql:8.0"
  ensure_env_value NATS_IMAGE "nats:2.10-alpine"
  ensure_env_value MINIO_IMAGE "minio/minio:latest"
  ensure_env_value MINIO_ROOT_USER "minioadmin"
  ensure_env_value MINIO_ROOT_PASSWORD "$(generate_random_hex 32)"
  ensure_env_value BACKUP_BASIC_AUTH_USERNAME "admin"
  ensure_env_value BACKUP_BASIC_AUTH_PASSWORD "$(generate_random_hex 48)"
  ensure_env_value MAIN_IMAGE "agentos-main:latest"
  ensure_env_value APP_BASE_IMAGE "agentos-app-runtime-base:latest"
  validate_env >/dev/null

  echo ""
  echo "=============================="
  echo "  .env 初始化完成"
  echo "=============================="
  echo "  文件: $ENV_FILE"
  echo "  已自动生成: MySQL / JWT / CONTROL_ENC_KEY / MinIO / backup 基础凭据"
  echo "  下一步:"
  echo "    1. 检查并修改 STORAGE_ROOT / CANONICAL_BASE_URL"
  echo "    2. 如需 HTTPS，放入证书并修改 ENABLE_HTTPS"
  echo "    3. 执行: bash build.sh doctor"
  echo "    4. 执行: bash build.sh up"
  echo "=============================="
}

cmd_doctor() {
  local failures=0
  local warnings=0
  local port80 port443 compose_version storage_parent certs_host_dir cert_host_path key_host_path cert_rel key_rel
  local compose_ready=0

  ensure_env_file
  validate_env

  echo "==> 部署预检开始"
  echo "    .env: $ENV_FILE"

  if [[ "$(uname -s)" == "Linux" ]]; then
    doctor_ok "宿主机平台: Linux"
  else
    doctor_fail "prod 当前只支持 Linux 宿主机（依赖 host network + privileged + 容器内 Podman）" || true
    failures=$((failures + 1))
  fi

  if detect_compose_cmd; then
    compose_ready=1
    compose_version="$("${COMPOSE_CMD[@]}" version 2>/dev/null | head -n 1 || true)"
  fi

  if (( compose_ready == 1 )) && [[ -n "$compose_version" ]]; then
    doctor_ok "Compose 可用: ${compose_version}"
  else
    doctor_fail "Compose 不可用，请先安装 podman compose 或 docker compose" || true
    failures=$((failures + 1))
  fi

  if (( compose_ready == 1 )); then
    if compose_run config >/dev/null 2>&1; then
      doctor_ok "docker-compose.yaml 可被当前 .env 正常渲染"
    else
      doctor_fail "docker-compose.yaml 渲染失败，请先修正 .env 或 compose 配置" || true
      failures=$((failures + 1))
    fi
  fi

  storage_parent="$(dirname "$STORAGE_ROOT")"
  if [[ -d "$STORAGE_ROOT" ]]; then
    doctor_ok "STORAGE_ROOT 已存在: $STORAGE_ROOT"
  elif [[ -d "$storage_parent" && -w "$storage_parent" ]]; then
    doctor_ok "STORAGE_ROOT 的父目录可写: $storage_parent"
  else
    doctor_fail "STORAGE_ROOT 不存在且父目录不可写: $STORAGE_ROOT" || true
    failures=$((failures + 1))
  fi

  if (( compose_ready == 1 )) && main_service_running; then
    doctor_ok "main 服务已存在，80/443 端口占用按已部署实例处理"
  else
    port80="$(port_listener_snapshot 80)"
    if [[ -n "$port80" ]]; then
      doctor_fail "80 端口被占用" || true
      echo "$port80"
      failures=$((failures + 1))
    else
      doctor_ok "80 端口可用"
    fi

    if [[ "$ENABLE_HTTPS" == "1" ]]; then
      port443="$(port_listener_snapshot 443)"
      if [[ -n "$port443" ]]; then
        doctor_fail "443 端口被占用" || true
        echo "$port443"
        failures=$((failures + 1))
      else
        doctor_ok "443 端口可用"
      fi
    fi
  fi

  if [[ "$ENABLE_HTTPS" == "1" ]]; then
    certs_host_dir="$(resolve_host_path "$TLS_CERTS_HOST_DIR")"
    cert_rel="${TLS_CERT_FILE#/app/tls/}"
    key_rel="${TLS_KEY_FILE#/app/tls/}"
    cert_host_path="${certs_host_dir}/${cert_rel}"
    key_host_path="${certs_host_dir}/${key_rel}"
    if [[ -f "$cert_host_path" && -f "$key_host_path" ]]; then
      doctor_ok "HTTPS 证书存在: $cert_host_path / $key_host_path"
    else
      doctor_fail "HTTPS 已启用，但证书文件不存在" || true
      failures=$((failures + 1))
    fi
  else
    doctor_ok "当前为 HTTP / 外部 TLS 终止模式"
  fi

  if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet nginx 2>/dev/null; then
    if command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
      doctor_warn "宿主机 nginx 正在运行；build.sh up 会自动停止它"
    else
      doctor_warn "宿主机 nginx 正在运行；build.sh up 需要 sudo 停止它"
    fi
    warnings=$((warnings + 1))
  else
    doctor_ok "未检测到宿主机 nginx 冲突"
  fi

  if [[ "$APP_BASE_IMAGE" == "agentos-app-runtime-base:latest" ]]; then
    doctor_warn "APP_BASE_IMAGE 仍是默认 tag: ${APP_BASE_IMAGE}"
    warnings=$((warnings + 1))
  else
    doctor_ok "APP_BASE_IMAGE 已显式指定: ${APP_BASE_IMAGE}"
  fi

  if [[ "${MINIO_IMAGE:-minio/minio:latest}" == *:latest ]]; then
    doctor_warn "MINIO_IMAGE 仍使用 latest，正式环境建议固定版本 tag"
    warnings=$((warnings + 1))
  else
    doctor_ok "MINIO_IMAGE 已固定版本: ${MINIO_IMAGE}"
  fi

  echo ""
  if (( failures > 0 )); then
    echo "==> 预检失败: ${failures} 个阻断项，${warnings} 个警告"
    return 1
  fi

  echo "==> 预检通过: 0 个阻断项，${warnings} 个警告"
  echo "==> 可以继续执行: bash build.sh up"
}

cmd_up() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  ensure_required_ports_available_for_first_up
  echo "==> 全量启动并构建..."
  compose_run up -d --build
  wait_for_stack_health
  print_success
}

pull_main_image() {
  local engine="${COMPOSE_CMD[0]}"
  echo "==> 拉取主镜像: ${MAIN_IMAGE}"
  "$engine" pull "${MAIN_IMAGE}"
}

cmd_up_image() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  ensure_required_ports_available_for_first_up
  pull_main_image
  echo "==> 基于已发布镜像启动（不在目标机本地构建）..."
  compose_run up -d --no-build
  wait_for_stack_health
  print_success
}

cmd_update() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  echo "==> 仅重建并更新 main / scheduler / backup 服务（不重启中间件）..."
  compose_run up -d --build --no-deps main scheduler backup
  wait_for_stack_health
  print_success
}

cmd_update_image() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  pull_main_image
  echo "==> 仅拉取镜像并更新 main / scheduler / backup 服务（不在目标机本地构建）..."
  compose_run up -d --no-build --no-deps main scheduler backup
  wait_for_stack_health
  print_success
}

cmd_pull_update() {
  validate_env
  echo "==> 执行 git pull --ff-only ..."
  git -C "$PROJECT_ROOT" pull --ff-only
  cmd_update
}

cmd_restart_main() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  echo "==> 重启 main 服务..."
  compose_run restart main
  wait_for_main_health
  print_success
}

cmd_restart_scheduler() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  prepare_storage_layout
  echo "==> 重启 scheduler 服务..."
  compose_run restart scheduler
  wait_for_scheduler_health
  print_success
}

cmd_build_app_base() {
  validate_env
  ensure_compose_cmd

  local cache_flag=""
  if [[ -n "${ARG1:-}" ]]; then
    if [[ "$ARG1" == "--no-cache" ]]; then
      cache_flag="--no-cache"
    else
      echo "ERROR: build-app-base 仅支持可选参数 --no-cache"
      exit 1
    fi
  fi

  if ! compose_run exec main bash -lc 'true' >/dev/null 2>&1; then
    echo "ERROR: main 服务未运行，无法进入容器单独构建 ${APP_BASE_IMAGE}"
    echo "请先执行: bash build.sh up"
    exit 1
  fi

  echo "==> 在 main 容器内单独构建 ${APP_BASE_IMAGE} ..."
  echo "==> 命令: podman build ${cache_flag} -t ${APP_BASE_IMAGE} /app/app-base"
  compose_run exec main bash -lc "set -euo pipefail; podman build ${cache_flag} -t '${APP_BASE_IMAGE}' /app/app-base"
}

cmd_logs() {
  validate_env
  local service="${ARG1:-main}"
  echo "==> 查看日志: $service"
  compose_run logs -f "$service"
}

cmd_status() {
  validate_env
  echo "==> 服务状态"
  compose_run ps
}

cmd_verify() {
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  wait_for_stack_health
}

cmd_down() {
  validate_env
  echo "==> 停止服务（保留数据卷）..."
  compose_run down
}

dispatch_command() {
  local command="$1"
  case "$command" in
    up)
      cmd_up
      ;;
    up-image)
      cmd_up_image
      ;;
    init)
      cmd_init
      ;;
    doctor)
      cmd_doctor
      ;;
    update)
      cmd_update
      ;;
    update-image)
      cmd_update_image
      ;;
    pull-update)
      cmd_pull_update
      ;;
    restart-main)
      cmd_restart_main
      ;;
    restart-scheduler)
      cmd_restart_scheduler
      ;;
    build-app-base)
      cmd_build_app_base
      ;;
    logs)
      cmd_logs
      ;;
    status|ps)
      cmd_status
      ;;
    verify)
      cmd_verify
      ;;
    down)
      cmd_down
      ;;
    help|-h|--help)
      usage
      ;;
    *)
      echo "ERROR: 未知命令: $command"
      echo ""
      usage
      exit 1
      ;;
  esac
}
