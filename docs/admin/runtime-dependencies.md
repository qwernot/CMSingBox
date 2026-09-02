# 运行时外部依赖审计

本页记录 CMSingBox 正常运行时会访问的外部地址。目标是：即使最初参考的项目、UI 站点或第三方规则仓库停止服务，已经部署的 CMSingBox 管理后台和代理服务仍可启动。

## 已内置、不依赖外站的内容

- 管理后台的 HTML、JavaScript、CSS、登录背景和 favicon 全部编译进 CMSingBox 二进制。
- 代理控制台 Zashboard 随公开安装包和 Docker 镜像一起安装。生成的 sing-box 配置不会再设置 `external_ui_download_url`，因此启动时不会向 Zashboard 的发布站点下载文件。
- 默认规则集镜像位于项目自有公开仓库 `qwernot/CM` 的 `rules/` 目录。旧版本保存的第三方规则仓库地址会在升级后自动迁移。
- Go 和前端 npm 开源库只在编译时使用；发布后的管理后台不会从 CDN 加载这些库。

## 仍会联网的功能

| 地址或类型 | 用途 | 失效时的影响 |
| --- | --- | --- |
| `github.com/SagerNet/sing-box` | 查询、下载用户主动选择的新 sing-box 内核 | 只影响在线更新；已安装内核继续工作 |
| `raw.githubusercontent.com/qwernot/CM` | 下载项目自有内核副本、规则集和公开安装文件 | 已下载文件继续可用；新安装或规则更新会失败 |
| 用户填写的订阅地址 | 拉取代理节点 | 该订阅无法更新；已有本地数据不会令管理站点崩溃 |
| 配置的 DoH 地址 | DNS 解析 | DNS 服务不可用会影响代理访问，可在设置中替换 |
| `www.gstatic.com/generate_204` | 节点健康检查 | 只影响测速结果，可在订阅设置中修改 |
| `666228.xyz` | 文档与购买入口 | 链接无法打开，不影响后台和代理核心 |

## 发布前检查

每次发布应确认：

1. `web/dist` 中不存在 CDN 脚本、字体或图片引用。
2. 生成配置中不存在 `external_ui_download_url`。
3. 默认规则地址指向 `qwernot/CM`，且公开仓库中的 `.srs` 文件校验通过。
4. 在断开公网或屏蔽参考站域名的条件下，登录页、设置页和已安装 sing-box 内核仍能启动。

注意：项目源码仍包含正常的 Go/npm 开源依赖声明，这些属于构建依赖。若希望在 GitHub、npm 和 Go 模块站全部不可用时仍能从零编译，应另外离线备份 Go module cache、npm cache、sing-box 内核与 Zashboard 压缩包。
