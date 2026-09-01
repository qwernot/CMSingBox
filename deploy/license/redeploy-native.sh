#!/bin/sh
set -eu

formal_public_key="EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE="
key_dir="/var/lib/cmsingbox-license"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行，或在命令前加入 sudo。" >&2
  exit 1
fi
if [ ! -t 0 ]; then
  echo "本脚本需要交互输入，请下载后从终端运行，不要直接通过 curl 管道执行。" >&2
  exit 1
fi
for command_name in base64 cmp curl sha256sum systemctl; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "缺少命令: $command_name" >&2
    exit 1
  fi
done

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
work_dir="$(mktemp -d /tmp/cmsingbox-license-restore.XXXXXX)"
terminal_echo_disabled=0
cleanup() {
  if [ "$terminal_echo_disabled" = "1" ]; then stty echo 2>/dev/null || true; fi
  rm -rf -- "$work_dir"
}
trap cleanup EXIT INT TERM

printf '授权中心端口 [9093]: '
IFS= read -r listen_port
listen_port=${listen_port:-9093}
printf '授权端登录密码 [Aa666333]: '
stty -echo
terminal_echo_disabled=1
IFS= read -r admin_password
stty echo
terminal_echo_disabled=0
printf '\n'
admin_password=${admin_password:-Aa666333}
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
input_public_file=${CMSINGBOX_LICENSE_PUBLIC_KEY_FILE:-}
input_private_file=${CMSINGBOX_LICENSE_PRIVATE_KEY_FILE:-}
if [ -n "$input_public_file" ] || [ -n "$input_private_file" ]; then
  if [ ! -f "$input_public_file" ] || [ ! -f "$input_private_file" ]; then
    echo "指定的公钥或私钥文件不存在。" >&2
    exit 1
  fi
  install -m 0644 "$input_public_file" "$work_dir/public.key"
  install -m 0600 "$input_private_file" "$work_dir/private.key"
  public_key=$(tr -d ' \t\r\n' < "$work_dir/public.key")
  printf '%s\n' "$public_key" > "$work_dir/public.key"
else
  printf '请输入原公钥（Base64，一行）: '
  IFS= read -r public_key
  printf '请输入原私钥（Base64，一行，输入内容不会显示）: '
  stty -echo
  terminal_echo_disabled=1
  IFS= read -r private_key
  stty echo
  terminal_echo_disabled=0
  printf '\n'
  public_key=$(printf '%s' "$public_key" | tr -d ' \t\r\n')
  private_key=$(printf '%s' "$private_key" | tr -d ' \t\r\n')
  printf '%s\n' "$public_key" > "$work_dir/public.key"
  printf '%s\n' "$private_key" > "$work_dir/private.key"
  unset private_key
fi
chmod 0600 "$work_dir/private.key"

if ! base64 -d "$work_dir/public.key" > "$work_dir/public.bin" 2>/dev/null || [ "$(wc -c < "$work_dir/public.bin")" -ne 32 ]; then
  echo "公钥格式错误：Ed25519 公钥解码后必须为 32 字节。" >&2
  exit 1
fi
if ! base64 -d "$work_dir/private.key" > "$work_dir/private.bin" 2>/dev/null || [ "$(wc -c < "$work_dir/private.bin")" -ne 64 ]; then
  echo "私钥格式错误：Ed25519 私钥解码后必须为 64 字节。" >&2
  exit 1
fi
rm -f "$work_dir/public.bin" "$work_dir/private.bin"

CMSINGBOX_LICENSE_INSTALL_ONLY=1 \
sh "$script_dir/install-native.sh"

/opt/cmsingbox-license/cmsingbox-license-tool public \
  -private "$work_dir/private.key" -public "$work_dir/derived-public.key" >/dev/null
if ! cmp -s "$work_dir/public.key" "$work_dir/derived-public.key"; then
  echo "公钥与私钥不配对，未替换任何现有密钥。" >&2
  exit 1
fi

if [ "$public_key" != "$formal_public_key" ]; then
  echo "警告：该公钥与当前正式 CMSingBox 客户端内置公钥不同。"
  echo "使用它签发的授权码不能被当前正式客户端验证。"
  printf '仍然继续安装吗？请输入 YES: '
  IFS= read -r confirmation
  [ "$confirmation" = "YES" ] || { echo "操作已取消，未替换密钥。"; exit 1; }
fi

timestamp=$(date +%Y%m%d-%H%M%S)
backup_dir="$key_dir/backup-$timestamp"
install -d -m 0700 "$key_dir" "$backup_dir"
had_old_key=0
if [ -f "$key_dir/private.key" ]; then
  install -m 0600 "$key_dir/private.key" "$backup_dir/private.key"
  had_old_key=1
fi
if [ -f "$key_dir/public.key" ]; then
  install -m 0644 "$key_dir/public.key" "$backup_dir/public.key"
fi
if [ -f "$key_dir/license-audit.jsonl" ]; then
  install -m 0600 "$key_dir/license-audit.jsonl" "$backup_dir/license-audit.jsonl"
fi
if [ -f /etc/cmsingbox-license.env ]; then
  install -m 0600 /etc/cmsingbox-license.env "$backup_dir/cmsingbox-license.env"
fi

systemctl stop cmsingbox-license.service 2>/dev/null || true
install -m 0600 "$work_dir/private.key" "$key_dir/private.key"
install -m 0644 "$work_dir/public.key" "$key_dir/public.key"
password_hash=$(printf '%s' "$admin_password" | sha256sum | awk '{print $1}')
umask 077
{
  printf 'CMSINGBOX_LICENSE_PASSWORD_HASH=%s\n' "$password_hash"
  printf 'CMSINGBOX_LICENSE_PORT=%s\n' "$listen_port"
} > /etc/cmsingbox-license.env
systemctl daemon-reload
systemctl enable --now cmsingbox-license.service

if ! curl -fsS --max-time 10 "http://127.0.0.1:${listen_port}/healthz" >/dev/null; then
  echo "新授权中心健康检查失败，正在恢复旧密钥。" >&2
  systemctl stop cmsingbox-license.service 2>/dev/null || true
  if [ "$had_old_key" = "1" ]; then
    install -m 0600 "$backup_dir/private.key" "$key_dir/private.key"
    if [ -f "$backup_dir/public.key" ]; then install -m 0644 "$backup_dir/public.key" "$key_dir/public.key"; fi
  fi
  if [ -f "$backup_dir/cmsingbox-license.env" ]; then
    install -m 0600 "$backup_dir/cmsingbox-license.env" /etc/cmsingbox-license.env
  fi
  systemctl daemon-reload
  if [ "$had_old_key" = "1" ]; then systemctl start cmsingbox-license.service || true; fi
  exit 1
fi

server_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -n "$server_ip" ] || server_ip="服务器IP"
echo
echo "迁移完成，授权中心地址: http://${server_ip}:${listen_port}"
echo "授权公钥: $public_key"
echo "旧文件备份目录: $backup_dir"
echo "请继续离线保存 private.key，禁止上传 GitHub。"
