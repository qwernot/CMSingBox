#!/bin/sh
set -eu

admin_password="${CMSINGBOX_LICENSE_PASSWORD:-Aa666333}"
listen_port="${CMSINGBOX_LICENSE_PORT:-9093}"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行，或在命令中加入 sudo。" >&2
  exit 1
fi
if [ "${#admin_password}" -lt 8 ]; then
  echo "授权端密码至少需要 8 位。" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "请先安装 Docker Engine 和 Docker Compose v2。" >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "需要 Docker Compose v2 插件。" >&2
  exit 1
fi

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
project_dir="$(CDPATH= cd -- "$script_dir/../.." && pwd)"
if [ ! -f "$project_dir/go.mod" ] || [ ! -d "$project_dir/cmd/license-server" ]; then
  echo "项目源码不完整，请在 CMSingBox 私有仓库中运行此脚本。" >&2
  exit 1
fi

cd "$script_dir"
mkdir -p license-data
password_hash="$(printf '%s' "$admin_password" | sha256sum | awk '{print $1}')"
umask 077
{
  printf 'CMSINGBOX_LICENSE_PASSWORD_HASH=%s\n' "$password_hash"
  printf 'CMSINGBOX_LICENSE_PORT=%s\n' "$listen_port"
} > .env

docker compose build
if [ ! -f license-data/private.key ]; then
  docker compose run --rm --no-deps \
    --entrypoint /usr/local/bin/cmsingbox-license-tool cmsingbox-license \
    keygen -private /license/private.key -public /license/public.key
fi
chmod 0600 license-data/private.key .env
docker compose up -d

server_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$server_ip" ] || server_ip="服务器IP"
echo
echo "CMSingBox 授权中心已启动: http://${server_ip}:${listen_port}"
echo "授权端密码: ${admin_password}"
echo "授权公钥: $(tr -d '\r\n' < license-data/public.key)"
echo "私钥目录: ${script_dir}/license-data"
echo "必须离线备份 private.key，禁止上传 GitHub。"
