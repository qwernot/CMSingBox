# 授权密钥、更换公钥与客户端打包

CMSingBox 使用 Ed25519 离线签名。授权端保存私钥并签发授权码，客户端只保存公钥并验证授权码。授权只限制可添加的订阅链接数量：未授权默认最多 1 条；订阅内节点数及手动节点数不限制。

## 最重要的规则

- `private.key` 只能留在授权端和离线备份中，禁止发给开发人员、客户或上传 GitHub。
- 需要协助打包时，只提供 `public.key` 中的一行 Base64 公钥。
- 公钥可以写入源码、二进制、部署脚本和公开仓库，它不能用于伪造授权。
- 每套私钥只对应一个公钥。重新生成密钥后，旧授权码无法通过新公钥验证，现有客户需要重新签发授权。
- 授权端宕机不影响已经激活的客户端；私钥丢失才是不可恢复的事故。

## 当前正式公钥

当前正式授权端输出的公钥是：

```text
EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE=
```

正式私钥位于授权服务器：

```text
/root/CMSingBox/deploy/license/license-data/private.key
```

此路径只用于管理员识别备份目标，不得把文件内容复制进文档或仓库。

## 1. 从授权端取得公钥

在授权服务器执行：

```bash
cd /root/CMSingBox/deploy/license
sudo tr -d '\r\n' < license-data/public.key
echo
```

只把命令输出的一行公钥交给负责打包的人。不要执行或发送 `cat license-data/private.key`。

检查公钥是否为合法的 Ed25519 公钥：

```bash
cd /root/CMSingBox/deploy/license
test "$(sudo base64 -d license-data/public.key | wc -c)" -eq 32 \
  && echo "公钥格式正确" \
  || echo "公钥格式错误"
```

## 2. 备份正式私钥

至少保存两份离线备份，并确保备份介质不在同一台服务器。必须一起保存公钥和签发审计记录：

```text
license-data/private.key
license-data/public.key
license-data/license-audit.jsonl
```

恢复授权端时，应恢复原来的 `private.key`，不能再次运行 keygen。恢复后将私钥权限设为 `0600`：

```bash
sudo chmod 0600 /root/CMSingBox/deploy/license/license-data/private.key
```

## 3. 客户端使用公钥的两种方式

客户端读取公钥的优先顺序为：

1. 运行环境变量 `CMSINGBOX_LICENSE_PUBLIC_KEY`；
2. 编译二进制时嵌入的 `LicensePublicKey`。

因此正式发布时应同时更新嵌入公钥和部署脚本的默认公钥。如果服务器还留有旧的环境变量，即使换了新二进制，也会继续使用旧公钥。

原生部署的运行时公钥保存在 `/etc/cmsingbox.env`，Docker 部署的运行时公钥保存在 `/opt/cmsingbox-docker/.env`。

## 4. 更新仓库中的公钥

完整源代码仓库 `CMSingBox` 需要检查 `docker-compose.yml`。公开部署仓库 `CM` 需要更新：

```text
deploy/install.sh
deploy/install-docker.sh
deploy/docker-compose.yml
```

将这些文件中的旧公钥替换为新的公钥。提交前检查旧公钥已经完全删除：

```bash
grep -RIn --exclude-dir=.git '旧公钥完整内容' \
  /path/to/CMSingBox /path/to/CM
```

该命令应没有输出。不要在任何文件中替换或写入私钥。

## 5. 构建正式 Linux 二进制

构建机需要 Go、Node.js，以及 npm 或 pnpm。进入私有源码仓库后执行：

```bash
cd /path/to/CMSingBox

export LICENSE_PUBLIC_KEY='EwFgPIqxKUjPY45bIUHviX4fyZLAGoww6q5QJs9fKcE='
export FREE_SUBSCRIPTION_LIMIT='1'
export VERSION='1.0.10'

# 前端有变化时先重新构建，避免打入旧页面
cd web
pnpm install --frozen-lockfile
pnpm run build
cd ..

# 输出 Linux amd64、arm64、armv7 三种二进制
SKIP_FRONTEND=1 ./build.sh linux
```

输出文件：

```text
dist/cmsingbox-linux-amd64
dist/cmsingbox-linux-arm64
dist/cmsingbox-linux-arm
```

其中 amd64 用于普通 Intel/AMD 服务器，arm64 用于大多数新 ARM 小主机，arm 为 ARMv7 设备。

## 6. 放入公开部署仓库

只复制客户端二进制，不要复制授权端程序、私钥、授权数据库或管理员文档：

```bash
install -m 0755 /path/to/CMSingBox/dist/cmsingbox-linux-amd64 \
  /path/to/CM/bin/cmsingbox-linux-amd64
install -m 0755 /path/to/CMSingBox/dist/cmsingbox-linux-arm64 \
  /path/to/CM/bin/cmsingbox-linux-arm64
install -m 0755 /path/to/CMSingBox/dist/cmsingbox-linux-arm \
  /path/to/CM/bin/cmsingbox-linux-arm
```

然后分别在两个仓库检查差异、提交和推送。公开仓库中不应出现 `private.key`、授权端源码或授权签发工具。

## 7. 更新已部署客户端

仓库和二进制发布完成后，重新执行对应安装命令。脚本会保留数据并写入新的运行时公钥。

原生部署：

```bash
curl -fsSL https://raw.githubusercontent.com/qwernot/CM/main/deploy/install.sh | sudo sh
```

Docker 独立 IP 部署：

```bash
curl -fsSL https://raw.githubusercontent.com/qwernot/CM/main/deploy/install-docker.sh \
  | sudo env CMSINGBOX_IP=192.168.1.20 sh
```

也可以临时指定公钥进行灰度测试：

```bash
sudo env CMSINGBOX_LICENSE_PUBLIC_KEY='新公钥' sh deploy/install.sh
```

正式发布不应长期依赖手工参数，应把正式公钥写入部署脚本默认值。

## 8. 发布前验证

1. 使用全新数据目录启动新二进制，确认未授权额度为 1。
2. 在客户端“设置 → 软件授权”复制设备码。
3. 使用正式授权端给该设备码签发一份短期测试授权。
4. 激活后确认订阅链接额度与签发值一致。
5. 重启客户端，确认授权仍有效。
6. 使用旧私钥签发的授权码应被新客户端拒绝。
7. 清除测试授权后，额度应恢复为 1。

如果新授权码提示签名无效，依次检查客户端的 `/etc/cmsingbox.env` 或 Docker `.env`、二进制的构建公钥、授权端的 `public.key` 是否完全一致。

## 9. 手工生成密钥与授权码

只有首次创建全新授权体系时才生成密钥：

```bash
go run ./cmd/license-tool keygen \
  -private license-private.key \
  -public license-public.key
```

手工签发示例：

```bash
go run ./cmd/license-tool issue \
  -private license-private.key \
  -device 409502 \
  -subscriptions 20 \
  -expires 2027-12-31 \
  -id customer-001
```

省略 `-expires` 表示永久授权。生产环境优先使用授权中心网页签发并保留审计记录。

## 授权行为说明

- 达到额度后不能新增订阅链接，已有订阅仍可刷新和编辑。
- 删除一条订阅后可以重新添加，额度按当前保存的订阅链接数量计算。
- 备份中的订阅数量超过当前授权额度时，导入会被拒绝且不会覆盖现有数据。
- 授权过期或清除后额度恢复为 1，已有超额数据不会自动删除。
- 删除客户端数据目录中的 `device.id` 会产生新设备码，原授权将不再匹配。
- 离线授权不会联网追踪或远程停用客户端。
