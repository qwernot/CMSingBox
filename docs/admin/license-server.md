# CMSingBox 授权签发网站

`cmd/license-server` 是独立的离线授权签发后台，仅使用 Go 标准库。它读取 Ed25519 私钥，通过管理员登录页面签发绑定设备码的 CMSingBox 授权码，并记录不含授权码和私钥的 JSONL 审计日志。

## 安全原则

- 签发网站必须与客户使用的 CMSingBox 管理端分开部署。
- 私钥只放在签发服务器，权限设为 `0600`，不要放入 Git、Docker 镜像或前端代码。
- 没有域名和 HTTPS 时只监听 `127.0.0.1`，通过 SSH 隧道访问，禁止直接开放公网端口。
- 公网部署必须放在 HTTPS 反向代理后，并启用 `-secure-cookie`。
- 管理密码只保存 SHA-256 摘要；应使用随机生成的高强度密码。

## 构建

```bash
go build -trimpath -o cmsingbox-license-server ./cmd/license-server
```

## 生成管理密码

```bash
ADMIN_PASSWORD="$(openssl rand -hex 16)"
ADMIN_PASSWORD_HASH="$(printf '%s' "$ADMIN_PASSWORD" | sha256sum | awk '{print $1}')"
printf '管理员密码：%s\n密码摘要：%s\n' "$ADMIN_PASSWORD" "$ADMIN_PASSWORD_HASH"
```

密码摘要可以写入仅 root 可读的 systemd 环境文件，原始密码交给管理员后不在服务器明文保存。

## 私有模式启动

```bash
CMSINGBOX_LICENSE_PASSWORD_HASH="$ADMIN_PASSWORD_HASH" \
./cmsingbox-license-server \
  -listen 127.0.0.1:9093 \
  -private-key /secure/path/private.key \
  -audit /secure/path/license-audit.jsonl
```

管理员电脑建立 SSH 隧道：

```bash
ssh -L 9093:127.0.0.1:9093 root@签发服务器
```

随后访问 `http://127.0.0.1:9093/`。此端口只通过加密 SSH 隧道传输，不暴露公网。

## 页面功能

- 管理员密码登录，12 小时后自动失效。
- 会话 Cookie 使用 `HttpOnly` 和 `SameSite=Strict`。
- 所有签发、退出操作均校验 CSRF Token 和请求来源。
- 输入六位设备码、订阅链接额度、授权编号和可选到期日。
- 输出可复制的 `CMS1` 授权码。
- 审计日志记录签发时间、访问来源、授权编号、设备码、额度和到期时间，但不记录私钥、管理员密码或完整授权码。

## 公网模式

有域名和有效 HTTPS 证书后，可让签发服务继续监听回环地址，由 Caddy 或 Nginx 反向代理，并在启动参数加入：

```text
-secure-cookie
```

在完成 HTTPS、访问控制和备份方案前，不应把签发服务直接监听到 `0.0.0.0`。
