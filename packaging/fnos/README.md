# CMSingBox 飞牛 fnOS FPK

本目录用于生成可在“飞牛应用中心 → 手动安装”中上传的 CMSingBox FPK。

## 工作方式

飞牛系统会占用宿主机的 DNS 53 端口，因此 FPK 不使用宿主机网络。安装程序会自动复用同网段的 Docker macvlan/ipvlan 网络；找不到可复用网络时才创建专用 macvlan，让 CMSingBox 容器获得独立局域网 IP：

```text
飞牛 NAS：原来的局域网 IP，继续使用飞牛自己的 53 端口
CMSingBox：独立局域网 IP，例如 192.168.1.20，使用自己的 53/2080/9090/80
```

FPK 本身与 CPU 架构无关，Docker 会自动拉取 `darkver8/cmsingbox:latest` 中与飞牛 NAS 对应的 amd64、arm64 或 arm/v7 镜像。镜像已经包含默认 sing-box 内核。

## 构建 FPK

```bash
cd /path/to/CMSingBox
VERSION=1.1.2 sh packaging/fnos/build-fpk.sh
```

输出：

```text
dist/fpk/CMSingBox-fnOS-1.1.2-all.fpk
dist/fpk/CMSingBox-fnOS-1.1.2-all.fpk.sha256
```

脚本没有检测到 `fnpack` 时，会从飞牛官方地址临时下载 `fnpack 1.2.3`。FPK 很小，第一次安装时飞牛需要联网拉取 CMSingBox 多架构镜像。

## 手动安装

1. 在飞牛中确认 Docker 已启用。
2. 查询飞牛的局域网网卡名称，例如 `eth0`、`enp3s0` 或 `bond0`。
3. 在主路由 DHCP 地址池之外准备一个空闲 IP，例如 `192.168.1.20`。
4. 打开“应用中心 → 手动安装”，上传 `CMSingBox-fnOS-版本-all.fpk`。
5. 按向导填写独立 IP、飞牛 LAN 网卡、局域网网段和网关。
6. 安装完成后点击飞牛桌面的 CMSingBox 图标，页面会跳转到独立 IP 的 80 端口。

默认示例：

```text
CMSingBox 独立 IP：192.168.1.20
飞牛局域网网卡：eth0（必须改成实际网卡）
局域网网段：192.168.1.0/24
局域网网关：192.168.1.1
```

初始账号和密码均为 `admin`，首次登录后立即修改密码。

## DNS 与路由器

在 CMSingBox 后台启用独立 DNS，并保持监听 `0.0.0.0:53`。容器拥有独立网络命名空间，因此不会与飞牛宿主机的 53 冲突。手机或路由器 DHCP 下发的 DNS 应填写 CMSingBox 独立 IP。

CMSingBox 面板里的透明代理用于接管主路由转发来的流量：先保存开启状态并应用 sing-box 配置，再应用 nftables，最后配置主路由下一跳。旧版面板里的“后台服务”是手动二进制部署遗留入口；FPK 已由飞牛应用中心管理，不需要再次安装，当前版本已将该卡片移除。

DNS 只能完成域名分流；透明代理还需要主路由把 `198.18.0.0/15` 和纯 IP 目标静态路由到 CMSingBox 独立 IP。AdGuard Home 不是解决 53 冲突的必要条件。

## 数据与升级

数据保存在飞牛为应用分配的 `TRIM_PKGVAR/data`，升级 FPK 或镜像不会主动删除数据。卸载前仍应在 CMSingBox 后台导出备份。

镜像更新后，在飞牛应用中心重新安装新版 FPK 即可拉取并重建容器；更新前先备份数据。
