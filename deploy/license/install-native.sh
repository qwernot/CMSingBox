#!/bin/sh
set -eu

admin_password="${CMSINGBOX_LICENSE_PASSWORD:-Aa666333}"
listen_port="${CMSINGBOX_LICENSE_PORT:-9093}"
go_version="1.24.10"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行，或在命令中加入 sudo。" >&2
  exit 1
fi
if [ "${#admin_password}" -lt 8 ]; then
  echo "授权端密码至少需要 8 位。" >&2
  exit 1
fi
case "$listen_port" in
  ''|*[!0-9]*) echo "授权端端口必须是数字。" >&2; exit 1 ;;
esac
if [ "$listen_port" -lt 1 ] || [ "$listen_port" -gt 65535 ]; then
  echo "授权端端口范围必须是 1-65535。" >&2
  exit 1
fi
for command_name in curl sha256sum tar systemctl; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name" >&2
    exit 1
  fi
done

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
project_dir="$(CDPATH= cd -- "$script_dir/../.." && pwd)"
if [ ! -f "$project_dir/go.mod" ] || [ ! -d "$project_dir/cmd/license-server" ]; then
  echo "项目源码不完整，请在 CMSingBox 私有仓库中运行此脚本。" >&2
  exit 1
fi

machine_arch="$(uname -m)"
case "$machine_arch" in
  x86_64|amd64)
    go_arch="amd64"
    go_sha256="dd52b974e3d9c5a7bbfb222c685806def6be5d6f7efd10f9caa9ca1fa2f47955"
    ;;
  aarch64|arm64)
    go_arch="arm64"
    go_sha256="94a99dae43dab8a3fe337485bbb89214b524285ec53ea02040514b0c2a9c3f94"
    ;;
  armv7l|armv6l)
    go_arch="armv6l"
    go_sha256="2e28837ccde684693edced2df01998723c7e9207890d84a6a05c3f511e6b7ccb"
    ;;
  *)
    echo "暂不支持的架构: $machine_arch" >&2
    exit 1
    ;;
esac

build_dir="$(mktemp -d /tmp/cmsingbox-license-build.XXXXXX)"
cleanup() { rm -rf -- "$build_dir"; }
trap cleanup EXIT INT TERM
go_archive="$build_dir/go.tar.gz"
curl -fL --retry 4 --retry-delay 2 -o "$go_archive" "https://go.dev/dl/go${go_version}.linux-${go_arch}.tar.gz"
printf '%s  %s\n' "$go_sha256" "$go_archive" | sha256sum -c -
tar -xzf "$go_archive" -C "$build_dir"

cd "$project_dir"
CGO_ENABLED=0 "$build_dir/go/bin/go" build -trimpath -ldflags="-s -w" -o "$build_dir/cmsingbox-license-server" ./cmd/license-server
CGO_ENABLED=0 "$build_dir/go/bin/go" build -trimpath -ldflags="-s -w" -o "$build_dir/cmsingbox-license-tool" ./cmd/license-tool

install -d -m 0755 /opt/cmsingbox-license
install -d -m 0700 /var/lib/cmsingbox-license
install -m 0755 "$build_dir/cmsingbox-license-server" /opt/cmsingbox-license/cmsingbox-license-server
install -m 0755 "$build_dir/cmsingbox-license-tool" /opt/cmsingbox-license/cmsingbox-license-tool
install -m 0644 "$script_dir/cmsingbox-license.service" /etc/systemd/system/cmsingbox-license.service

legacy_key_dir="$script_dir/license-data"
if [ ! -f /var/lib/cmsingbox-license/private.key ] && [ -f "$legacy_key_dir/private.key" ]; then
  echo "检测到 Docker 授权私钥，正在迁移到原生数据目录（原文件保留）。"
  install -m 0600 "$legacy_key_dir/private.key" /var/lib/cmsingbox-license/private.key
  if [ -f "$legacy_key_dir/public.key" ]; then
    install -m 0644 "$legacy_key_dir/public.key" /var/lib/cmsingbox-license/public.key
  fi
fi
if [ ! -f /var/lib/cmsingbox-license/private.key ]; then
  /opt/cmsingbox-license/cmsingbox-license-tool keygen -private /var/lib/cmsingbox-license/private.key -public /var/lib/cmsingbox-license/public.key
fi
if [ ! -f /var/lib/cmsingbox-license/public.key ]; then
  /opt/cmsingbox-license/cmsingbox-license-tool public -private /var/lib/cmsingbox-license/private.key -public /var/lib/cmsingbox-license/public.key
fi
chmod 0600 /var/lib/cmsingbox-license/private.key
chmod 0644 /var/lib/cmsingbox-license/public.key

password_hash="$(printf '%s' "$admin_password" | sha256sum | awk '{print $1}')"
umask 077
{
  printf 'CMSINGBOX_LICENSE_PASSWORD_HASH=%s\n' "$password_hash"
  printf 'CMSINGBOX_LICENSE_PORT=%s\n' "$listen_port"
} > /etc/cmsingbox-license.env

systemctl daemon-reload
if [ "${CMSINGBOX_LICENSE_NO_START:-0}" = "1" ]; then
  echo "授权中心程序已经安装；按要求暂不启动服务。"
  exit 0
fi
systemctl enable --now cmsingbox-license.service

server_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$server_ip" ] || server_ip="服务器IP"
echo
echo "CMSingBox 授权中心（原生 systemd）已启动: http://${server_ip}:${listen_port}"
echo "授权端密码: ${admin_password}"
echo "授权公钥: $(tr -d '\r\n' < /var/lib/cmsingbox-license/public.key)"
echo "私钥目录: /var/lib/cmsingbox-license"
echo "必须离线备份 private.key，禁止上传 GitHub。"
