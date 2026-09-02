# 私有源码快照

本目录保存 CMSingBox 的版本化源码快照及 SHA-256 校验文件，用于在 Git 历史之外快速下载和恢复指定版本。

- 快照包含 Git 已跟踪的源码、脚本和管理员文档，并主动排除本机工具配置及运行缓存数据库。
- 快照不包含运行数据、订阅、密码、环境文件、授权审计记录或任何授权私钥。
- 恢复后仍需从离线加密备份单独恢复授权中心的 `private.key`。

校验示例：

```bash
sha256sum -c CMSingBox-v1.0.19-source.tar.gz.sha256
tar -xzf CMSingBox-v1.0.19-source.tar.gz
```
