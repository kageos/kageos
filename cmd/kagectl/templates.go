package main

const composeTemplate = `
services:
{{- if .IncludeMySQL }}
  mysql:
    image: {{ q .Images.MySQL }}
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: {{ q .MySQL.Password }}
      TZ: {{ q .Timezone }}
    command: >
      --character-set-server=utf8mb4
      --collation-server=utf8mb4_unicode_ci
    ports:
      - "127.0.0.1:3306:3306"
    volumes:
      - {{ .Storage.Root }}/mysql:/var/lib/mysql
      - ./infra/mysql-init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", {{ q (printf "-u%s" .MySQL.User) }}, {{ q (printf "-p%s" .MySQL.Password) }}]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    networks: [aos]
{{- end }}

{{- if .IncludeNATS }}
  nats:
    image: {{ q .Images.NATS }}
    restart: unless-stopped
    ports:
      - "127.0.0.1:4222:4222"
    volumes:
      - ./infra/nats-server.conf:/etc/nats/nats-server.conf:ro
    command: ["-c", "/etc/nats/nats-server.conf"]
    networks: [aos]
{{- end }}

{{- if .IncludeMinIO }}
  minio:
    image: {{ q .Images.MinIO }}
    restart: unless-stopped
    environment:
      MINIO_ROOT_USER: {{ q .MinIO.RootUser }}
      MINIO_ROOT_PASSWORD: {{ q .MinIO.RootPassword }}
      TZ: {{ q .Timezone }}
    command: server /data --console-address ":9001"
    ports:
      - "127.0.0.1:9000:9000"
    volumes:
      - {{ .Storage.Root }}/minio:/data
    networks: [aos]
{{- end }}

  app-base-builder:
    profiles: ["build"]
    build:
      context: ../../..
      dockerfile: deploy/prod/app-base-builder.Dockerfile
    image: {{ q .AppBaseBuilderImage }}
{{- if .UseHostNetwork }}
    network_mode: host
{{- else }}
    networks: [aos]
{{- end }}
    privileged: true
    environment:
      KAGEOS_APP_BASE_IMAGE: {{ q .Images.AppBase }}
      KAGEOS_APP_BASE_ACTION: "ensure"
      KAGEOS_APP_BASE_BUILD_NO_CACHE: "0"
    volumes:
      - {{ .Storage.Root }}/podman_storage:/var/lib/containers

  main:
    build:
      context: ../../..
      dockerfile: deploy/prod/Dockerfile
    image: {{ q .Images.Main }}
{{- if .UseHostNetwork }}
    network_mode: host
{{- else }}
    ports:
      - {{ q (printf "%d:%d" .Site.HTTPPort .Site.HTTPPort) }}
{{- if or (eq .Site.TLSMode "https") (eq .Site.TLSMode "redirect") }}
      - {{ q (printf "%d:%d" .Site.HTTPSPort .Site.HTTPSPort) }}
{{- end }}
    networks: [aos]
{{- end }}
    privileged: true
    restart: unless-stopped
    environment:
      TZ: {{ q .Timezone }}
      CANONICAL_BASE_URL: {{ q .Site.BaseURL }}
      MYSQL_HOST: {{ q .MySQLHostForMain }}
      MYSQL_PORT: {{ q .MySQLPortForMain }}
      MYSQL_ROOT_PASSWORD: {{ q .MySQL.Password }}
      MINIO_HOST: {{ q .MinIOHostForMain }}
      MINIO_PORT: {{ q .MinIOPortForMain }}
      MINIO_ROOT_PASSWORD: {{ q .MinIO.SecretKey }}
      NATS_HOST: {{ q .NATSHostForMain }}
      NATS_PORT: {{ q .NATSPortForMain }}
      NATS_URL: {{ q .NATSURL }}
      NATS_SEED_HOST: {{ q .NATSHostForMain }}
      NATS_SEED_PORT: {{ q .NATSPortForMain }}
      NATS_SEED_USER: {{ q .NATSAuthUser }}
      NATS_SEED_PASSWORD: {{ q .NATSAuthPassword }}
      JWT_SECRET: {{ q .Secrets.JWTSecret }}
      KAGEOS_APP_DB_SECRET_KEY: {{ q .Secrets.AppDBSecret }}
      KAGEOS_APP_DB_CLUSTER_KEY: {{ q .AppDBClusterKey }}
      SYSTEM_USER_PASSWORD: {{ q .SystemUser.Password }}
      KAGEOS_COMPANY_CODE: {{ q .Company.Code }}
      KAGEOS_COMPANY_NAME: {{ q .Company.Name }}
      KAGEOS_COMPANY_LOGO_URL: {{ q .Company.LogoURL }}
      SMTP_MODE: {{ q .SMTP.Mode }}
      SMTP_HOST: {{ q .SMTP.Host }}
      SMTP_PORT: {{ q .SMTP.Port }}
      SMTP_USERNAME: {{ q .SMTP.Username }}
      SMTP_PASSWORD: {{ q .SMTP.Password }}
      SMTP_FROM: {{ q .SMTP.From }}
      SMTP_FROM_NAME: {{ q .SMTP.FromName }}
{{- range .LLMSeedEnvVars }}
      {{ . }}: {{ q (printf "${%s:-}" .) }}
{{- end }}
      TLS_MODE: {{ q .Site.TLSMode }}
      HTTP_PORT: {{ q (printf "%d" .Site.HTTPPort) }}
      HTTPS_PORT: {{ q (printf "%d" .Site.HTTPSPort) }}
      TLS_CERT_FILE: {{ q .Site.CertFile }}
      TLS_KEY_FILE: {{ q .Site.KeyFile }}
      KAGEOS_APP_BASE_IMAGE: {{ q .Images.AppBase }}
    volumes:
      - {{ .Storage.Root }}/podman_storage:/var/lib/containers
      - {{ .Storage.Root }}/logs:/app/logs
      - {{ .Storage.Root }}/namespace:/app/namespace
      - {{ .Storage.Root }}/data:/app/data
      - ./config:/app/config.prod.template:ro
      - {{ .TLSCertsHostDir }}:/app/tls
    healthcheck:
      test: ["CMD", "/app/health/main.sh"]
      interval: 20s
      timeout: 10s
      retries: 12
      start_period: 90s

networks:
  aos:
    name: kageos-customer-aos
`

const envTemplate = `
CANONICAL_BASE_URL={{ .Site.BaseURL }}
TZ={{ .Timezone }}
KAGEOS_NETWORK_PROFILE={{ .Network.Profile }}
TLS_MODE={{ .Site.TLSMode }}
HTTP_PORT={{ .Site.HTTPPort }}
HTTPS_PORT={{ .Site.HTTPSPort }}
TLS_CERT_FILE={{ .Site.CertFile }}
TLS_KEY_FILE={{ .Site.KeyFile }}
KAGEOS_TLS_CERT_PEM_B64={{ .Site.TLSCertPEMB64 }}
KAGEOS_TLS_KEY_PEM_B64={{ .Site.TLSKeyPEMB64 }}
MAIN_IMAGE={{ .Images.Main }}
KAGEOS_APP_BASE_IMAGE={{ .Images.AppBase }}
MYSQL_MODE={{ .MySQL.Mode }}
MYSQL_HOST={{ .MySQLHostForMain }}
MYSQL_PORT={{ .MySQLPortForMain }}
MYSQL_ROOT_PASSWORD={{ .MySQL.Password }}
KAGEOS_APP_DB_SECRET_KEY={{ .Secrets.AppDBSecret }}
KAGEOS_APP_DB_CLUSTER_KEY={{ .AppDBClusterKey }}
NATS_MODE={{ .NATS.Mode }}
NATS_HOST={{ .NATSHostForMain }}
NATS_PORT={{ .NATSPortForMain }}
NATS_URL={{ .NATSURL }}
NATS_SEED_USER={{ .NATSAuthUser }}
NATS_SEED_PASSWORD={{ .NATSAuthPassword }}
MINIO_MODE={{ .MinIO.Mode }}
MINIO_ENDPOINT={{ .MinIOEndpoint }}
MINIO_HOST={{ .MinIOHostForMain }}
MINIO_PORT={{ .MinIOPortForMain }}
MINIO_ROOT_USER={{ .MinIO.RootUser }}
MINIO_ROOT_PASSWORD={{ .MinIO.SecretKey }}
JWT_SECRET={{ .Secrets.JWTSecret }}
SYSTEM_USER_PASSWORD={{ .SystemUser.Password }}
KAGEOS_COMPANY_CODE={{ .Company.Code }}
KAGEOS_COMPANY_NAME={{ q .Company.Name }}
KAGEOS_COMPANY_LOGO_URL={{ q .Company.LogoURL }}
KAGEOS_REGISTRATION_MODE={{ .Auth.RegistrationMode }}
SMTP_MODE={{ .SMTP.Mode }}
SMTP_HOST={{ .SMTP.Host }}
SMTP_PORT={{ .SMTP.Port }}
SMTP_USERNAME={{ .SMTP.Username }}
SMTP_PASSWORD={{ .SMTP.Password }}
SMTP_FROM={{ .SMTP.From }}
SMTP_FROM_NAME={{ .SMTP.FromName }}
`

const natsConfigTemplate = `
max_payload: 10485760
port: 4222
logtime: true
{{- if .NATS.AuthEnabled }}
authorization {
  user: {{ q .NATS.User }}
  password: {{ q .NATS.Password }}
}
{{- end }}
`

const mysqlInitTemplate = `
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.AppDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.StorageDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.AgentDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.ConnectorDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.HRDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.TimerDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.MessageDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
`

const globalConfigTemplate = `
site:
  base_url: {{ q .Site.BaseURL }}

gateway:
  host: "0.0.0.0"
  port: 9090
  base_url: "http://127.0.0.1:9090"
  internal_url: "http://127.0.0.1:9090"

nats:
  url: {{ q .NATSURL }}

jwt:
  secret: {{ q .Secrets.JWTSecret }}
  access_token_expire: 2592000
  refresh_token_expire: 7776000
  issuer: "kageos"

sdk:
  nats_url: {{ q .SDKNATSURL }}
  gateway_url: {{ q .SDKGatewayURL }}
  env_vars:
    FFMPEG_PATH: "/usr/bin/ffmpeg"
    GHOSTSCRIPT_PATH: "/usr/bin/gs"
    PDFTOTEXT_PATH: "/usr/bin/pdftotext"
    PDFINFO_PATH: "/usr/bin/pdfinfo"
    PDFIMAGES_PATH: "/usr/bin/pdfimages"
    PDFTOPPM_PATH: "/usr/bin/pdftoppm"
    GRAPHICSMAGICK_PATH: "/usr/bin/gm"
    LUA_PATH: "/usr/bin/lua"
    PYTHON_PATH: "/usr/bin/python3"
    PIP_PATH: "/usr/bin/pip3"

`

const apiGatewayConfigTemplate = `
server:
  port: 9090
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false
  allow_nats_degraded_startup: false

routes:
  - path: "/storage"
    service_name: "storage"
    targets:
      - url: "http://127.0.0.1:9092"
    timeout: 300
  - path: "/agent"
    service_name: "agent"
    targets:
      - url: "http://127.0.0.1:9095"
    timeout: 300
  - path: "/workspace"
    service_name: "workspace"
    targets:
      - url: "http://127.0.0.1:9091"
    timeout: 300
  - path: "/public/api"
    service_name: "workspace-public"
    targets:
      - url: "http://127.0.0.1:9091"
    timeout: 300
  - path: "/hr"
    service_name: "hr"
    targets:
      - url: "http://127.0.0.1:9097"
    timeout: 300
  - path: "/connector"
    service_name: "connector"
    targets:
      - url: "http://127.0.0.1:9096"
    timeout: 300
  - path: "/timer"
    service_name: "timer"
    targets:
      - url: "http://127.0.0.1:9098"
    timeout: 300
  - path: "/message"
    service_name: "message"
    targets:
      - url: "http://127.0.0.1:9099"
    timeout: 300

timeouts:
  default: 300
`

const timerSchedulerConfigTemplate = `
server:
  port: 9098
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.TimerDatabase }}
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 300
  log_level: "warn"
  slow_threshold: 200

scheduler:
  poll_interval_millis: 1000
  batch_size: 50
  dispatch_lease_seconds: 30
  execution_lease_seconds: 3600
  queue_ack_timeout_seconds: 120
  # Retry every two minutes for about one hour before timing out.
  max_dispatch_attempts: 30
  max_heartbeat_misses: 3
  max_outbox_attempts: 8
  payload_limit_bytes: 262144
`

const messageServerConfigTemplate = `
server:
  port: 9099
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false
  allow_nats_degraded_startup: false

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.MessageDatabase }}
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 300
  log_level: "warn"
  slow_threshold: 200
`

const appRuntimeConfigTemplate = `
runtime:
  port: 9093
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false

timeouts:
  app_server_request: 30
  container_startup: 2
  app_startup_notification: 300
  update_callback: 240
  container_cleanup: 10

app_manage:
  app_dir:
    base_path: {{ q .AppRuntimeBasePath }}
  build:
    keep_debug_symbols: false

container:
  timeout: 30
{{- if .AppContainerNetworkMode }}
  network_mode: {{ q .AppContainerNetworkMode }}
{{- end }}
  image:
    base_image: {{ q .Images.AppBase }}

app_database:
  enabled: true
  dialect: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  admin_user: {{ q .MySQL.User }}
  admin_password: {{ q .MySQL.Password }}
  grant_host: "%"
  secret_key: {{ q .Secrets.AppDBSecret }}
  cluster_key: {{ q .AppDBClusterKey }}
  database_prefix: "kgo_"
  user_prefix: "kgu_"
  max_open_conns: 2
  max_idle_conns: 0
  max_idle_time: 30
  max_lifetime: 600
`

const appServerConfigTemplate = `
server:
  port: 9091
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.AppDatabase }}
  max_idle_conns: 30
  max_open_conns: 200
  max_lifetime: 300
  log_level: "warn"
  slow_threshold: 200

timeouts:
  app_request: 300
  nats_request: 300
`

const appStorageConfigTemplate = `
server:
  port: 9092
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false

audit:
  upload_tracking:
    enabled: true
  download_tracking:
    enabled: true
    retention_days: 90

storage:
  type: "minio"
  minio:
    endpoint: {{ q .MinIOEndpoint }}
    server_endpoint: {{ q .SDKMinIOEndpoint }}
    access_key: {{ q .MinIO.AccessKey }}
    secret_key: {{ q .MinIO.SecretKey }}
    use_ssl: {{ .MinIO.UseSSL }}
    region: {{ q .MinIO.Region }}
    default_bucket: {{ q .MinIO.Bucket }}
  upload:
    max_size: 4294967296
    token_expire: 3600

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.StorageDatabase }}
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 300
  log_level: "info"
  slow_threshold: 200
`

const connectorServerConfigTemplate = `
server:
  port: 9096
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.ConnectorDatabase }}
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 300
  log_level: "warn"
  slow_threshold: 200

oauth:
  callback_base_url: {{ q .Site.BaseURL }}
  state_ttl_seconds: 600
  provider_admins:
    - "system"
  token_encryption_secret: ""
  providers: []
`

const agentServerConfigTemplate = `
server:
  port: 9095
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.AgentDatabase }}
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 300
  log_level: "info"
  slow_threshold: 200
{{- if .LLMs.Configs }}

llms:
  default: {{ q .LLMs.Default }}
  configs:
{{- range .LLMs.Configs }}
    - code: {{ q .Code }}
      name: {{ q .Name }}
{{- if .Provider }}
      provider: {{ q .Provider }}
{{- end }}
{{- if .Protocol }}
      protocol: {{ q .Protocol }}
{{- end }}
      model: {{ q .Model }}
{{- if .APIKey }}
      api_key: {{ q .APIKey }}
{{- end }}
{{- if .APIKeyEnv }}
      api_key_env: {{ q .APIKeyEnv }}
{{- end }}
      api_base: {{ q .APIBase }}
{{- if .EndpointPath }}
      endpoint_path: {{ q .EndpointPath }}
{{- end }}
{{- if .APIVersion }}
      api_version: {{ q .APIVersion }}
{{- end }}
{{- if .AuthScheme }}
      auth_scheme: {{ q .AuthScheme }}
{{- end }}
{{- if .Headers }}
      headers: {{ q .Headers }}
{{- end }}
      timeout: {{ .Timeout }}
      max_tokens: {{ .MaxTokens }}
{{- if .ExtraConfig }}
      extra_config: {{ q .ExtraConfig }}
{{- end }}
{{- if .Capabilities }}
      capabilities: {{ q .Capabilities }}
{{- end }}
{{- if .IsDefault }}
      is_default: true
{{- end }}
      visibility: {{ .Visibility }}
{{- if .Admin }}
      admin: {{ q .Admin }}
{{- end }}
{{- end }}
{{- end }}
`

const hrServerConfigTemplate = `
server:
  port: 9097
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false
  allow_nats_degraded_startup: false

db:
  type: "mysql"
  host: {{ q .MySQLHostForMain }}
  port: {{ .MySQLPortForMain }}
  user: {{ q .MySQL.User }}
  password: {{ q .MySQL.Password }}
  name: {{ q .MySQL.HRDatabase }}
  max_idle_conns: 10
  max_open_conns: 100
  max_lifetime: 300
  log_level: "warn"
  slow_threshold: 200

email:
  mode: {{ q .SMTP.Mode }}
  smtp:
    host: {{ q .SMTP.Host }}
    port: {{ .SMTP.Port }}
    username: {{ q .SMTP.Username }}
    password: {{ q .SMTP.Password }}
    from: {{ q .SMTP.From }}
    from_name: {{ q .SMTP.FromName }}
  verification:
    code_length: 6
    code_expire: 300

auth:
  registration_mode: {{ q .Auth.RegistrationMode }}

company:
  code: {{ q .Company.Code }}
  name: {{ q .Company.Name }}
  logo_url: {{ q .Company.LogoURL }}

system_user:
  password: {{ q .SystemUser.Password }}
`
