# CMSingBox 架构与部署

## 组件

- `sbm`：Go/Gin 管理服务，保存配置、生成 Sing-box JSON、管理进程和系统服务。
- Web UI：React/TypeScript 单页应用，生产构建嵌入 `sbm` 二进制。
- DNS Proxy：独立 UDP/TCP DNS 服务，按来源 IP/CIDR 在代理与直连上游之间分流，内置 TTL 缓存和最近 1000 条查询日志。
- Firewall Manager：生成并应用 nftables TProxy 规则及 Linux 策略路由。只有用户在界面中明确确认后才修改系统网络。
- Sing-box：由管理服务下载、生成配置并启动；Clash API 可连接 Zashboard。

## 安全默认值

- 初次登录为 `admin / admin`，部署后应立即在“设置 → 账户安全”修改。
- 管理 API 默认全部要求 HttpOnly 会话认证。
- 客户端订阅使用随机 24 位十六进制路径。
- 配置备份不包含后台密码和会话。
- DNS 服务与透明代理默认关闭；DNS 默认监听 `5353`，确认部署无冲突后可改为 `53`。
- 数据文件通过临时文件、`fsync`、原子替换保存，权限为 `0600`。

## Docker 部署

```bash
docker compose build
docker compose up -d
```

旁路由需要 `network_mode: host`、`NET_ADMIN`、`NET_RAW` 和 `/dev/net/tun`。打开 `http://旁路由IP:9090` 完成配置。

推荐顺序：

1. 修改后台密码。
2. 添加订阅、筛选组和规则，预览并应用 Sing-box 配置。
3. 配置代理/直连 DNS 上游，启用 DNS 服务；作为局域网 DNS 时将监听地址改为 `0.0.0.0:53`。
4. 启用透明代理配置、保存并确认 Sing-box 正常运行。
5. 预览 nftables 规则，最后点击“应用 nftables”。
6. 将主路由 DHCP 网关和 DNS 指向旁路由地址。

## 开发验证

```bash
go test ./...
cd web && pnpm install --frozen-lockfile && pnpm build
```

防火墙单元测试只验证规则生成，不修改开发机 nftables。DNS 集成测试使用临时 UDP 端口和本地模拟上游。
