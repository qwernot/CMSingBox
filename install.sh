#!/bin/sh
set -eu

repository="qwernot/CMSingBox"
repository_ref="${CMSINGBOX_REPO_REF:-main}"
install_dir="${CMSINGBOX_INSTALL_DIR:-/opt/cmsingbox-docker}"
archive_url="https://github.com/${repository}/archive/refs/heads/${repository_ref}.tar.gz"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行，或在命令中加入 sudo。" >&2
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

temporary_dir="$(mktemp -d /tmp/cmsingbox-install.XXXXXX)"
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM

echo "正在下载 CMSingBox ${repository_ref}..."
curl -fsSL "$archive_url" | tar -xz -C "$temporary_dir"
source_dir="$(find "$temporary_dir" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [ -z "$source_dir" ] || [ ! -f "$source_dir/docker-compose.yml" ]; then
  echo "安装包不完整。" >&2
  exit 1
fi

mkdir -p "$install_dir" "$install_dir/data"
cp -a "$source_dir/." "$install_dir/"
cd "$install_dir"

echo "正在构建并启动 CMSingBox..."
docker compose up -d --build

lan_ip="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}')"
if [ -z "$lan_ip" ]; then
  lan_ip="小主机局域网IP"
fi

echo
echo "CMSingBox 已启动。"
echo "管理地址: http://${lan_ip}:9092"
echo "初始账号: admin"
echo "初始密码: admin"
echo "数据目录: ${install_dir}/data"
echo "首次登录后请立即修改密码，并在设置中开启“允许局域网访问”。"
echo "查看日志: cd ${install_dir} && docker compose logs -f --tail=100"
