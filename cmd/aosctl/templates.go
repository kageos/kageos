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

networks:
  aos:
    name: kageos-customer-aos
`

const envTemplate = `
CANONICAL_BASE_URL={{ .Site.BaseURL }}
TLS_MODE={{ .Site.TLSMode }}
TLS_CERT_FILE={{ .Site.CertFile }}
TLS_KEY_FILE={{ .Site.KeyFile }}
KAGEOS_TLS_CERT_PEM_B64={{ .Site.TLSCertPEMB64 }}
KAGEOS_TLS_KEY_PEM_B64={{ .Site.TLSKeyPEMB64 }}
MAIN_IMAGE={{ .Images.Main }}
APP_BASE_IMAGE={{ .Images.AppBase }}
MYSQL_MODE={{ .MySQL.Mode }}
MYSQL_HOST={{ .MySQLHostForMain }}
MYSQL_PORT={{ .MySQLPortForMain }}
MYSQL_ROOT_PASSWORD={{ .MySQL.Password }}
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
MINIO_ROOT_PASSWORD={{ .MinIO.SecretKey }}
JWT_SECRET={{ .Secrets.JWTSecret }}
SYSTEM_USER_PASSWORD={{ .SystemUser.Password }}
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
