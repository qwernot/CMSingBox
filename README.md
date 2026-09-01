# CMSingBox

CMSingBox 是一个面向 sing-box 的中文可视化管理平台，支持订阅与节点、路由规则、HTTP/SOCKS5 代理、DNS、运行日志、配置备份以及离线授权管理。

## 部署文档

- [普通用户在线文档](https://qwernot.github.io/CM/)
- [原生一键部署](https://qwernot.github.io/CM/deploy/native.html)
- [Docker 独立 IP 部署](https://qwernot.github.io/CM/deploy/docker.html)
- [RouterOS 容器部署](https://qwernot.github.io/CM/deploy/routeros.html)
- [管理员与授权文档](docs/admin/README.md)

## 原生一键部署

普通 Linux 服务器使用原生 systemd 一键部署，不经过 Docker：

```bash
curl -fsSL https://raw.githubusercontent.com/qwernot/CM/main/deploy/install.sh | sudo sh
```

## Docker 独立 IP 部署

Docker 使用 macvlan，为 CMSingBox 分配独立局域网 IP，避免与宿主机的 DNS 53、代理 2080、后台 9092 端口冲突。

先在路由器 DHCP 自动分配范围之外准备一个空闲 IP，然后执行：

```bash
curl -fsSL https://raw.githubusercontent.com/qwernot/CM/main/deploy/install-docker.sh | sudo env CMSINGBOX_IP=192.168.1.20 sh
```

请把 `192.168.1.20` 改成实际准备给 CMSingBox 使用的地址。安装完成后访问 `http://CMSingBox局域网IP:9092`。

初始账号和密码均为 `admin`，首次登录后必须立即修改密码。

## 授权中心部署

授权中心与普通 CMSingBox 主程序完全分开，只由授权管理员部署：

```bash
git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
sudo env CMSINGBOX_LICENSE_PASSWORD='Aa666333' sh deploy/license/install.sh
```

授权私钥不得上传 GitHub，必须单独离线备份。普通客户不需要部署授权中心，只需要把六位设备码交给授权管理员。

## 数据目录

原生部署的数据目录为 `/var/lib/cmsingbox`；Docker 部署的数据目录为 `/opt/cmsingbox-docker/data`。更新不会删除这些目录；迁移、卸载或重装前仍应先在后台导出备份，并复制整个数据目录。

## 端口

| 端口 | 协议 | 用途 |
| --- | --- | --- |
| 9092 | TCP | CMSingBox 管理后台 |
| 2080 | TCP | HTTP/SOCKS5 混合代理 |
| 53 | TCP/UDP | DNS 服务，默认关闭 |
| 9093 | TCP | 独立授权中心，仅管理员部署 |

## 权利说明

CMSingBox 为私有持有软件。未经书面授权，不得复制、重新发布、出售、转授权或制作衍生发行版本。
