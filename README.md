# CMSingBox

## Docker 一键部署

CMSingBox 主程序（不包含授权中心）：

```bash
curl -fsSL https://raw.githubusercontent.com/qwernot/CMSingBox/main/install.sh | sudo sh
```

安装后访问 `http://小主机或服务器IP:9092`，初始账号与密码均为 `admin`。首次登录后请立即修改密码，并按需开启“允许局域网访问”。默认使用 Docker Host 网络，局域网设备直接填写小主机本身的 IP，无需填写 Docker IP。

授权中心只供授权管理员单独部署，不包含在普通安装命令中：

```bash
curl -fsSL https://raw.githubusercontent.com/qwernot/CMSingBox/main/install-license.sh | sudo env CMSINGBOX_LICENSE_PASSWORD='请改成至少8位强密码' sh
```

授权私钥保存在授权服务器的 `license-data` 目录，禁止提交仓库并必须离线备份。详见部署后的 `/docs/user` 与 `/docs/admin` 网页文档。

[English](#english) | [中文](#中文)

---

<a name="english"></a>

## English

A modern web-based management panel for [sing-box](https://github.com/SagerNet/sing-box), providing an intuitive interface to manage subscriptions, rules, filters, and more.

> This fork adds authenticated administration, configuration backup/restore, full host monitoring, DNS source-IP routing and observability, advanced Sing-box JSON extensions, client configuration links, and nftables TProxy management. See [Architecture and deployment](docs/architecture.md).

### Features

- **Subscription Management**
  - Support multiple formats: SS, VMess, VLESS, Trojan, Hysteria2, TUIC
  - Clash YAML and Base64 encoded subscriptions
  - Traffic statistics (used/remaining/total)
  - Expiration date tracking
  - Auto-refresh with configurable intervals

- **Node Management**
  - Auto-parse nodes from subscriptions
  - Manual node addition
  - Country grouping with emoji flags
  - Node filtering by keywords and countries

- **Rule Configuration**
  - Custom rules (domain, IP, port, geosite, geoip)
  - 13 preset rule groups (Ads, AI services, streaming, etc.)
  - Rule priority management
  - Rule set validation tool

- **Filter System**
  - Include/exclude by keywords
  - Country-based filtering
  - Proxy modes: URL-test (auto) / Select (manual)

- **DNS Management**
  - Multiple DNS protocols (UDP, DoT, DoH)
  - Custom hosts mapping
  - DNS routing rules

- **Service Control**
  - Start/Stop/Restart sing-box
  - Configuration hot-reload
  - Auto-apply on config changes
  - Process recovery on startup

- **System Monitoring**
  - Real-time CPU and memory usage
  - Application and sing-box logs
  - Service status dashboard

- **macOS Support**
  - launchd service integration
  - Auto-start on boot
  - Background daemon mode

- **Kernel Management**
  - Auto-download sing-box binary
  - Version checking and updates
  - Multi-platform support

### Screenshots

![Dashboard](docs/screenshots/dashbord.png)
![Subscriptions](docs/screenshots/subscriptions.png)
![Rules](docs/screenshots/rules.png)
![Settings](docs/screenshots/settings.png)
![Logs](docs/screenshots/log.png)

### Installation

#### Pre-built Binaries

Use the internally built CMSingBox release artifact.

#### Build from Source

```bash
# Clone the repository
cd CMSingBox

# Build for all platforms
./build.sh all

# Or build for current platform only
./build.sh current

# Output binaries are in ./build/
```

**Build Options:**
```bash
./build.sh all       # Build for all platforms (Linux/macOS x amd64/arm64)
./build.sh linux     # Build for Linux only
./build.sh darwin    # Build for macOS only
./build.sh current   # Build for current platform
./build.sh frontend  # Build frontend only
./build.sh clean     # Clean build directory
```

### Usage

```bash
# Basic usage
./cmsingbox

# Custom data directory and port
./cmsingbox -data ~/.cmsingbox -port 9090
```

**Command Line Options:**
| Option | Default | Description |
|--------|---------|-------------|
| `-data` | `~/.cmsingbox` | Data directory path |
| `-port` | `9090` | Web server port |

After starting, open `http://localhost:9090` in your browser.

### Configuration

**Data Directory Structure:**
```
~/.cmsingbox/
├── data.json           # Configuration data
├── generated/
│   └── config.json     # Generated sing-box config
├── bin/
│   └── sing-box        # sing-box binary
├── logs/
│   ├── sbm.log         # Application logs
│   └── singbox.log     # sing-box logs
└── singbox.pid         # PID file
```

### Tech Stack

- **Backend:** Go, Gin, gopsutil
- **Frontend:** React 19, TypeScript, NextUI, Tailwind CSS
- **Build:** Single binary with embedded frontend

### Requirements

- Go 1.21+ (for building)
- Node.js 18+ (for building frontend)
- sing-box (auto-downloaded or manual installation)

### License

Proprietary software. All rights reserved by the CMSingBox owner. No permission is granted to copy, redistribute, sublicense, or create derivative distributions without written authorization.

---

<a name="中文"></a>

## 中文

一个现代化的 [sing-box](https://github.com/SagerNet/sing-box) Web 管理面板，提供直观的界面来管理订阅、规则、过滤器等。

> 当前版本新增后台认证、备份恢复、完整主机监控、DNS 来源 IP 分流与监控、高级 Sing-box JSON 扩展、手机客户端配置链接和 nftables TProxy 管理。部署方式见 [架构与部署](docs/architecture.md)。

### 功能特性

- **订阅管理**
  - 支持多种格式：SS、VMess、VLESS、Trojan、Hysteria2、TUIC
  - 兼容 Clash YAML 和 Base64 编码订阅
  - 流量统计（已用/剩余/总量）
  - 过期时间追踪
  - 可配置间隔的自动刷新

- **节点管理**
  - 自动从订阅解析节点
  - 手动添加节点
  - 按国家分组（带 emoji 国旗）
  - 按关键字和国家过滤节点

- **规则配置**
  - 自定义规则（域名、IP、端口、geosite、geoip）
  - 13 个预设规则组（广告、AI 服务、流媒体等）
  - 规则优先级管理
  - 规则集验证工具

- **过滤器系统**
  - 按关键字包含/排除
  - 按国家过滤
  - 代理模式：自动测速 / 手动选择

- **DNS 管理**
  - 多种 DNS 协议（UDP、DoT、DoH）
  - 自定义 hosts 映射
  - DNS 路由规则

- **服务控制**
  - 启动/停止/重启 sing-box
  - 配置热重载
  - 配置变更后自动应用
  - 启动时自动恢复进程

- **系统监控**
  - 实时 CPU 和内存使用率
  - 应用和 sing-box 日志
  - 服务状态仪表盘

- **macOS 支持**
  - launchd 服务集成
  - 开机自启
  - 后台守护进程模式

- **内核管理**
  - 自动下载 sing-box 二进制文件
  - 版本检查和更新
  - 多平台支持

### 截图

![仪表盘](docs/screenshots/dashbord.png)
![订阅管理](docs/screenshots/subscriptions.png)
![规则配置](docs/screenshots/rules.png)
![设置](docs/screenshots/settings.png)
![日志](docs/screenshots/log.png)

### 安装

#### 预编译二进制文件

使用内部构建并签名的 CMSingBox 发布产物。

#### 从源码构建

```bash
# 克隆仓库
cd CMSingBox

# 构建所有平台
./build.sh all

# 或只构建当前平台
./build.sh current

# 输出文件在 ./build/ 目录
```

**构建选项：**
```bash
./build.sh all       # 构建所有平台（Linux/macOS x amd64/arm64）
./build.sh linux     # 仅构建 Linux
./build.sh darwin    # 仅构建 macOS
./build.sh current   # 仅构建当前平台
./build.sh frontend  # 仅构建前端
./build.sh clean     # 清理构建目录
```

### 使用方法

```bash
# 基本用法
./cmsingbox

# 自定义数据目录和端口
./cmsingbox -data ~/.cmsingbox -port 9090
```

**命令行参数：**
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-data` | `~/.cmsingbox` | 数据目录路径 |
| `-port` | `9090` | Web 服务端口 |

启动后，在浏览器中打开 `http://localhost:9090`。

### 配置

**数据目录结构：**
```
~/.cmsingbox/
├── data.json           # 配置数据
├── generated/
│   └── config.json     # 生成的 sing-box 配置
├── bin/
│   └── sing-box        # sing-box 二进制文件
├── logs/
│   ├── sbm.log         # 应用日志
│   └── singbox.log     # sing-box 日志
└── singbox.pid         # PID 文件
```

### 技术栈

- **后端：** Go、Gin、gopsutil
- **前端：** React 19、TypeScript、NextUI、Tailwind CSS
- **构建：** 单一二进制文件，内嵌前端

### 环境要求

- Go 1.21+（用于构建）
- Node.js 18+（用于构建前端）
- sing-box（可自动下载或手动安装）

### 授权与分发

本分支按私有、闭源产品维护，新增代码不对外授予开源许可。源自上游项目及其他第三方组件的代码仍分别遵循其原有许可证和版权声明；闭源分发时也必须保留这些法定声明。软件授权的生成、构建与签发流程见 [私有离线授权](docs/licensing.md)。
