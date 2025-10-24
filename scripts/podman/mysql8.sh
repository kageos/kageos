#!/usr/bin/env bash

# 在本机通过 Podman 启动 MySQL 8（持久化数据，暴露 3306）
# 使用前先确保已创建数据卷：podman volume create mysql-data

podman run -d --name mysql8 --restart=always \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=app_db \
  -e MYSQL_USER=app \
  -e MYSQL_PASSWORD=app \
  -v mysql-data:/var/lib/mysql \
  -p 3306:3306 \
  docker.io/library/mysql:8.0





