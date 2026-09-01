#!/bin/sh
set -eu

repository="qwernot/CMSingBox"
repository_ref="${CMSINGBOX_REPO_REF:-main}"
install_dir="${CMSINGBOX_LICENSE_INSTALL_DIR:-/opt/cmsingbox-license-docker}"
archive_url="https://github.com/${repository}/archive/refs/heads/${repository_ref}.tar.gz"
admin_password="${CMSINGBOX_LICENSE_PASSWORD:-}"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行，或在命令中加入 sudo。" >&2
  exit 1
fi
if [ -z "$admin_password" ] || [ "${#admin_password}" -lt 8 ]; then
  echo "请通过 CMSINGBOX_LICENSE_PASSWORD 提供至少 8 位的授权端密码。" >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "缺少 curl，请先安装 curl。" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "未检测到 Docker，正在安装 Docker Engine..."
  curl -fsSL https://get.docker.com | sh
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "需要 Docker Compose v2 插件。" >&2
  exit 1
fi

temporary_dir="$(mktemp -d /tmp/cmsingbox-license-install.XXXXXX)"
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM
curl -fsSL "$archive_url" | tar -xz -C "$temporary_dir"
source_dir="$(find "$temporary_dir" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [ -z "$source_dir" ] || [ ! -f "$source_dir/docker-compose.license.yml" ]; then
  echo "安装包不完整。" >&2
  exit 1
fi

mkdir -p "$install_dir" "$install_dir/license-data"
cp -a "$source_dir/." "$install_dir/"
cd "$install_dir"

password_hash="$(printf '%s' "$admin_password" | sha256sum | awk '{print $1}')"
umask 077
printf 'CMSINGBOX_LICENSE_PASSWORD_HASH=%s\n' "$password_hash" > .env

echo "正在构建授权中心..."
docker compose -f docker-compose.license.yml build
if [ ! -f license-data/private.key ]; then
  docker compose -f docker-compose.license.yml run --rm --no-deps \
    --entrypoint /usr/local/bin/cmsingbox-license-tool cmsingbox-license \
    keygen -private /license/private.key -public /license/public.key
fi
chmod 600 license-data/private.key .env
docker compose -f docker-compose.license.yml up -d

server_ip="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')"
if [ -z "$server_ip" ]; then
  server_ip="服务器IP"
fi

echo
echo "CMSingBox 授权中心已启动。"
echo "授权地址: http://${server_ip}:${CMSINGBOX_LICENSE_PORT:-9093}"
echo "授权公钥: $(tr -d '\r\n' < license-data/public.key)"
echo "私钥目录: ${install_dir}/license-data（必须单独备份，禁止上传 GitHub）"
echo "注意：新生成密钥时，CMSingBox 主程序必须使用上方公钥重新构建。"
