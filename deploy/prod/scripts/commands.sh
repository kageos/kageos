parse_init_options() {
  INIT_FORCE=0
  INIT_USE_IMAGE="${1:-0}"
  local arg

  for arg in "${COMMAND_ARGS[@]}"; do
    case "$arg" in
      --force)
        INIT_FORCE=1
        ;;
      --image)
        INIT_USE_IMAGE=1
        ;;
      *)
        echo "ERROR: init 仅支持可选参数 --image / --force"
        exit 1
        ;;
    esac
  done
}

parse_update_options() {
  UPDATE_USE_IMAGE="${1:-0}"
  local arg

  for arg in "${COMMAND_ARGS[@]}"; do
    case "$arg" in
      --image)
        UPDATE_USE_IMAGE=1
        ;;
      *)
        echo "ERROR: update 仅支持可选参数 --image"
        exit 1
        ;;
    esac
  done
}

prepare_init_context() {
  local force="${1:-0}"

  require_linux_host

  if [[ ! -f "$ENV_FILE" || "$force" == "1" ]]; then
    echo "==> 初始化 .env ..."
    create_env_from_example "$force"
  else
    echo "==> .env 已存在，仅补齐空值 ..."
  fi

  bootstrap_env_defaults
  load_env_defaults
  ensure_compose_cmd

  if ! compose_run config >/dev/null 2>&1; then
    echo "ERROR: docker-compose.yaml 渲染失败，请先修正 .env 或 compose 配置"
    exit 1
  fi

  print_storage_mode
  prepare_storage_layout
}

pull_infra_images() {
  echo "==> 预拉取中间件镜像（mysql / nats / minio）..."
  compose_run pull mysql nats minio
}

build_main_image() {
  echo "==> 构建主镜像: ${MAIN_IMAGE}"
  compose_run build main
}

main_image_available() {
  local engine="${COMPOSE_CMD[0]}"
  if [[ "$engine" == "podman" ]]; then
    podman image exists "${MAIN_IMAGE}" >/dev/null 2>&1
  else
    docker image inspect "${MAIN_IMAGE}" >/dev/null 2>&1
  fi
}

run_app_base_tool() {
  local action="$1"
  local no_cache="${2:-0}"

  compose_run run --rm --no-deps \
    -e APP_BASE_ACTION="$action" \
    -e APP_BASE_BUILD_NO_CACHE="$no_cache" \
    --entrypoint /app/entrypoint-app-base.sh \
    main
}

print_init_success() {
  local main_mode="$1"
  local canonical_hint="填写 CANONICAL_BASE_URL"

  if [[ -n "$CANONICAL_BASE_URL" ]]; then
    canonical_hint="检查 CANONICAL_BASE_URL / TLS_MODE / 证书是否正确"
  fi

  echo ""
  echo "=============================="
  echo "  初始化完成"
  echo "=============================="
  echo "  文件: $ENV_FILE"
  echo "  已准备: mysql / nats / minio / ${main_mode} / ${APP_BASE_IMAGE}"
  echo "  下一步:"
  echo "    1. ${canonical_hint}"
  echo "    2. 执行: bash build.sh up"
  echo "    3. 如需显式预检，可先执行: bash build.sh doctor"
  echo "=============================="
}

cmd_init() {
  parse_init_options 0
  install_host_podman
  prepare_init_context "$INIT_FORCE"
  pull_infra_images
  if [[ "$INIT_USE_IMAGE" == "1" ]]; then
    pull_main_image
  else
    build_main_image
  fi

  echo "==> 准备用户应用基础镜像: ${APP_BASE_IMAGE}"
  if ! run_app_base_tool ensure 0; then
    echo "ERROR: 用户应用基础镜像初始化失败，请检查 MAIN_IMAGE 是否已准备好以及宿主机是否支持容器内 Podman。"
    exit 1
  fi

  print_init_success "$MAIN_IMAGE"
}

cmd_init_image() {
  echo "WARN: init-image 已兼容为 init --image；建议改用: bash build.sh init --image"
  COMMAND_ARGS=("${COMMAND_ARGS[@]}" "--image")
  cmd_init
}

cmd_doctor() {
  local failures=0
  local warnings=0
  local port80 port443 compose_version storage_parent certs_host_dir cert_host_path key_host_path cert_rel key_rel
  local compose_ready=0

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

  if host_podman_ready; then
    doctor_ok "宿主机 podman / podman compose 已就绪"
  else
    doctor_warn "宿主机 podman / podman compose 未就绪；首次执行 bash build.sh init 会自动尝试安装"
    warnings=$((warnings + 1))
  fi

  if (( compose_ready == 1 )); then
    if compose_run config >/dev/null 2>&1; then
      doctor_ok "docker-compose.yaml 可被当前 .env 正常渲染"
    else
      doctor_fail "docker-compose.yaml 渲染失败，请先修正 .env 或 compose 配置" || true
      failures=$((failures + 1))
    fi
  fi

  storage_parent="$(dirname "$FIXED_STORAGE_ROOT")"
  if [[ -d "$FIXED_STORAGE_ROOT" ]]; then
    doctor_ok "固定存储目录已存在: $FIXED_STORAGE_ROOT"
  elif [[ -d "$storage_parent" && -w "$storage_parent" ]]; then
    doctor_ok "固定存储目录的父目录可写: $storage_parent"
  else
    doctor_fail "固定存储目录不存在且父目录不可写: $FIXED_STORAGE_ROOT" || true
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

    if tls_mode_uses_local_https "$TLS_MODE"; then
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

  if tls_mode_uses_local_https "$TLS_MODE"; then
    certs_host_dir="$(resolve_host_path "$TLS_CERTS_HOST_DIR")"
    cert_rel="${TLS_CERT_FILE#/app/tls/}"
    key_rel="${TLS_KEY_FILE#/app/tls/}"
    cert_host_path="${certs_host_dir}/${cert_rel}"
    key_host_path="${certs_host_dir}/${key_rel}"
    if [[ -f "$cert_host_path" && -f "$key_host_path" ]]; then
      doctor_ok "TLS_MODE=${TLS_MODE} 的证书存在: $cert_host_path / $key_host_path"
    else
      doctor_fail "TLS_MODE=${TLS_MODE} 需要本地证书，但证书文件不存在" || true
      failures=$((failures + 1))
    fi
  elif [[ "$TLS_MODE" == "external" ]]; then
    doctor_ok "当前为 external 模式：容器仅提供 HTTP，由外部入口终止 TLS"
  else
    doctor_ok "当前为 http 模式：容器仅提供 HTTP"
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

  if (( compose_ready == 1 )); then
    if main_image_available; then
      doctor_ok "主镜像已就绪: ${MAIN_IMAGE}"
      if run_app_base_tool check 0 >/dev/null 2>&1; then
        doctor_ok "用户应用基础镜像已就绪: ${APP_BASE_IMAGE}"
      else
        doctor_fail "用户应用基础镜像未初始化，请先执行 ${INIT_IMAGE_USAGE_HINT}" || true
        failures=$((failures + 1))
      fi
    else
      doctor_fail "主镜像未初始化，请先执行 ${INIT_IMAGE_USAGE_HINT}" || true
      failures=$((failures + 1))
    fi
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
  if ! main_image_available; then
    echo "ERROR: 未找到主镜像 ${MAIN_IMAGE}，请先执行 ${INIT_IMAGE_USAGE_HINT}"
    exit 1
  fi
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  prepare_storage_layout
  if ! run_app_base_tool check 0 >/dev/null 2>&1; then
    echo "ERROR: 未找到用户应用基础镜像 ${APP_BASE_IMAGE}，请先执行 ${INIT_IMAGE_USAGE_HINT}"
    exit 1
  fi
  stop_host_nginx_if_needed
  ensure_required_ports_available_for_first_up
  echo "==> 启动已初始化服务（不再构建镜像）..."
  compose_run up -d --no-build
  wait_for_stack_health
  print_success
}

pull_main_image() {
  local engine="${COMPOSE_CMD[0]}"
  echo "==> 拉取主镜像: ${MAIN_IMAGE}"
  "$engine" pull "${MAIN_IMAGE}"
}

cmd_up_image() {
  echo "WARN: up-image 已兼容为 up；建议改用: bash build.sh up"
  cmd_up
}

cmd_update() {
  parse_update_options 0
  validate_env
  ensure_compose_cmd
  echo "==> 使用: ${COMPOSE_CMD[*]}"
  print_storage_mode
  stop_host_nginx_if_needed
  prepare_storage_layout
  if [[ "$UPDATE_USE_IMAGE" == "1" ]]; then
    pull_main_image
    echo "==> 仅拉取镜像并更新 main / scheduler / backup 服务（不在目标机本地构建）..."
    compose_run up -d --no-build --no-deps main scheduler backup
  else
    echo "==> 仅重建并更新 main / scheduler / backup 服务（不重启中间件）..."
    compose_run up -d --build --no-deps main scheduler backup
  fi
  wait_for_stack_health
  print_success
}

cmd_update_image() {
  echo "WARN: update-image 已兼容为 update --image；建议改用: bash build.sh update --image"
  COMMAND_ARGS=("${COMMAND_ARGS[@]}" "--image")
  cmd_update
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
  require_linux_host
  ensure_env_bootstrapped
  load_env_defaults
  ensure_compose_cmd
  print_storage_mode
  prepare_storage_layout

  local no_cache=0
  if [[ -n "${ARG1:-}" ]]; then
    if [[ "$ARG1" == "--no-cache" ]]; then
      no_cache=1
    else
      echo "ERROR: build-app-base 仅支持可选参数 --no-cache"
      exit 1
    fi
  fi

  if ! compose_run config >/dev/null 2>&1; then
    echo "ERROR: docker-compose.yaml 渲染失败，请先修正 .env 或 compose 配置"
    exit 1
  fi

  if ! main_image_available; then
    echo "ERROR: 未找到主镜像 ${MAIN_IMAGE}，请先执行 ${INIT_IMAGE_USAGE_HINT}"
    exit 1
  fi

  echo "==> 在与 main 相同的运行环境里重建 ${APP_BASE_IMAGE} ..."
  if [[ "$no_cache" == "1" ]]; then
    echo "==> 已启用 --no-cache：本次不复用 layer 缓存"
  fi

  if ! run_app_base_tool rebuild "$no_cache"; then
    echo "ERROR: 用户应用基础镜像重建失败；请先执行 ${INIT_IMAGE_USAGE_HINT}，确保 MAIN_IMAGE 已准备好。"
    exit 1
  fi
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
    init-image)
      cmd_init_image
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
