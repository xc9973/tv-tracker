#!/bin/sh
set -e

if [ ! -f .env ]; then
  echo "[!] .env 文件不存在，请先复制 .env.example" >&2
  exit 1
fi

echo "[+] 构建 tv-tracker-api 镜像"
docker compose build tv-tracker-api

echo "[+] 构建 tv-tracker-web 镜像"
docker compose build tv-tracker-web

echo "[✓] 构建完成，运行 docker compose up -d 启动服务"
