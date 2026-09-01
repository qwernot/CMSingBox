#!/bin/sh
set -eu

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 root 运行，或在命令前加入 sudo。" >&2
  exit 1
fi
if [ ! -t 0 ]; then
  echo "本脚本需要交互输入解密口令，请从终端运行。" >&2
  exit 1
fi
if ! command -v openssl >/dev/null 2>&1; then
  echo "缺少 openssl，请先安装 openssl。" >&2
  exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
sealed_dir="$script_dir/sealed"
encrypted_private="$sealed_dir/formal-private.key.enc"
public_key="$sealed_dir/formal-public.key"
if [ ! -f "$encrypted_private" ] || [ ! -f "$public_key" ]; then
  echo "固定授权密钥包不完整，请重新拉取 CMSingBox 私有仓库。" >&2
  exit 1
fi

work_dir=$(mktemp -d /tmp/cmsingbox-formal-key.XXXXXX)
terminal_echo_disabled=0
cleanup() {
  if [ "$terminal_echo_disabled" = "1" ]; then stty echo 2>/dev/null || true; fi
  rm -rf -- "$work_dir"
}
trap cleanup EXIT INT TERM

printf '请输入固定私钥解密口令（输入内容不会显示）: '
stty -echo
terminal_echo_disabled=1
IFS= read -r decrypt_password
stty echo
terminal_echo_disabled=0
printf '\n'
if [ -z "$decrypt_password" ]; then
  echo "解密口令不能为空。" >&2
  exit 1
fi

if ! printf '%s' "$decrypt_password" | openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 -md sha256 \
  -in "$encrypted_private" -out "$work_dir/private.key" -pass stdin 2>/dev/null; then
  echo "私钥解密失败：口令错误或密钥包损坏。" >&2
  exit 1
fi
unset decrypt_password
chmod 0600 "$work_dir/private.key"

CMSINGBOX_LICENSE_PUBLIC_KEY_FILE="$public_key" \
CMSINGBOX_LICENSE_PRIVATE_KEY_FILE="$work_dir/private.key" \
sh "$script_dir/redeploy-native.sh"
