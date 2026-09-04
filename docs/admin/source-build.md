# CMSingBox 源码修改、构建与发布

本文面向持有 `qwernot/CMSingBox` 私有仓库的项目管理员，说明以后自行修改前端或后端后，如何测试、构建客户端、更新原生部署、制作 Docker 镜像，以及单独重建授权中心。

> 仓库内的 `cmd/sbm` 是普通 CMSingBox 客户端，`cmd/license-server` 和 `cmd/license-tool` 属于授权中心。客户端只需要正式公钥；任何客户端构建步骤都不需要、也不得读取授权私钥。

## 1. 准备构建环境

推荐在 Linux 构建机上安装以下工具：

- Git
- Go 1.24.10 或更高版本；最低入口版本是 Go 1.23，仓库会请求 `go1.24.10` 工具链
- Node.js 20 LTS
- pnpm 10
- 构建 Docker 镜像时还需要 Docker Engine、Compose v2 和 buildx

确认版本：

```bash
git --version
go version
node --version
pnpm --version
docker version
docker buildx version
```

如果已经安装 Node.js 20，但没有 pnpm，可使用 Node.js 自带的 Corepack：

```bash
sudo corepack enable
corepack prepare pnpm@10.28.0 --activate
```

首次克隆私有仓库：

```bash
git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
```

如果当前网络无法连接 GitHub SSH 的 22 端口，可改用 443 端口：

```bash
git clone ssh://git@ssh.github.com:443/qwernot/CMSingBox.git
cd CMSingBox
```

已经有源码目录时，只更新主分支：

```bash
cd /path/to/CMSingBox
git status --short
git pull --ff-only origin main
```

`git pull` 前必须先查看 `git status`。如果有尚未提交的修改，应先提交或建立新分支，不要用 `git reset --hard` 覆盖自己的代码。

## 2. 源码目录与修改位置

| 路径 | 用途 |
| --- | --- |
| `web/src/` | React 管理页面和样式 |
| `web/public/` | favicon、登录页图片等静态资源 |
| `web/dist/` | 前端构建产物，最终嵌入 Go 二进制，不要直接修改 |
| `cmd/sbm/` | CMSingBox 客户端入口、版本和编译参数 |
| `internal/api/` | 后台 API、登录、备份、防火墙等功能 |
| `internal/builder/` | sing-box 配置生成 |
| `internal/dnsproxy/` | 独立 DNS 服务 |
| `internal/licensing/` | 客户端授权验证与授权签名公共逻辑 |
| `cmd/license-server/` | 独立授权签发网站 |
| `cmd/license-tool/` | 密钥和授权命令行工具 |
| `build.sh` | 客户端跨架构构建脚本 |
| `Dockerfile.release` | 把已构建客户端和 sing-box 内核打入正式镜像 |

修改 Go 文件后执行格式化：

```bash
gofmt -w 修改过的.go
```

不要直接修改 `web/dist`。前端源码改完后重新构建，它会覆盖该目录。

## 3. 本地开发前端

安装锁定版本的依赖：

```bash
cd /path/to/CMSingBox/web
pnpm install --frozen-lockfile
```

启动开发页面：

```bash
pnpm run dev
```

开发服务器只用于调整页面。需要调用真实后台 API 时，仍应启动 CMSingBox 后端，或按 Vite 配置使用代理。修改完成后执行：

```bash
pnpm run build
```

`pnpm run lint` 是额外的代码质量检查。当前历史代码仍有一批 ESLint 存量问题，所以它不等同于能否生成发布包；修改相关文件时应逐步清理对应告警，而 `pnpm run build` 必须成功。

`pnpm run build` 会生成 `web/dist/index.html` 等文件。Go 的 `web/embed.go` 会在编译时把整个 `web/dist` 打进客户端二进制。

> `build.sh` 检测到 `web/dist/index.html` 已存在时会跳过前端构建。因此，只要改过 `web/src` 或 `web/public`，就必须先手动执行一次 `pnpm run build`，否则新二进制里可能还是旧页面。

## 4. 后端测试

返回仓库根目录运行完整 Go 测试：

```bash
cd /path/to/CMSingBox
go test ./...
```

只修改某个模块时可以先运行对应测试，例如：

```bash
go test ./internal/builder
go test ./internal/dnsproxy
go test ./internal/licensing
go test ./cmd/license-server
```

正式构建前仍应再执行一次 `go test ./...`。如果修改了订阅限制、授权验签或配置生成，不能只看页面是否正常。

## 5. 设置版本和授权公钥

正式客户端必须嵌入与总授权中心私钥配对的公钥。当前正式公钥记录在[授权密钥、更换公钥与客户端打包](licensing.md)中，也可以从授权端的 `public.key` 读取。

下面只读取公钥，不读取私钥：

```bash
export LICENSE_PUBLIC_KEY="$(tr -d '\r\n' < /安全路径/public.key)"
export FREE_SUBSCRIPTION_LIMIT='1'
export VERSION='1.0.27'
```

将示例版本 `1.0.27` 改成准备发布的新版本。版本号、公钥和未授权订阅链接上限通过 Go `-ldflags` 写入二进制，而不是写入前端。

检查公钥格式：

```bash
test "$(printf '%s' "$LICENSE_PUBLIC_KEY" | base64 -d | wc -c)" -eq 32 \
  && echo '公钥格式正确' \
  || echo '公钥格式错误，停止构建'
```

注意：运行时环境变量 `CMSINGBOX_LICENSE_PUBLIC_KEY` 的优先级高于二进制内嵌公钥。更新服务器后若仍提示授权签名无效，应检查 `/etc/cmsingbox.env` 或 Docker 运行环境是否还保存了旧公钥。

## 6. 构建 CMSingBox 客户端

### 构建 Linux 全部架构

先构建前端，再调用仓库脚本：

```bash
cd /path/to/CMSingBox/web
pnpm install --frozen-lockfile
pnpm run build

cd ..
go test ./...
VERSION="$VERSION" \
LICENSE_PUBLIC_KEY="$LICENSE_PUBLIC_KEY" \
FREE_SUBSCRIPTION_LIMIT="$FREE_SUBSCRIPTION_LIMIT" \
SKIP_FRONTEND=1 ./build.sh linux
```

输出文件：

```text
dist/cmsingbox-linux-amd64
dist/cmsingbox-linux-arm64
dist/cmsingbox-linux-arm
```

- `amd64`：Intel/AMD 64 位服务器和电脑
- `arm64`：大多数 ARM64 小主机、开发板和新款 RouterOS 设备
- `arm`：ARMv7 设备

### 只构建当前机器

```bash
VERSION="$VERSION" \
LICENSE_PUBLIC_KEY="$LICENSE_PUBLIC_KEY" \
FREE_SUBSCRIPTION_LIMIT="$FREE_SUBSCRIPTION_LIMIT" \
SKIP_FRONTEND=1 ./build.sh current
```

构建完成后，`dist/cmsingbox` 会链接到当前平台对应的二进制。

### 生成校验值

```bash
sha256sum dist/cmsingbox-linux-* > dist/SHA256SUMS
file dist/cmsingbox-linux-*
```

`build.sh` 每次都会清空 `dist`，所以不要把需要保留的 sing-box 内核或旧版文件只放在该目录中。

## 7. 在测试端启动新二进制

不要直接用生产数据做第一次测试。建立临时数据目录，并使用未占用的后台端口：

```bash
TEST_DATA="$(mktemp -d /tmp/cmsingbox-test.XXXXXX)"
CMSINGBOX_LICENSE_PUBLIC_KEY="$LICENSE_PUBLIC_KEY" \
  ./dist/cmsingbox -data "$TEST_DATA" -port 19092
```

另一个终端访问 `http://构建机IP:19092`，初始账号和密码均为 `admin`。至少检查：

1. 登录页、设置页和移动端页面是本次修改后的版本。
2. 未授权时只能添加 1 条订阅链接。
3. 用正式授权端签发短期测试码后可以正常激活。
4. 订阅刷新、配置生成、DNS、HTTP/SOCKS5 和代理控制台正常。
5. 日志中没有持续出现 `ERROR` 或重启循环。

按 `Ctrl+C` 结束测试，然后删除临时目录：

```bash
case "$TEST_DATA" in
  /tmp/cmsingbox-test.*) rm -rf -- "$TEST_DATA" ;;
  *) echo '临时目录不符合预期，拒绝删除' ;;
esac
```

删除前务必确认变量以 `/tmp/cmsingbox-test.` 开头，绝不能把生产数据目录代入该命令。

## 8. 更新原生 systemd 部署

先在 CMSingBox 后台导出备份，再在服务器复制新架构对应的二进制。以下示例是 amd64：

```bash
sudo systemctl stop cmsingbox
sudo cp -a /opt/cmsingbox/cmsingbox \
  "/opt/cmsingbox/cmsingbox.backup.$(date +%Y%m%d-%H%M%S)"
sudo install -m 0755 dist/cmsingbox-linux-amd64 /opt/cmsingbox/cmsingbox
sudo systemctl start cmsingbox
sudo systemctl status cmsingbox --no-pager
sudo journalctl -u cmsingbox -n 100 --no-pager
```

该操作只替换程序，不删除 `/var/lib/cmsingbox`，订阅、规则和授权数据会保留。若启动失败，可停止服务并把刚才的 `.backup.时间` 文件复制回 `/opt/cmsingbox/cmsingbox`。

如果希望用户的一键安装脚本也获得新版本，把三个 Linux 客户端复制到公开 `CM` 仓库：

```bash
install -m 0755 dist/cmsingbox-linux-amd64 /path/to/CM/bin/cmsingbox-linux-amd64
install -m 0755 dist/cmsingbox-linux-arm64 /path/to/CM/bin/cmsingbox-linux-arm64
install -m 0755 dist/cmsingbox-linux-arm /path/to/CM/bin/cmsingbox-linux-arm
```

检查后分别提交私有源码仓库和公开部署仓库。公开仓库只放客户端二进制和部署文件，不放授权端源码、私钥、管理密码或签发记录。

## 9. 构建包含默认内核的 Docker 镜像

CMSingBox 与 sing-box 是两个独立程序。源码构建只会产生 CMSingBox 客户端；正式 Docker 镜像还需要为每个架构准备 sing-box 内核：

```text
dist/sing-box-linux-amd64
dist/sing-box-linux-arm64
dist/sing-box-linux-arm
```

可以从已经审核并保存到 `CM/bin` 的版本复制，不要在构建时临时下载来源不明的文件：

```bash
cp /path/to/CM/bin/sing-box-linux-amd64 dist/
cp /path/to/CM/bin/sing-box-linux-arm64 dist/
cp /path/to/CM/bin/sing-box-linux-arm dist/
chmod 0755 dist/sing-box-linux-*
```

确认六个文件都存在：

```bash
for arch in amd64 arm64 arm; do
  test -x "dist/cmsingbox-linux-$arch" || exit 1
  test -x "dist/sing-box-linux-$arch" || exit 1
done
```

首次准备 buildx：

```bash
docker buildx create --name cmsingbox-builder --use
docker buildx inspect --bootstrap
```

已有同名构建器时只需：

```bash
docker buildx use cmsingbox-builder
```

登录 Docker Hub 时建议使用访问令牌，不要把密码写进脚本或 shell 历史：

```bash
printf '%s' "$DOCKERHUB_TOKEN" | docker login -u darkver8 --password-stdin
```

先只构建并推送带版本号的多架构镜像：

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -f Dockerfile.release \
  -t "darkver8/cmsingbox:$VERSION" \
  --push .
```

检查镜像清单：

```bash
docker buildx imagetools inspect "darkver8/cmsingbox:$VERSION"
```

在测试机验证版本标签后，再让 `latest` 指向完全相同的多架构镜像：

```bash
docker buildx imagetools create \
  -t darkver8/cmsingbox:latest \
  "darkver8/cmsingbox:$VERSION"
docker buildx imagetools inspect darkver8/cmsingbox:latest
```

只有测试通过的版本才覆盖 `latest`。若想先测试单一架构，可用临时标签并加入 `--load`，不要推送 `latest`。

部署机更新镜像：

```bash
cd /opt/cmsingbox-docker
docker compose pull
docker compose up -d
docker compose logs --tail=100 cmsingbox
```

容器的 `/data` 挂载目录必须保留；镜像更新不应删除该目录。

## 10. 单独重建授权中心

修改 `cmd/license-server`、`cmd/license-tool` 或 `internal/licensing` 后，先测试：

```bash
go test ./cmd/license-server ./internal/licensing
go build -trimpath -o /tmp/cmsingbox-license-server ./cmd/license-server
go build -trimpath -o /tmp/cmsingbox-license-tool ./cmd/license-tool
```

### 原生授权中心

使用安装脚本的“只更新程序”模式，不生成或替换密钥：

```bash
sudo env CMSINGBOX_LICENSE_INSTALL_ONLY=1 sh deploy/license/install-native.sh
sudo systemctl daemon-reload
sudo systemctl restart cmsingbox-license
sudo systemctl status cmsingbox-license --no-pager
sudo journalctl -u cmsingbox-license -n 100 --no-pager
```

该模式会替换 `/opt/cmsingbox-license` 中的程序，但不会修改 `/var/lib/cmsingbox-license/private.key`、`public.key` 和审计日志。

### Docker 授权中心

```bash
docker compose -f deploy/license/docker-compose.yml build --no-cache
docker compose -f deploy/license/docker-compose.yml up -d
docker compose -f deploy/license/docker-compose.yml logs --tail=100
```

`deploy/license/license-data` 必须继续挂载，绝不能在更新镜像时删除。更新前仍应离线备份 `private.key`、`public.key` 和 `license-audit.jsonl`。

## 11. 提交与发布建议

修改前建立分支：

```bash
git switch -c feature/功能名称
```

提交前检查：

```bash
git status --short
git diff --check
go test ./...
cd web && pnpm run build && cd ..
```

另外运行 `cd web && pnpm run lint` 查看代码质量问题；至少不能因为本次修改新增 ESLint 错误。

提交源码时不要强行加入被 `.gitignore` 排除的以下内容：

```text
private.key
license-private.key
license-data/
.env
CMSingBox-license-decryption-passphrase.txt
web/node_modules/
web/dist/
dist/
```

正式发布推荐顺序：

1. 提交源码并推送私有仓库。
2. 构建前端、运行全部测试。
3. 使用正式公钥构建三个 Linux 客户端。
4. 用临时数据目录完成客户端和授权测试。
5. 更新 `CM/bin` 并测试原生一键安装。
6. 构建带内核的多架构 Docker 镜像，先推版本标签。
7. 在测试机拉取版本标签验证 DNS、代理和控制台。
8. 验证通过后再更新 `latest` 和正式服务器。

## 12. 常见问题

### 修改页面后，二进制里还是旧界面

原因通常是 `web/dist` 没有重建。执行 `cd web && pnpm run build`，再回到仓库根目录重新运行 `build.sh`。

### 新二进制提示授权签名无效

依次比较授权端 `public.key`、构建时的 `LICENSE_PUBLIC_KEY`、原生 `/etc/cmsingbox.env` 或 Docker 环境变量。客户端使用的公钥必须与签发私钥严格配对。

### `go build` 提示 `pattern all:dist: no matching files found`

前端尚未构建。先执行：

```bash
cd web
pnpm install --frozen-lockfile
pnpm run build
cd ..
```

### Docker 镜像启动后没有 sing-box 内核

正式镜像必须使用 `Dockerfile.release`，并在构建前准备 `dist/sing-box-linux-*`。直接使用开发用 `Dockerfile` 只构建 CMSingBox 程序，不负责打包默认内核。

### 跨架构二进制无法在构建机运行

这是正常现象。amd64 机器不能直接运行 arm64/armv7 二进制。用 `file` 检查架构，并在对应真实设备或容器平台上测试。

### 更新后订阅和设置消失

说明启动时使用了新的数据目录。原生部署应继续使用 `/var/lib/cmsingbox`，Docker 应继续挂载原来的 `/data` 卷或宿主机数据目录。
