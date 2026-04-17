#!/usr/bin/env bash
set -euo pipefail

APT_USE_MIRROR="${APT_USE_MIRROR:-1}"

if [[ "${APT_USE_MIRROR}" != "1" ]]; then
  exit 0
fi

if [[ -f /etc/apt/sources.list.d/debian.sources ]]; then
  sed -i \
    -e 's|http://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' \
    -e 's|https://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' \
    -e 's|http://security.debian.org/debian-security|http://mirrors.aliyun.com/debian-security|g' \
    -e 's|https://security.debian.org/debian-security|http://mirrors.aliyun.com/debian-security|g' \
    /etc/apt/sources.list.d/debian.sources
elif [[ -f /etc/apt/sources.list ]]; then
  sed -i \
    's|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com/debian-security|g' \
    /etc/apt/sources.list
fi
