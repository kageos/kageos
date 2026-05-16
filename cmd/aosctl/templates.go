package main

const composeTemplate = `
services:
{{- if .IncludeMySQL }}
  mysql:
    image: {{ q .Images.MySQL }}
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: {{ q .MySQL.Password }}
      TZ: Asia/Shanghai
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
      TZ: Asia/Shanghai
    command: server /data --console-address ":9001"
    ports:
      - "127.0.0.1:9000:9000"
    volumes:
      - {{ .Storage.Root }}/minio:/data
    networks: [aos]
{{- end }}

  main:
    build:
      context: ../../..
      dockerfile: deploy/prod/Dockerfile
    image: {{ q .Images.Main }}
    network_mode: host
    privileged: true
    restart: unless-stopped
    environment:
      CANONICAL_BASE_URL: {{ q .Site.BaseURL }}
      MYSQL_HOST: {{ q .MySQLHostForMain }}
      MYSQL_PORT: {{ q .MySQLPortForMain }}
      MYSQL_ROOT_PASSWORD: {{ q .MySQL.BackupAdminPass }}
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
      CONTROL_ENC_KEY: {{ q .Secrets.ControlEncKey }}
      SYSTEM_USER_PASSWORD: {{ q .SystemUser.Password }}
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
      TLS_CERT_FILE: {{ q .Site.CertFile }}
      TLS_KEY_FILE: {{ q .Site.KeyFile }}
      APP_BASE_IMAGE: {{ q .Images.AppBase }}
    volumes:
      - {{ .Storage.Root }}/podman_storage:/var/lib/containers
      - {{ .Storage.Root }}/logs:/app/logs
      - {{ .Storage.Root }}/namespace:/app/namespace
      - {{ .Storage.Root }}/data:/app/data
      - ./config:/app/config.prod.template:ro
      - {{ .TLSCertsHostDir }}:/app/tls:ro
    healthcheck:
      test: ["CMD", "/app/health/main.sh"]
      interval: 20s
      timeout: 10s
      retries: 12
      start_period: 90s

  backup:
    image: {{ q .Images.Main }}
    entrypoint: ["/app/entrypoint-backup.sh"]
    restart: unless-stopped
    environment:
      MYSQL_HOST: {{ q .BackupMySQLHost }}
      MYSQL_PORT: {{ q .BackupMySQLPort }}
      MYSQL_ROOT_PASSWORD: {{ q .MySQL.BackupAdminPass }}
      MINIO_HOST: {{ q .BackupMinIOHost }}
      MINIO_PORT: {{ q .BackupMinIOPort }}
      MINIO_ROOT_PASSWORD: {{ q .MinIO.SecretKey }}
      BACKUP_BASIC_AUTH_PASSWORD: {{ q .Secrets.BackupBasicAuthPass }}
    ports:
      - "127.0.0.1:19088:19088"
    volumes:
      - {{ .Storage.Root }}/logs:/app/logs
      - {{ .Storage.Root }}/namespace:/app/namespace:ro
      - {{ .Storage.Root }}/data:/app/data
      - {{ .Storage.Root }}/mysql:/storage/mysql:ro
      - {{ .Storage.Root }}/minio:/storage/minio:ro
      - {{ .Storage.Root }}/podman_storage:/storage/podman_storage:ro
      - ./config:/app/config.prod.template:ro
    healthcheck:
      test: ["CMD", "/app/health/backup.sh"]
      interval: 20s
      timeout: 10s
      retries: 6
      start_period: 30s
    networks: [aos]

networks:
  aos:
    name: ai-agent-os-customer-aos
`

const envTemplate = `
CANONICAL_BASE_URL={{ .Site.BaseURL }}
TLS_MODE={{ .Site.TLSMode }}
MAIN_IMAGE={{ .Images.Main }}
APP_BASE_IMAGE={{ .Images.AppBase }}
MYSQL_MODE={{ .MySQL.Mode }}
MYSQL_HOST={{ .MySQLHostForMain }}
MYSQL_PORT={{ .MySQLPortForMain }}
NATS_MODE={{ .NATS.Mode }}
NATS_URL={{ .NATSURL }}
MINIO_MODE={{ .MinIO.Mode }}
MINIO_ENDPOINT={{ .MinIOEndpoint }}
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
CREATE DATABASE IF NOT EXISTS {{ mysqlIdent .MySQL.HRDatabase }} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
`

const globalConfigTemplate = `
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
  issuer: "ai-agent-os"

control_service:
  enabled: true
  encryption_key: {{ q .Secrets.ControlEncKey }}
  key_path: "/app/data/license/license.key"

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
  - path: "/control"
    service_name: "control"
    targets:
      - url: "http://127.0.0.1:9096"
    timeout: 300
  - path: "/message"
    service_name: "message"
    targets:
      - url: "http://127.0.0.1:9109"
    timeout: 300
  - path: "/workspace"
    service_name: "workspace"
    targets:
      - url: "http://127.0.0.1:9091"
    timeout: 300
  - path: "/hr"
    service_name: "hr"
    targets:
      - url: "http://127.0.0.1:9097"
    timeout: 300

timeouts:
  default: 300
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
  container_cleanup: 10

container:
  timeout: 30
  image:
    base_image: {{ q .Images.AppBase }}
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
      provider: {{ q .Provider }}
      model: {{ q .Model }}
{{- if .APIKey }}
      api_key: {{ q .APIKey }}
{{- end }}
{{- if .APIKeyEnv }}
      api_key_env: {{ q .APIKeyEnv }}
{{- end }}
      api_base: {{ q .APIBase }}
      timeout: {{ .Timeout }}
      max_tokens: {{ .MaxTokens }}
{{- if .ExtraConfig }}
      extra_config: {{ q .ExtraConfig }}
{{- end }}
      use_thinking: {{ .UseThinking }}
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

system_user:
  password: {{ q .SystemUser.Password }}
`

const messageServerConfigTemplate = `
server:
  port: 9109
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  allow_nats_degraded_startup: false

# 当前用于解析收件人用户/部门，指向 hr-server 数据库。
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
`

const controlServiceConfigTemplate = `
server:
  port: 9096
  listen_host: "127.0.0.1"
  log_level: "info"
  debug: false
  enable_pprof: false

license:
  path: "/app/data/license/license.json"
  encryption_key: {{ q .Secrets.ControlEncKey }}
  publish_interval: 300
`

const backupServiceConfigTemplate = `
server:
  port: 19088
  listen_host: {{ q .BackupListenHost }}
  log_level: "info"
  debug: false

storage:
  root: {{ q .Storage.Root }}
  namespace_path: "/app/namespace"
  data_path: "/app/data"
  logs_path: "/app/logs"
  mysql_path: "/storage/mysql"
  minio_path: "/storage/minio"
  podman_storage_path: "/storage/podman_storage"

repository:
  root_path: "/app/data/backup/repo"
  state_path: "/app/data/backup/state"
  staging_path: "/app/data/backup/staging"

database:
  path: "/app/data/backup/state/backup-service.db"

maintenance:
  marker_path: "/app/data/backup/state/maintenance.flag"
  page_path: "/app/data/backup/state/maintenance.html"
  metadata_path: "/app/data/backup/state/maintenance.json"

dependencies:
  mysql_address: {{ q .BackupMySQLAddress }}
  minio_address: {{ q .BackupMinIOAddress }}

mysql:
  user: {{ q .MySQL.BackupAdminUser }}
  password: {{ q .MySQL.BackupAdminPass }}

minio:
  access_key: {{ q .MinIO.AccessKey }}
  secret_key: {{ q .MinIO.SecretKey }}

auth:
  username: {{ q .Secrets.BackupBasicAuthUser }}
  password: {{ q .Secrets.BackupBasicAuthPass }}
  realm: "Backup Control Plane"

tooling:
  mysql_binary: "mysql"
  mysqldump_binary: "mysqldump"
  restic_binary: "restic"
  minio_client_binary: "mc"
`
