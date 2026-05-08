root /var/www/web/dist;
index index.html;
error_page 503 /__backup_maintenance__.html;

set $backup_maintenance 0;
if (-f /app/data/backup/state/maintenance.flag) {
    set $backup_maintenance 1;
}
if ($uri = /__backup_maintenance__.html) {
    set $backup_maintenance 0;
}
if ($backup_maintenance = 1) {
    return 503;
}

gzip on;
gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
gzip_min_length 256;

location = /__backup_maintenance__.html {
    internal;
    alias /app/data/backup/state/maintenance.html;
    add_header Cache-Control "no-store" always;
}

location /assets/ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}

location /workspace/api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
    client_max_body_size 4096m;
}

location /agent/api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}

location /storage/api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_set_header Host $http_host;
    proxy_set_header X-Forwarded-Port $server_port;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
    client_max_body_size 4096m;
}

location /control/api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}

location /message/api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}

location /hr/api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}

location /api/ {
    proxy_pass http://127.0.0.1:9090;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}

location /ai-agent-os/ {
    proxy_pass http://127.0.0.1:9000;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    client_max_body_size 4096m;
    proxy_buffering off;
    proxy_request_buffering off;
}

location / {
    try_files $uri $uri/ /index.html;
    error_page 405 =200 /index.html;
}
