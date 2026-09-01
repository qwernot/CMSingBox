import { useEffect, useMemo, useState } from 'react';
import {
  Activity, BookOpen, CheckCircle2, ChevronRight, CircleHelp, Copy, ExternalLink,
  FileKey2, Globe2, KeyRound, Menu, Monitor, Moon, Network, Server, Settings,
  ShieldCheck, Smartphone, Sun, Terminal, X,
} from 'lucide-react';

type DocsMode = 'user' | 'admin';
type Section = { id: string; label: string; icon: typeof BookOpen };

const userSections: Section[] = [
  { id: 'overview', label: '快速开始', icon: BookOpen },
  { id: 'deploy', label: '部署 CMSingBox', icon: Server },
  { id: 'lan', label: '局域网与 IP', icon: Network },
  { id: 'connection', label: '连接信息', icon: Network },
  { id: 'http', label: 'HTTP 代理', icon: Globe2 },
  { id: 'socks', label: 'SOCKS5 代理', icon: ShieldCheck },
  { id: 'mobile', label: '手机使用', icon: Smartphone },
  { id: 'desktop', label: '电脑使用', icon: Monitor },
  { id: 'troubleshooting', label: '常见问题', icon: CircleHelp },
];

const adminSections: Section[] = [
  { id: 'admin-start', label: '管理入门', icon: BookOpen },
  { id: 'admin-deploy', label: '部署 CMSingBox', icon: Server },
  { id: 'admin-lan', label: '局域网部署', icon: Network },
  { id: 'subscriptions', label: '订阅与节点', icon: Network },
  { id: 'ports', label: '端口与代理', icon: Settings },
  { id: 'dns', label: 'DNS 设置', icon: Globe2 },
  { id: 'operations', label: '维护与排障', icon: Terminal },
  { id: 'license-overview', label: '授权机制', icon: KeyRound },
  { id: 'license-deploy', label: '部署授权端', icon: Server },
  { id: 'license-issue', label: '签发授权', icon: FileKey2 },
  { id: 'license-release', label: '公钥与客户端打包', icon: Terminal },
];

function CodeBlock({ children }: { children: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(children);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };
  return <div className="group relative my-4 overflow-hidden rounded-2xl border border-slate-800 bg-[#07111f] shadow-xl shadow-slate-950/10">
    <button onClick={copy} className="absolute right-3 top-3 flex items-center gap-1 rounded-lg border border-white/10 bg-white/5 px-2 py-1 text-xs text-slate-400 hover:text-white"><Copy className="h-3.5 w-3.5" />{copied ? '已复制' : '复制'}</button>
    <pre className="overflow-x-auto px-5 py-5 pr-24 text-[13px] leading-6 text-cyan-100"><code>{children}</code></pre>
  </div>;
}

function Note({ children, kind = 'info' }: { children: React.ReactNode; kind?: 'info' | 'warning' | 'success' }) {
  const color = kind === 'warning' ? 'border-amber-300 bg-amber-50 text-amber-950 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100' : kind === 'success' ? 'border-emerald-300 bg-emerald-50 text-emerald-950 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-100' : 'border-blue-300 bg-blue-50 text-blue-950 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-100';
  return <div className={`my-5 rounded-2xl border px-5 py-4 text-sm leading-7 ${color}`}>{children}</div>;
}

function SectionTitle({ id, eyebrow, title, children }: { id: string; eyebrow: string; title: string; children?: React.ReactNode }) {
  return <section id={id} className="scroll-mt-24 border-b border-slate-200 py-10 last:border-0 dark:border-slate-800 sm:py-14">
    <p className="mb-2 text-sm font-semibold text-blue-600 dark:text-cyan-400">{eyebrow}</p>
    <h2 className="text-2xl font-bold tracking-tight text-slate-950 dark:text-white sm:text-3xl">{title}</h2>
    <div className="mt-5 text-[15px] leading-8 text-slate-600 dark:text-slate-300">{children}</div>
  </section>;
}

function UserContent() {
  return <>
    <SectionTitle id="overview" eyebrow="开始使用" title="几分钟接入 CMSingBox">
      <p>CMSingBox 将订阅中的节点汇总到一台服务器，并向局域网或公网设备提供 HTTP 与 SOCKS5 代理。使用前请先向管理员获取服务器 IP、代理端口以及认证信息。</p>
      <div className="mt-6 grid gap-4 sm:grid-cols-3">
        {['获取连接信息', '选择代理类型', '测试联网'].map((item, index) => <div key={item} className="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-800 dark:bg-slate-900"><span className="text-sm font-bold text-blue-600">0{index + 1}</span><p className="mt-2 font-semibold text-slate-900 dark:text-white">{item}</p></div>)}
      </div>
      <Note kind="success"><b>节点数不受授权限制。</b>授权仅控制后台最多可添加多少条订阅链接；普通用户连接代理时无需处理授权。</Note>
    </SectionTitle>
    <SectionTitle id="deploy" eyebrow="原生部署" title="一条命令安装 CMSingBox">
      <p>普通 Linux 服务器可直接安装原生程序和 sing-box 基础内核并由 systemd 管理，此命令<b>不经过 Docker，也不包含授权中心</b>。</p>
      <CodeBlock>{`curl -fsSL https://raw.githubusercontent.com/qwernot/CM/main/deploy/install.sh | sudo sh`}</CodeBlock>
      <p>安装完成后访问脚本输出的 <span className="font-mono">http://设备IP:9092</span>。初始账号和密码均为 <span className="font-mono">admin</span>，首次登录后必须立即修改密码。</p>
      <CodeBlock>{`# 查看运行状态和日志
systemctl status cmsingbox --no-pager
journalctl -u cmsingbox -f -n 100

# 更新：重新执行上面的一键安装命令`}</CodeBlock>
      <Note kind="warning">数据目录 <span className="font-mono">/var/lib/cmsingbox</span> 保存订阅、设置、许可证和 sing-box 文件。升级不会覆盖该目录；迁移或重装前仍应单独备份。</Note>
    </SectionTitle>
    <SectionTitle id="lan" eyebrow="小主机部署" title="局域网应该填写哪个 IP">
      <p>Docker 默认使用 macvlan，因此 CMSingBox 使用一个与小主机不同的独立局域网 IP。例如为容器预留 <span className="font-mono">192.168.1.20</span>：</p>
      <CodeBlock>{`管理后台：http://192.168.1.20:9092
HTTP 代理：192.168.1.20:2080
SOCKS5 代理：192.168.1.20:2080
DNS（启用后）：192.168.1.20:53`}</CodeBlock>
      <ol className="list-decimal space-y-3 pl-5"><li>选择与小主机相同网段、未被占用且位于 DHCP 自动分配范围之外的 IP。</li><li>客户端填写 CMSingBox 容器 IP，不填写小主机 IP、<span className="font-mono">127.0.0.1</span> 或 Docker 172.x 地址。</li><li>登录后台，在设置中开启“允许局域网访问”，代理监听才会对其他设备开放。</li><li>macvlan 默认会隔离宿主机与容器，这是 Docker 的正常行为，其他局域网设备仍可访问。</li></ol>
      <Note>网卡、网段和网关由脚本自动检测，也可通过环境变量覆盖。完整 Docker、ROS 和排障教程请查看 <a className="font-semibold text-blue-600 dark:text-cyan-300" href="https://qwernot.github.io/CM/">CMSingBox 部署文档</a>。</Note>
    </SectionTitle>
    <SectionTitle id="connection" eyebrow="准备信息" title="连接前需要四项信息">
      <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800"><table className="w-full min-w-[560px] text-left text-sm"><thead className="bg-slate-50 text-slate-900 dark:bg-slate-900 dark:text-white"><tr><th className="p-4">项目</th><th className="p-4">示例</th><th className="p-4">说明</th></tr></thead><tbody className="divide-y divide-slate-200 dark:divide-slate-800"><tr><td className="p-4">服务器地址</td><td className="p-4 font-mono">192.168.1.20</td><td className="p-4">局域网用小主机 IP，外网用公网 IP 或域名；不要填写 127.0.0.1</td></tr><tr><td className="p-4">代理端口</td><td className="p-4 font-mono">2080</td><td className="p-4">以管理后台显示为准</td></tr><tr><td className="p-4">用户名 / 密码</td><td className="p-4">由管理员提供</td><td className="p-4">若关闭认证，可留空</td></tr><tr><td className="p-4">代理协议</td><td className="p-4">HTTP 或 SOCKS5</td><td className="p-4">同一混合端口均可使用</td></tr></tbody></table></div>
      <Note kind="warning">后台登录密码与代理认证密码是两套独立凭据。不要把后台管理员密码填写到代理客户端，除非管理员明确设置成相同内容。</Note>
    </SectionTitle>
    <SectionTitle id="http" eyebrow="通用接入" title="使用 HTTP 代理">
      <p>在系统或应用的“HTTP 代理”设置中填写服务器 IP 和端口。开启认证时，再填写代理用户名和密码。</p>
      <CodeBlock>{`服务器：123.57.254.92\n端口：2080\n类型：HTTP\n认证：按管理员提供的信息填写`}</CodeBlock>
      <p>命令行可用下面的方式验证。出现外网 IP 即表示代理可用：</p>
      <CodeBlock>{`curl -x http://用户名:密码@123.57.254.92:2080 https://api.ipify.org`}</CodeBlock>
    </SectionTitle>
    <SectionTitle id="socks" eyebrow="通用接入" title="使用 SOCKS5 代理">
      <p>支持 SOCKS5 的客户端可直接使用相同的服务器地址和混合端口。建议选择“通过代理解析 DNS”或使用 <span className="font-mono">socks5h</span>，避免本地 DNS 泄漏。</p>
      <CodeBlock>{`curl -x socks5h://用户名:密码@123.57.254.92:2080 https://api.ipify.org`}</CodeBlock>
    </SectionTitle>
    <SectionTitle id="mobile" eyebrow="移动设备" title="iOS 与 Android">
      <ol className="list-decimal space-y-3 pl-5"><li>打开支持 HTTP 或 SOCKS5 的代理客户端。</li><li>新建手动代理，服务器填写公网 IP，端口填写后台显示的代理端口。</li><li>按管理员要求开启或关闭用户名密码认证。</li><li>保存后启用配置，再用浏览器访问网页测试。</li></ol>
      <Note>手机切换 Wi-Fi 与移动网络后公网出口可能变化。如服务器防火墙设置了来源 IP 白名单，需要管理员同步放行。</Note>
    </SectionTitle>
    <SectionTitle id="desktop" eyebrow="桌面设备" title="Windows 与 macOS">
      <p>可在操作系统网络设置中配置 HTTP 代理，也可在浏览器、下载器或开发工具中单独配置。SOCKS5 通常需要应用自身支持或使用代理客户端。</p>
      <div className="mt-5 grid gap-4 sm:grid-cols-2"><div className="rounded-2xl bg-slate-100 p-5 dark:bg-slate-900"><b className="text-slate-900 dark:text-white">Windows</b><p className="mt-2">设置 → 网络和 Internet → 代理 → 手动设置代理。</p></div><div className="rounded-2xl bg-slate-100 p-5 dark:bg-slate-900"><b className="text-slate-900 dark:text-white">macOS</b><p className="mt-2">系统设置 → 网络 → 当前网络 → 详细信息 → 代理。</p></div></div>
    </SectionTitle>
    <SectionTitle id="troubleshooting" eyebrow="自助排查" title="常见问题">
      <div className="space-y-5"><div><b className="text-slate-900 dark:text-white">无法连接服务器</b><p>确认填写的是服务器公网 IP，不是 127.0.0.1；检查端口、安全组和系统防火墙。</p></div><div><b className="text-slate-900 dark:text-white">返回 407 Proxy Authentication Required</b><p>代理认证已开启但凭据为空或错误，请向管理员确认代理用户名和密码。</p></div><div><b className="text-slate-900 dark:text-white">能连接但打不开网站</b><p>可能是订阅未更新、节点不可用或 DNS 异常，请让管理员在控制台查看节点和 sing-box 日志。</p></div></div>
    </SectionTitle>
  </>;
}

function AdminContent() {
  return <>
    <SectionTitle id="admin-start" eyebrow="管理入门" title="登录与首次安全配置">
      <p>通过 <span className="font-mono">http://服务器IP:9092</span> 打开 CMSingBox。首次登录后应立即在“设置 → 登录安全”中修改后台密码，并妥善保存。</p>
      <Note kind="warning">后台管理端口不建议完全暴露到互联网。请使用安全组白名单、VPN、反向代理 HTTPS 或防火墙限制来源地址。</Note>
    </SectionTitle>
    <SectionTitle id="admin-deploy" eyebrow="Docker 部署" title="安装、更新和停止主程序">
      <p>主程序默认使用 macvlan 独立局域网 IP，适合 Linux 服务器、NAS 和局域网小主机。部署包已带 sing-box 基础内核，后续仍可在后台更新。主程序与授权中心完全分开，普通客户只需要执行下面这一条：</p>
      <CodeBlock>{`curl -fsSL https://raw.githubusercontent.com/qwernot/CM/main/deploy/install-docker.sh | sudo env CMSINGBOX_IP=192.168.1.20 sh`}</CodeBlock>
      <p>默认安装到 <span className="font-mono">/opt/cmsingbox-docker</span>，持久数据位于其中的 <span className="font-mono">data</span>。再次执行同一命令会更新程序、重新构建容器并保留数据。</p>
      <CodeBlock>{`# 日常管理
cd /opt/cmsingbox-docker
docker compose ps
docker compose restart
docker compose logs -f --tail=100

# 停止但保留数据
docker compose down`}</CodeBlock>
      <Note kind="warning">不要删除 data 目录，否则配置和已激活许可证会丢失。</Note>
    </SectionTitle>
    <SectionTitle id="admin-lan" eyebrow="网络规划" title="局域网小主机与 Docker IP">
      <p>CMSingBox 默认使用 macvlan 独立 IP，使 DNS 53、代理 2080 和后台 9092 都由容器独占，不与小主机已有服务抢端口。部署前必须按现场网络选择一个 DHCP 范围外的空闲地址。</p>
      <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800"><table className="w-full min-w-[600px] text-left text-sm"><thead className="bg-slate-50 dark:bg-slate-900"><tr><th className="p-4">项目</th><th className="p-4">示例</th><th className="p-4">要求</th></tr></thead><tbody className="divide-y divide-slate-200 dark:divide-slate-800"><tr><td className="p-4">小主机地址</td><td className="p-4 font-mono">192.168.1.232</td><td className="p-4">保持原有地址，不用于客户端连接</td></tr><tr><td className="p-4">CMSingBox 独立地址</td><td className="p-4 font-mono">192.168.1.20</td><td className="p-4">与小主机同网段、位于 DHCP 池外</td></tr><tr><td className="p-4">客户端代理地址</td><td className="p-4 font-mono">192.168.1.20:2080</td><td className="p-4">填写容器独立地址并开启局域网访问</td></tr></tbody></table></div>
      <Note kind="warning">macvlan 模式不会与宿主机 53 端口冲突，但宿主机默认无法直接访问 macvlan 容器。不要把 CMSingBox IP 放进路由器 DHCP 自动分配池，以免发生地址冲突。</Note>
    </SectionTitle>
    <SectionTitle id="subscriptions" eyebrow="核心功能" title="订阅、节点与规则">
      <ol className="list-decimal space-y-3 pl-5"><li>进入“订阅管理”，添加订阅名称和 URL。</li><li>手动更新或等待定时任务拉取节点。</li><li>在“规则管理”配置域名、IP 或分流规则。</li><li>保存后由 CMSingBox 生成配置，并让 sing-box 自动重载。</li></ol>
      <Note><b>授权限制的是订阅链接数量。</b>未授权最多添加 1 条订阅链接；授权后按许可证额度添加。每条订阅中的节点数以及手动节点数不限制。</Note>
    </SectionTitle>
    <SectionTitle id="ports" eyebrow="网络服务" title="端口与 HTTP/SOCKS 代理">
      <div className="overflow-x-auto rounded-2xl border border-slate-200 dark:border-slate-800"><table className="w-full min-w-[560px] text-left text-sm"><thead className="bg-slate-50 dark:bg-slate-900"><tr><th className="p-4">端口</th><th className="p-4">用途</th><th className="p-4">建议</th></tr></thead><tbody className="divide-y divide-slate-200 dark:divide-slate-800"><tr><td className="p-4 font-mono">9092/TCP</td><td className="p-4">CMSingBox 管理后台</td><td className="p-4">限制来源 IP</td></tr><tr><td className="p-4 font-mono">2080/TCP</td><td className="p-4">HTTP/SOCKS5 混合代理</td><td className="p-4">按需开放并建议启用认证</td></tr><tr><td className="p-4 font-mono">53/UDP,TCP</td><td className="p-4">DNS 监听</td><td className="p-4">仅对受信网络开放</td></tr><tr><td className="p-4 font-mono">9093/TCP</td><td className="p-4">授权签发后台</td><td className="p-4">只允许管理员访问</td></tr></tbody></table></div>
      <p className="mt-5">代理认证可以在设置中关闭；若服务暴露到公网，强烈建议开启。代理用户名和密码可单独修改，不依赖后台登录账号。</p>
    </SectionTitle>
    <SectionTitle id="dns" eyebrow="DNS 服务" title="监听地址和端口">
      <p>DNS 默认监听 <span className="font-mono">0.0.0.0:53</span>。后台可分别修改监听 IP 和端口；修改后保存设置，并观察 sing-box 服务是否正常运行。</p>
      <Note kind="warning">53 是特权端口，也可能被 systemd-resolved、dnsmasq 或其他 DNS 服务占用。若启动失败，请先用 <span className="font-mono">ss -lntup | grep ':53'</span> 检查冲突。</Note>
    </SectionTitle>
    <SectionTitle id="operations" eyebrow="日常维护" title="备份、升级与排障">
      <p>升级前先在设置页导出备份。出现异常时依次检查 CMSingBox、sing-box 服务状态和最近日志。</p>
      <CodeBlock>{`systemctl status cmsingbox --no-pager\nsystemctl status sing-box --no-pager\njournalctl -u cmsingbox -n 100 --no-pager\njournalctl -u sing-box -n 100 --no-pager`}</CodeBlock>
      <p>配置修改后若 sing-box 反复重启，优先检查端口占用、订阅节点字段兼容性和生成配置的语法。</p>
    </SectionTitle>
    <SectionTitle id="license-overview" eyebrow="私有授权" title="授权机制说明">
      <p>CMSingBox 使用离线签名许可证。被授权服务器显示设备码，授权端使用私钥签发授权码，主程序只内置公钥进行验签，因此主服务器不需要连接授权端。</p>
      <div className="mt-6 grid gap-4 sm:grid-cols-3">{[['1', '复制设备码'], ['2', '授权端签发'], ['3', '主后台激活']].map(([n, label]) => <div key={n} className="rounded-2xl border border-slate-200 p-5 dark:border-slate-800"><span className="grid h-8 w-8 place-items-center rounded-full bg-blue-600 text-sm font-bold text-white">{n}</span><p className="mt-3 font-semibold text-slate-900 dark:text-white">{label}</p></div>)}</div>
      <Note kind="success">许可证只限制可添加的订阅链接数量，不限制订阅内节点数。授权端和私钥应由项目持有者独立保管。</Note>
    </SectionTitle>
    <SectionTitle id="license-deploy" eyebrow="授权端" title="在服务器部署授权服务">
      <p>授权中心是你持有的总签发端，可以给任意 CMSingBox 机器签发绑定设备码的许可证。它不包含在普通客户安装命令中，只在授权管理员自己的服务器执行：</p>
      <CodeBlock>{`git clone git@github.com:qwernot/CMSingBox.git
cd CMSingBox
sudo env CMSINGBOX_LICENSE_PASSWORD='Aa666333' sh deploy/license/install.sh`}</CodeBlock>
      <p>Docker 版默认监听 9093，私钥和审计日志保存在源码目录的 <span className="font-mono">deploy/license/license-data</span>。首次部署会生成密钥，以后更新会保留原私钥。下面保留 systemd 手工部署参数供高级维护使用。</p>
      <CodeBlock>{`# 生成登录密码哈希\nprintf '%s' '你的授权端密码' | sha256sum\n\n# /etc/cmsingbox-license.env（权限 0600）\nCMSINGBOX_LICENSE_PASSWORD_HASH=<上一步的64位哈希>\n\n# /etc/systemd/system/cmsingbox-license.service 中的启动命令\nExecStart=/opt/cmsingbox-license/cmsingbox-license-server \\\n+  -listen 0.0.0.0:9093 \\\n+  -private-key /var/lib/cmsingbox-license/private.key \\\n+  -audit /var/lib/cmsingbox-license/license-audit.jsonl\n\n# 启动并设置开机自启\nsystemctl daemon-reload\nsystemctl enable --now cmsingbox-license\nsystemctl status cmsingbox-license --no-pager`}</CodeBlock>
      <Note kind="warning">私钥决定全部许可证是否有效，必须离线备份且绝不能提交 GitHub。迁移已有总授权端时应复制原私钥，而不是重新生成。建议 9093 仅对白名单 IP 开放并配置 HTTPS。</Note>
      <p>授权端不要求与客户机器在同一服务器或同一网络。客户只需把六位设备码发给你，客户 CMSingBox 不需要联网访问授权端。</p>
    </SectionTitle>
    <SectionTitle id="license-issue" eyebrow="授权操作" title="生成与交付授权码">
      <ol className="list-decimal space-y-3 pl-5"><li>访问 <span className="font-mono">http://服务器IP:9093</span> 并使用授权管理员密码登录。</li><li>填写客户设备码、允许的订阅链接数量以及有效期。</li><li>生成授权码后复制给客户，不需要把私钥或授权端账号交给客户。</li><li>客户在 CMSingBox“设置 → 软件授权”中粘贴并激活。</li></ol>
      <p className="mt-5">修改授权端密码后重新执行部署脚本，或在授权部署目录中重启容器：</p>
      <CodeBlock>{`cd deploy/license\ndocker compose restart\ncurl http://127.0.0.1:9093/healthz`}</CodeBlock>
    </SectionTitle>
    <SectionTitle id="license-release" eyebrow="密钥轮换" title="同步公钥并重新打包客户端">
      <p>授权端只保存私钥，客户端只需要公钥。需要协助打包时只能提供 <span className="font-mono">public.key</span> 的一行 Base64 内容，绝不能发送 <span className="font-mono">private.key</span>。</p>
      <CodeBlock>{`# 在授权端读取并校验公钥
cd /root/CMSingBox/deploy/license
sudo tr -d '\\r\\n' < license-data/public.key; echo
test "$(sudo base64 -d license-data/public.key | wc -c)" -eq 32 && echo "公钥格式正确"`}</CodeBlock>
      <p>客户端优先使用环境变量 <span className="font-mono">CMSINGBOX_LICENSE_PUBLIC_KEY</span>，没有设置时才使用编译进二进制的公钥。因此发布时必须同时更新二进制、原生安装脚本、Docker 安装脚本和 Compose 默认值。</p>
      <CodeBlock>{`cd /path/to/CMSingBox
export LICENSE_PUBLIC_KEY='新的 Base64 公钥'
export FREE_SUBSCRIPTION_LIMIT='1'
export VERSION='1.0.10'

cd web && pnpm install --frozen-lockfile && pnpm run build && cd ..
SKIP_FRONTEND=1 ./build.sh linux

# 输出文件
ls -lh dist/cmsingbox-linux-amd64 \\
  dist/cmsingbox-linux-arm64 \\
  dist/cmsingbox-linux-arm`}</CodeBlock>
      <p>把三个客户端二进制复制到公开 CM 仓库的 <span className="font-mono">bin/</span>，但不得复制授权端、私钥或签发工具。随后重新执行客户使用的安装命令，脚本会保留数据并写入新公钥。</p>
      <Note kind="warning">重新生成密钥后，旧私钥签发的全部授权都会失效，必须给现有客户重新签发。正式私钥至少保留两份离线备份；授权端迁移时恢复原私钥，不要重新 keygen。</Note>
      <p>更完整的文件清单、发布检查和故障排查见私有仓库 <span className="font-mono">docs/admin/licensing.md</span>。</p>
    </SectionTitle>
  </>;
}

export default function Docs({ mode }: { mode: DocsMode }) {
  const [dark, setDark] = useState(() => localStorage.getItem('cmsingbox-docs-theme') !== 'light');
  const [menuOpen, setMenuOpen] = useState(false);
  const sections = useMemo(() => mode === 'admin' ? adminSections : userSections, [mode]);

  useEffect(() => {
    document.title = `${mode === 'admin' ? '管理员文档' : '用户文档'} | CMSingBox`;
    localStorage.setItem('cmsingbox-docs-theme', dark ? 'dark' : 'light');
  }, [dark, mode]);

  const sidebar = <nav className="space-y-1">
    <p className="mb-3 px-3 text-xs font-bold uppercase tracking-[.18em] text-slate-400">{mode === 'admin' ? '管理员指南' : '使用指南'}</p>
    {sections.map(({ id, label, icon: Icon }) => <a key={id} href={`#${id}`} onClick={() => setMenuOpen(false)} className="group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-slate-600 transition hover:bg-blue-50 hover:text-blue-700 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-cyan-300"><Icon className="h-4 w-4" />{label}<ChevronRight className="ml-auto h-3.5 w-3.5 opacity-0 transition group-hover:opacity-100" /></a>)}
  </nav>;

  return <div className={dark ? 'dark' : ''}>
    <div className="min-h-screen bg-white text-slate-700 transition-colors dark:bg-[#07101d] dark:text-slate-300">
      <header className="fixed inset-x-0 top-0 z-40 border-b border-slate-200/80 bg-white/85 backdrop-blur-xl dark:border-white/10 dark:bg-[#07101d]/85">
        <div className="mx-auto flex h-16 max-w-[1500px] items-center gap-3 px-4 sm:px-6">
          <button aria-label="打开目录" onClick={() => setMenuOpen(true)} className="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10 lg:hidden"><Menu className="h-5 w-5" /></button>
          <a href={mode === 'admin' ? '/docs/admin' : '/docs/user'} className="flex items-center gap-3 font-bold text-slate-950 dark:text-white"><span className="grid h-9 w-9 place-items-center rounded-xl bg-gradient-to-br from-blue-600 to-cyan-400 shadow-lg shadow-blue-500/20"><Activity className="h-5 w-5 text-white" /></span><span>CMSingBox <em className="ml-1 not-italic text-blue-600 dark:text-cyan-400">Docs</em></span></a>
          <nav className="ml-auto hidden items-center gap-1 sm:flex"><a href="/docs/user" className={`rounded-lg px-3 py-2 text-sm ${mode === 'user' ? 'bg-blue-50 font-semibold text-blue-700 dark:bg-white/10 dark:text-cyan-300' : ''}`}>用户文档</a><a href="/docs/admin" className={`rounded-lg px-3 py-2 text-sm ${mode === 'admin' ? 'bg-blue-50 font-semibold text-blue-700 dark:bg-white/10 dark:text-cyan-300' : ''}`}>管理员文档</a><a href="/" className="ml-1 flex items-center gap-1 rounded-lg px-3 py-2 text-sm hover:bg-slate-100 dark:hover:bg-white/10">进入控制台 <ExternalLink className="h-3.5 w-3.5" /></a></nav>
          <button aria-label="切换主题" onClick={() => setDark(!dark)} className="ml-auto rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10 sm:ml-1">{dark ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}</button>
        </div>
      </header>

      {menuOpen && <div className="fixed inset-0 z-50 lg:hidden"><button aria-label="关闭目录遮罩" className="absolute inset-0 bg-slate-950/60 backdrop-blur-sm" onClick={() => setMenuOpen(false)} /><aside className="absolute inset-y-0 left-0 w-[290px] overflow-y-auto bg-white p-5 shadow-2xl dark:bg-slate-950"><div className="mb-8 flex items-center justify-between font-bold text-slate-950 dark:text-white">文档目录<button aria-label="关闭目录" onClick={() => setMenuOpen(false)} className="rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-white/10"><X className="h-5 w-5" /></button></div>{sidebar}<div className="mt-8 space-y-2 border-t border-slate-200 pt-5 dark:border-slate-800"><a href="/docs/user" className="block rounded-lg px-3 py-2 text-sm">用户文档</a><a href="/docs/admin" className="block rounded-lg px-3 py-2 text-sm">管理员文档</a><a href="/" className="block rounded-lg px-3 py-2 text-sm">进入控制台</a></div></aside></div>}

      <aside className="fixed bottom-0 left-0 top-16 hidden w-64 overflow-y-auto border-r border-slate-200 bg-slate-50/50 px-5 py-8 dark:border-slate-800 dark:bg-slate-950/30 lg:block">{sidebar}</aside>
      <main className="pt-16 lg:pl-64">
        <div className="border-b border-slate-200 bg-[radial-gradient(circle_at_80%_10%,rgba(56,189,248,.16),transparent_30%),linear-gradient(135deg,rgba(37,99,235,.08),transparent_55%)] dark:border-slate-800">
          <div className="mx-auto max-w-4xl px-5 py-14 sm:px-8 sm:py-20">
            <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-blue-200 bg-blue-50 px-3 py-1 text-xs font-semibold text-blue-700 dark:border-cyan-400/20 dark:bg-cyan-400/10 dark:text-cyan-300"><CheckCircle2 className="h-3.5 w-3.5" />CMSingBox 官方文档</div>
            <h1 className="max-w-3xl text-4xl font-black tracking-tight text-slate-950 dark:text-white sm:text-5xl">{mode === 'admin' ? '管理员部署与运维指南' : '轻松连接你的代理服务'}</h1>
            <p className="mt-5 max-w-2xl text-base leading-8 text-slate-600 dark:text-slate-300 sm:text-lg">{mode === 'admin' ? '从订阅管理、代理端口、DNS 到私有授权签发，一份覆盖 CMSingBox 日常管理的完整指南。' : '了解 HTTP、SOCKS5 的连接方式，并在手机与电脑上快速开始使用 CMSingBox。'}</p>
            <a href={`#${sections[0].id}`} className="mt-7 inline-flex items-center gap-2 rounded-xl bg-blue-600 px-5 py-3 text-sm font-semibold text-white shadow-lg shadow-blue-600/20 hover:bg-blue-500">开始阅读 <ChevronRight className="h-4 w-4" /></a>
          </div>
        </div>
        <article className="mx-auto max-w-4xl px-5 sm:px-8">{mode === 'admin' ? <AdminContent /> : <UserContent />}</article>
        <footer className="border-t border-slate-200 px-5 py-8 text-center text-xs text-slate-400 dark:border-slate-800">CMSingBox · Private Network Console</footer>
      </main>
    </div>
  </div>;
}
