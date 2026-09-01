# 授权中心重装与原密钥恢复

本文只面向 CMSingBox 授权管理员，说明在服务器重装、硬盘迁移、Docker 改原生、原生改 Docker 或授权服务损坏后，如何继续使用以前的 Ed25519 私钥和公钥。

## 最简方法：交互式恢复脚本

如果原公钥、原私钥都已保存，推荐直接使用交互式脚本。它会依次询问端口、登录密码、公钥和私钥，并自动完成格式检查、密钥配对检查、旧密钥备份、systemd 安装、启动与健康检查：

```bash
git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
sudo sh deploy/license/redeploy-native.sh
```

按提示输入即可：

```text
授权中心端口 [9093]:
授权端登录密码 [Aa666333]:
请输入原公钥（Base64，一行）:
请输入原私钥（Base64，一行，输入内容不会显示）:
```

直接按回车会使用方括号里的默认端口和密码。私钥输入期间终端不会显示字符，这是正常现象；粘贴完整私钥后按回车即可。

脚本具有以下保护：

1. 公钥必须能解码为 32 字节，私钥必须能解码为 64 字节。
2. 脚本会从私钥重新计算公钥，二者不配对时立即停止，不修改现有密钥。
3. 替换前会把原密钥和授权审计记录备份到 `/var/lib/cmsingbox-license/backup-日期时间/`。
4. 新服务健康检查失败时会自动恢复旧密钥并尝试重新启动旧服务。
5. 如果输入的公钥不是当前正式客户端内置公钥，脚本会明确警告并要求输入 `YES` 才能继续。

脚本必须从可交互终端运行，不能使用 `curl ... | sh`，因为私钥需要安全地从终端读取。迁移完成后仍应把 `private.key` 保存到至少两份离线加密介质；GitHub 即使是私有仓库也不能用于保存私钥。

## 1. 最重要的结论

CMSingBox 客户端二进制内置的是授权公钥，授权中心持有与之配对的私钥。重装授权中心时必须继续使用原来的私钥；只要私钥不变，已经发布的客户端、已经签发的授权码以及以后签发的新授权码都可以继续离线验签，授权中心临时离线也不影响已经激活的客户设备。

当前正式客户端内置的公钥是：

```text
EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE=
```

重装前后都必须确认授权中心导出的公钥与上面这一行完全一致。任何字符不同都表示密钥不匹配，此时禁止继续签发。

> 严禁删除旧私钥后重新执行 `keygen`。新私钥无法签发能被旧客户端接受的授权码。私钥一旦永久丢失，只能生成新密钥、重新构建并重新发布所有客户端。

## 2. 先确认旧密钥在哪里

Docker 部署默认位置：

```text
CMSingBox/deploy/license/license-data/private.key
CMSingBox/deploy/license/license-data/public.key
CMSingBox/deploy/license/license-data/license-audit.jsonl
```

原生 systemd 部署默认位置：

```text
/var/lib/cmsingbox-license/private.key
/var/lib/cmsingbox-license/public.key
/var/lib/cmsingbox-license/license-audit.jsonl
```

查看文件是否存在和权限时只使用 `ls` 或 `stat`，不要使用 `cat private.key`：

```bash
sudo ls -la /var/lib/cmsingbox-license
sudo stat -c '%a %U:%G %n' \
  /var/lib/cmsingbox-license/private.key \
  /var/lib/cmsingbox-license/public.key
```

正确权限应为：私钥 `0600`，公钥 `0644`，数据目录 `0700`。

## 3. 重装前做双份备份

在旧服务器上创建一个只允许 root 访问的备份目录：

```bash
license_backup_dir="/root/cmsingbox-license-backup-$(date +%Y%m%d-%H%M%S)"
sudo install -d -m 0700 "$license_backup_dir"
```

原生部署执行：

```bash
sudo install -m 0600 /var/lib/cmsingbox-license/private.key "$license_backup_dir/private.key"
sudo install -m 0644 /var/lib/cmsingbox-license/public.key "$license_backup_dir/public.key"
if sudo test -f /var/lib/cmsingbox-license/license-audit.jsonl; then
  sudo install -m 0600 /var/lib/cmsingbox-license/license-audit.jsonl "$license_backup_dir/license-audit.jsonl"
fi
```

Docker 部署需要在私有源码目录中执行：

```bash
cd /root/CMSingBox
sudo install -m 0600 deploy/license/license-data/private.key "$license_backup_dir/private.key"
sudo install -m 0644 deploy/license/license-data/public.key "$license_backup_dir/public.key"
if sudo test -f deploy/license/license-data/license-audit.jsonl; then
  sudo install -m 0600 deploy/license/license-data/license-audit.jsonl "$license_backup_dir/license-audit.jsonl"
fi
```

生成校验文件：

```bash
sudo sh -c "cd '$license_backup_dir' && sha256sum private.key public.key > SHA256SUMS"
sudo chmod 0600 "$license_backup_dir/SHA256SUMS"
sudo ls -la "$license_backup_dir"
```

然后把整个备份目录复制到至少两个不在原服务器上的加密存储位置。建议一份放离线 U 盘或加密硬盘，一份放受控的密码库或加密备份。不要发送到聊天群、邮件附件、网盘公开链接，也不要提交 GitHub。

## 4. 在新服务器验证备份完整性

把备份安全传到新服务器后，先检查校验值：

```bash
license_backup_dir="/root/cmsingbox-license-backup"
sudo sh -c "cd '$license_backup_dir' && sha256sum -c SHA256SUMS"
```

结果必须显示 `private.key: OK` 和 `public.key: OK`。

只显示公钥进行人工核对，绝不显示私钥：

```bash
sudo cat "$license_backup_dir/public.key" | tr -d '\r\n'
echo
```

输出必须是：

```text
EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE=
```

## 5. 推荐方案：恢复为原生 systemd 授权中心

### 5.1 克隆私有源码

```bash
git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
```

如果仓库已经存在：

```bash
cd /root/CMSingBox
git pull --ff-only
```

### 5.2 先放回旧密钥

必须在执行安装脚本之前恢复密钥：

```bash
sudo install -d -m 0700 /var/lib/cmsingbox-license
sudo install -m 0600 "$license_backup_dir/private.key" /var/lib/cmsingbox-license/private.key
sudo install -m 0644 "$license_backup_dir/public.key" /var/lib/cmsingbox-license/public.key
if sudo test -f "$license_backup_dir/license-audit.jsonl"; then
  sudo install -m 0600 "$license_backup_dir/license-audit.jsonl" /var/lib/cmsingbox-license/license-audit.jsonl
fi
```

再次检查公钥：

```bash
sudo cat /var/lib/cmsingbox-license/public.key | tr -d '\r\n'
echo
```

### 5.3 执行原生安装

```bash
sudo env \
  CMSINGBOX_LICENSE_PASSWORD='Aa666333' \
  CMSINGBOX_LICENSE_PORT='9093' \
  sh deploy/license/install-native.sh
```

安装脚本检测到 `/var/lib/cmsingbox-license/private.key` 已存在时会保留它，不会生成新密钥。脚本会安装二进制、环境文件和 systemd 服务。

### 5.4 验收 systemd 服务

```bash
sudo systemctl status cmsingbox-license --no-pager
sudo systemctl is-enabled cmsingbox-license
sudo systemctl is-active cmsingbox-license
curl -fsS http://127.0.0.1:9093/healthz
```

预期结果分别为 `enabled`、`active` 和 `ok`。

确认监听端口：

```bash
sudo ss -lntp | grep ':9093 '
```

然后用浏览器打开：

```text
http://授权服务器IP:9093
```

使用安装时设置的管理员密码登录。不要立即生成正式客户授权，先完成第 9 节的密钥配对验证。

## 6. Docker 授权中心改为原生 systemd

此迁移必须确保 9093 同一时间只由一个服务占用。

### 6.1 先备份 Docker 密钥

按照第 3 节完成备份并验证公钥。

### 6.2 在备用端口验证原生服务

把旧密钥恢复到 `/var/lib/cmsingbox-license` 后，先使用备用端口安装：

```bash
cd /root/CMSingBox
sudo env \
  CMSINGBOX_LICENSE_PASSWORD='Aa666333' \
  CMSINGBOX_LICENSE_PORT='19093' \
  sh deploy/license/install-native.sh
curl -fsS http://127.0.0.1:19093/healthz
```

确认输出 `ok` 后再切换正式端口。

### 6.3 停止 Docker，切换回 9093

```bash
sudo docker stop cmsingbox-license
sudo sed -i 's/^CMSINGBOX_LICENSE_PORT=.*/CMSINGBOX_LICENSE_PORT=9093/' /etc/cmsingbox-license.env
sudo systemctl restart cmsingbox-license
```

验证：

```bash
sudo systemctl is-active cmsingbox-license
curl -fsS http://127.0.0.1:9093/healthz
sudo docker ps -a --filter name='^cmsingbox-license$'
```

旧 Docker 容器建议先保留但保持停止，观察几天后再决定是否删除。保留容器不会影响原生服务，也便于快速回退。

### 6.4 回滚到 Docker

如果原生服务异常：

```bash
sudo systemctl stop cmsingbox-license
sudo docker start cmsingbox-license
curl -fsS http://127.0.0.1:9093/healthz
```

确认 Docker 恢复后再排查原生服务。不要让两个服务同时抢占 9093。

## 7. 原生授权中心改为 Docker

先把原生密钥复制到私有仓库的 Docker 数据目录：

```bash
cd /root/CMSingBox
sudo install -d -m 0700 deploy/license/license-data
sudo install -m 0600 /var/lib/cmsingbox-license/private.key deploy/license/license-data/private.key
sudo install -m 0644 /var/lib/cmsingbox-license/public.key deploy/license/license-data/public.key
if sudo test -f /var/lib/cmsingbox-license/license-audit.jsonl; then
  sudo install -m 0600 /var/lib/cmsingbox-license/license-audit.jsonl deploy/license/license-data/license-audit.jsonl
fi
```

先停止原生服务，再启动 Docker：

```bash
sudo systemctl stop cmsingbox-license
sudo env CMSINGBOX_LICENSE_PASSWORD='Aa666333' sh deploy/license/install.sh
curl -fsS http://127.0.0.1:9093/healthz
```

Docker 安装脚本检测到 `license-data/private.key` 已存在时不会重新生成密钥。

## 8. 只有 private.key，没有 public.key

公钥可以从原私钥重新推导，私钥不能从公钥恢复。先安装或构建最新版 `cmsingbox-license-tool`，然后执行：

```bash
sudo install -d -m 0700 /var/lib/cmsingbox-license
sudo install -m 0600 "$license_backup_dir/private.key" /var/lib/cmsingbox-license/private.key
sudo /opt/cmsingbox-license/cmsingbox-license-tool public \
  -private /var/lib/cmsingbox-license/private.key \
  -public /var/lib/cmsingbox-license/public.key
sudo chmod 0644 /var/lib/cmsingbox-license/public.key
```

随后显示并核对公钥：

```bash
sudo cat /var/lib/cmsingbox-license/public.key | tr -d '\r\n'
echo
```

只有输出与正式公钥完全一致时才能继续部署。

## 9. 验证私钥和客户端公钥确实配对

最安全的验证方式是从私钥重新导出一个临时公钥，再与正式公钥文件比较：

```bash
license_verify_dir="$(mktemp -d /tmp/cmsingbox-license-verify.XXXXXX)"
sudo /opt/cmsingbox-license/cmsingbox-license-tool public \
  -private /var/lib/cmsingbox-license/private.key \
  -public "$license_verify_dir/public.from-private.key"
sudo cmp /var/lib/cmsingbox-license/public.key "$license_verify_dir/public.from-private.key"
echo "私钥与公钥配对正确"
sudo rm -f "$license_verify_dir/public.from-private.key"
sudo rmdir "$license_verify_dir"
```

`cmp` 没有输出且命令继续执行，才表示两者完全一致。

还应确认当前 CMSingBox 客户端内置的公钥。管理员构建客户端时必须使用：

```bash
LICENSE_PUBLIC_KEY='EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE=' \
FREE_SUBSCRIPTION_LIMIT=1 \
VERSION=你的版本号 \
./build.sh linux
```

不要把私钥传给客户端，也不要把私钥写入上述命令。

## 10. 密码与端口修改

原生服务的密码摘要和端口保存在：

```text
/etc/cmsingbox-license.env
```

重新设置密码：

```bash
new_license_password='换成新的高强度密码'
new_license_hash="$(printf '%s' "$new_license_password" | sha256sum | awk '{print $1}')"
sudo sed -i "s/^CMSINGBOX_LICENSE_PASSWORD_HASH=.*/CMSINGBOX_LICENSE_PASSWORD_HASH=$new_license_hash/" /etc/cmsingbox-license.env
sudo systemctl restart cmsingbox-license
```

修改端口：

```bash
sudo sed -i 's/^CMSINGBOX_LICENSE_PORT=.*/CMSINGBOX_LICENSE_PORT=9093/' /etc/cmsingbox-license.env
sudo systemctl restart cmsingbox-license
```

修改后检查防火墙、反向代理和浏览器访问地址。

## 11. 完整验收清单

重装完成后逐项确认：

1. `cmsingbox-license.service` 为 `enabled` 和 `active`。
2. `/healthz` 返回 `ok`。
3. 9093 只被一个授权服务监听。
4. 私钥权限为 `0600`，数据目录权限为 `0700`。
5. 从私钥导出的公钥与 `public.key` 完全一致。
6. 公钥与正式客户端公钥 `EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE=` 完全一致。
7. 管理员可以登录授权页面。
8. 使用测试设备码签发一枚测试授权，并在对应测试客户端成功激活。
9. 已有客户设备在授权中心离线时仍能显示已授权。
10. 私钥备份已经复制到两个独立的加密存储位置。

## 12. 常见错误

### 公钥与正式公钥不同

立即停止签发。检查是否拿错备份、是否在空目录运行过 `keygen`、是否把测试密钥当成正式密钥。不要尝试通过修改客户端数据文件绕过验签。

### 9093 端口占用

```bash
sudo ss -lntp | grep ':9093 '
sudo systemctl status cmsingbox-license --no-pager
sudo docker ps --filter name='cmsingbox-license'
```

通常是 Docker 和 systemd 同时启动。保留需要的一个，停止另一个。

### 登录显示来源无效

浏览器必须直接访问授权中心或经过正确配置的反向代理。确认反向代理保留 `Host`、`X-Forwarded-Proto`，并使用同一域名打开页面和提交表单。不要通过一个域名打开页面后向另一个 IP 提交登录。

### 服务启动但签发失败

```bash
sudo journalctl -u cmsingbox-license -n 100 --no-pager
sudo stat -c '%a %U:%G %n' /var/lib/cmsingbox-license/private.key
```

检查私钥文件是否存在、格式是否完整、服务是否有读取权限。不要把私钥内容粘贴到日志或工单中。

### 私钥永久丢失

公钥无法反推出私钥。只能生成新密钥对，用新公钥重新构建 CMSingBox 客户端，并为客户重新签发授权。旧客户端不能接受新私钥签发的授权码，因此离线备份是不可替代的。
