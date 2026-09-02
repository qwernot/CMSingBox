# CMSingBox 管理员文档

本目录只面向 CMSingBox 项目管理员，包含系统架构、离线授权设计和总授权中心部署说明。普通用户部署与使用教程统一发布在 [CM 中文文档](https://qwernot.github.io/CM/) 中。

运行时外部地址及失效影响见 [运行时外部依赖审计](runtime-dependencies.md)。

## 目录

- [系统架构](architecture.md)
- [离线授权机制](licensing.md)
- [授权签发网站](license-server.md)
- [授权中心重装与原密钥恢复](license-redeploy.md)

## 总授权中心部署

授权端不包含在普通用户的一键部署中。先克隆私有源码仓库，再选择 Docker 或原生 systemd 方式单独部署。

### 原生 systemd 部署（不使用 Docker）

```bash
git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
sudo env CMSINGBOX_LICENSE_PASSWORD='Aa666333' sh deploy/license/install-native.sh
```

已有原公钥和原私钥、需要迁移或重装时，使用交互式恢复脚本：

```bash
sudo sh deploy/license/redeploy-native.sh
```

脚本会隐藏私钥输入、校验公私钥是否配对、自动备份旧密钥并在失败时回滚。完整说明见[授权中心重装与原密钥恢复](license-redeploy.md)。

使用私有仓库内的固定加密正式密钥包部署：

```bash
sudo sh deploy/license/install-formal-key.sh
```

该方式每次都恢复当前正式客户端对应的同一对密钥，但仍需输入单独离线保存的解密口令。

### Docker 部署

```bash
git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
sudo env CMSINGBOX_LICENSE_PASSWORD='Aa666333' sh deploy/license/install.sh
```

默认访问地址为 `http://授权服务器IP:9093`。一个总授权中心可以给任意机器的六位设备码签发许可证，不要求客户机器与授权端在同一服务器或同一局域网。

## 授权规则

- 未授权最多添加 1 条订阅链接。
- 授权只限制订阅链接数量，不限制订阅内节点数和手动节点数。
- 客户主程序只保存公钥并进行离线验签；私钥只保存在总授权中心。
- Docker 私钥位于 `deploy/license/license-data/private.key`，原生私钥位于 `/var/lib/cmsingbox-license/private.key`，必须离线备份，绝不能提交到 GitHub。
- 更换密钥、同步客户端公钥和重新打包二进制，请严格按照 [授权密钥、更换公钥与客户端打包](licensing.md) 操作。
