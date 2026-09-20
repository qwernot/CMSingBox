#!/bin/bash

config_file="/var/apps/cmsingbox/etc/network.env"
lan_ip=""
if [ -r "$config_file" ]; then
  lan_ip="$(sed -n 's/^CMSINGBOX_IP=//p' "$config_file" | head -n 1)"
fi
if ! printf '%s' "$lan_ip" | grep -Eq '^([0-9]{1,3}\.){3}[0-9]{1,3}$'; then
  printf 'Status: 503 Service Unavailable\r\n'
  printf 'Content-Type: text/plain; charset=utf-8\r\n\r\n'
  printf 'CMSingBox 独立 IP 配置不存在，请在应用中心重新安装并填写网络参数。\n'
  exit 0
fi
printf 'Status: 302 Found\r\n'
printf 'Location: http://%s/\r\n' "$lan_ip"
printf 'Content-Type: text/plain; charset=utf-8\r\n\r\n'
printf '正在打开 CMSingBox...\n'
