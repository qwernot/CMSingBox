import { useEffect, useState, type ReactNode } from 'react';
import { Button, Chip, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader } from '@nextui-org/react';
import { Activity, BarChart3, Box, Cpu, Filter, Globe2, HardDrive, MemoryStick, Network, Play, RefreshCw, Route, Server, Square, Wifi } from 'lucide-react';
import { useStore, type Subscription } from '../store';
import { configApi, monitorApi, serviceApi } from '../api';
import { toast } from '../components/Toast';

type DNSStats = { running?: boolean; stats?: { total?: number; success?: number; failed?: number; average_response_ms?: number } };
type Usage = { upload: number; download: number; session_upload: number; session_download: number; updated_at?: string; available: boolean };

function bytes(value = 0) {
  if (!Number.isFinite(value) || value < 0) return '0 B';
  const units = ['B', 'KB', 'GB', 'TB'];
  if (value < 1024) return `${Math.round(value)} B`;
  if (value < 1024 ** 2) return `${(value / 1024).toFixed(1)} KB`;
  if (value < 1024 ** 3) return `${(value / 1024 ** 2).toFixed(2)} MB`;
  let unit = 2;
  let scaled = value / 1024 ** 3;
  while (scaled >= 1024 && unit < units.length - 1) { scaled /= 1024; unit++; }
  return `${scaled.toFixed(2)} ${units[unit]}`;
}

const panel = 'app-panel rounded-2xl border border-slate-200/70 bg-white p-5 shadow-sm dark:border-white/5 dark:bg-[#1c1d21]';
const muted = 'text-slate-500 dark:text-zinc-400';

function Metric({ icon, label, children, tone = 'blue' }: { icon: ReactNode; label: string; children: ReactNode; tone?: 'blue' | 'purple' | 'green' | 'amber' }) {
  const colors = { blue: 'bg-blue-500/15 text-blue-500', purple: 'bg-purple-500/15 text-purple-500', green: 'bg-emerald-500/15 text-emerald-500', amber: 'bg-amber-500/15 text-amber-500' };
  return <div className={`${panel} flex min-w-0 items-center gap-4`}>
    <span className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl ${colors[tone]}`}>{icon}</span>
    <div className="min-w-0"><div className={`text-sm ${muted}`}>{label}</div><div className="truncate text-xl font-semibold text-slate-900 dark:text-white">{children}</div></div>
  </div>;
}

function Section({ icon, title, subtitle, children }: { icon: ReactNode; title: string; subtitle?: string; children: ReactNode }) {
  return <section className={panel}><div className="mb-5 flex items-center gap-3"><span className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/15 text-blue-500">{icon}</span><div><h2 className="font-semibold text-slate-900 dark:text-white">{title}</h2>{subtitle && <p className={`text-xs ${muted}`}>{subtitle}</p>}</div></div>{children}</section>;
}

function Row({ label, value }: { label: string; value: ReactNode }) {
  return <div className="flex min-w-0 items-center justify-between gap-3 py-2 text-sm"><span className={muted}>{label}</span><span className="max-w-[65%] truncate text-right font-medium text-slate-800 dark:text-zinc-200" title={typeof value === 'string' ? value : undefined}>{value}</span></div>;
}

function Bar({ label, value, tone = 'bg-blue-500' }: { label: string; value: number; tone?: string }) {
  const amount = Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0));
  return <div className="mb-3"><div className="mb-1.5 flex justify-between text-xs"><span className={muted}>{label}</span><span>{amount.toFixed(1)}%</span></div><div className="h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-white/5"><div className={`h-full rounded-full ${tone}`} style={{ width: `${amount}%` }} /></div></div>;
}

function SubscriptionTraffic({ sub }: { sub: Subscription }) {
  const traffic = sub.traffic;
  const used = traffic?.used || 0;
  const total = traffic?.total || 0;
  const knownSplit = !!(traffic?.upload || traffic?.download);
  return <div className="rounded-xl border border-slate-200/70 p-4 dark:border-white/5">
    <div className="mb-2 flex flex-wrap items-center justify-between gap-2"><div className="font-medium">{sub.name}</div><Chip size="sm" color={sub.enabled ? 'success' : 'default'} variant="flat">{sub.enabled ? '已启用' : '已停用'}</Chip></div>
    {traffic && total > 0 ? <>
      <div className="mb-1 flex justify-between text-sm"><span className={muted}>已用流量</span><span>{bytes(used)} / {bytes(total)}</span></div>
      <Bar label="套餐用量" value={used / total * 100} tone="bg-purple-500" />
      <div className={`flex flex-wrap gap-x-5 gap-y-1 text-xs ${muted}`}><span>剩余 {bytes(Math.max(0, traffic.remaining))}</span>{knownSplit && <><span>↑ {bytes(traffic.upload)}</span><span>↓ {bytes(traffic.download)}</span></>}</div>
      {!knownSplit && <p className={`mt-1 text-xs ${muted}`}>上传、下载明细将在下次刷新订阅后显示</p>}
    </> : <p className={`text-sm ${muted}`}>订阅源未提供流量配额信息</p>}
    {sub.expire_at && <p className={`mt-2 text-xs ${muted}`}>到期：{new Date(sub.expire_at).toLocaleString()}</p>}
  </div>;
}

export default function Dashboard() {
  const { serviceStatus, subscriptions, manualNodes, countryGroups, filters, rules, ruleGroups, systemInfo, fetchServiceStatus, fetchSubscriptions, fetchManualNodes, fetchCountryGroups, fetchFilters, fetchRules, fetchRuleGroups, fetchSystemInfo } = useStore();
  const [dns, setDNS] = useState<DNSStats | null>(null);
  const [usage, setUsage] = useState<Usage | null>(null);
  const [error, setError] = useState<{ title: string; message: string } | null>(null);
  const showError = (title: string, cause: unknown) => {
    const err = cause as { response?: { data?: { error?: string } }; message?: string };
    setError({ title, message: err?.response?.data?.error || err?.message || '操作失败' });
  };
  useEffect(() => {
    void fetchSubscriptions(); void fetchManualNodes(); void fetchCountryGroups(); void fetchFilters(); void fetchRules(); void fetchRuleGroups();
    const refresh = () => {
      void fetchServiceStatus(); void fetchSystemInfo();
      void monitorApi.dns().then(res => setDNS(res.data.data)).catch(() => setDNS(null));
      void monitorApi.traffic().then(res => setUsage(res.data.data)).catch(() => setUsage(null));
    };
    refresh();
    const timer = setInterval(refresh, 5000);
    return () => clearInterval(timer);
  }, []);
  const action = async (title: string, task: () => Promise<unknown>) => {
    try { await task(); await fetchServiceStatus(); toast.success(`${title}成功`); }
    catch (cause) { showError(`${title}失败`, cause); }
  };
  const host = systemInfo?.host;
  const totalNodes = subscriptions.reduce((sum, sub) => sum + sub.node_count, 0) + manualNodes.filter(node => node.enabled).length;
  const activeRules = rules.filter(rule => rule.enabled).length + ruleGroups.filter(group => group.enabled).length;
  const stats = dns?.stats;
  const rate = (n?: number) => `${bytes(n)}/s`;

  return <div className="space-y-5 pb-8">
    <div><p className="page-kicker">Overview</p><h1 className="mt-1 text-2xl font-semibold tracking-tight text-slate-900 dark:text-white sm:text-[28px]">仪表盘</h1><p className={`mt-1 text-sm ${muted}`}>设备、代理与订阅状态一览</p></div>
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Metric icon={<Server size={22} />} label="主机名">{host?.hostname || '-'}</Metric>
      <Metric icon={<Cpu size={22} />} label="CPU 使用率" tone="purple">{(host?.cpu_percent || 0).toFixed(1)}%</Metric>
      <Metric icon={<MemoryStick size={22} />} label="内存使用率" tone="green">{(host?.memory_percent || 0).toFixed(1)}%</Metric>
      <Metric icon={<Network size={22} />} label="网络传输" tone="amber"><span className="text-sm text-emerald-500">↑{rate(host?.network_up_bps)}</span> <span className="text-sm text-blue-500">↓{rate(host?.network_down_bps)}</span></Metric>
      <Metric icon={<Box size={22} />} label="总节点数">{totalNodes}</Metric>
      <Metric icon={<Wifi size={22} />} label="订阅数" tone="purple">{subscriptions.length}</Metric>
      <Metric icon={<Filter size={22} />} label="筛选数" tone="green">{filters.filter(filter => filter.enabled).length}</Metric>
      <Metric icon={<Route size={22} />} label="规则数" tone="amber">{activeRules}</Metric>
    </div>

    <div className="grid gap-4 lg:grid-cols-3">
      <Section icon={<Server size={21} />} title="设备信息" subtitle={host?.os || ''}>
        <Row label="主机名" value={host?.hostname || '-'} /><Row label="IP 地址" value={host?.ip_address || '-'} /><Row label="CPU" value={host?.cpu_model || '-'} />
      </Section>
      <Section icon={<HardDrive size={21} />} title="硬件信息" subtitle={host?.architecture || ''}>
        <Bar label="CPU 使用率" value={host?.cpu_percent || 0} /><Bar label="内存使用率" value={host?.memory_percent || 0} tone="bg-purple-500" /><Bar label="磁盘使用率" value={host?.disk_percent || 0} tone="bg-amber-500" />
      </Section>
      <Section icon={<Activity size={21} />} title="sing-box" subtitle={serviceStatus?.version?.match(/version\s+([\d.]+)/)?.[1] || serviceStatus?.version || '未安装'}>
        <Row label="运行状态" value={<span className={serviceStatus?.running ? 'text-emerald-500' : 'text-rose-500'}>{serviceStatus?.running ? '运行中' : '已停止'}</span>} />
        <Row label="进程 ID" value={serviceStatus?.pid || '-'} />
        <div className="mt-3 flex flex-wrap gap-2">
          {serviceStatus?.running ? <>
            <Button size="sm" variant="flat" color="primary" startContent={<RefreshCw size={15} />} onPress={() => void action('重启服务', serviceApi.restart)}>重启</Button>
            <Button size="sm" variant="flat" color="danger" startContent={<Square size={15} />} onPress={() => void action('停止服务', serviceApi.stop)}>停止</Button>
          </> : <Button size="sm" color="success" startContent={<Play size={15} />} onPress={() => void action('启动服务', serviceApi.start)}>启动</Button>}
          <Button size="sm" variant="flat" onPress={() => void action('应用配置', configApi.apply)}>应用配置</Button>
        </div>
      </Section>
    </div>

    <Section icon={<Filter size={21} />} title="筛选节点统计" subtitle={`${countryGroups.length} 个分组 · ${totalNodes} 个节点`}>
      {countryGroups.length ? <div className="grid gap-x-8 gap-y-1 sm:grid-cols-2 xl:grid-cols-3">{countryGroups.map(group => <Row key={group.code} label={`${group.emoji || '🌐'} ${group.name}`} value={<span className="rounded-full bg-blue-500/15 px-2.5 py-1 text-xs text-blue-500">{group.node_count} 个节点</span>} />)}</div> : <p className={`text-sm ${muted}`}>暂无节点分组</p>}
    </Section>

    <Section icon={<BarChart3 size={21} />} title="订阅流量" subtitle={`${subscriptions.length} 个订阅`}>
      {subscriptions.length ? <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">{subscriptions.map(sub => <SubscriptionTraffic key={sub.id} sub={sub} />)}</div> : <p className={`text-sm ${muted}`}>暂无订阅</p>}
    </Section>

    <Section icon={<Activity size={21} />} title="代理使用量" subtitle="CMSingBox 持久化记录 · 内核重启后累计值保留">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Metric icon={<Network size={20} />} label="累计上传" tone="green">{bytes(usage?.upload)}</Metric>
        <Metric icon={<Network size={20} />} label="累计下载">{bytes(usage?.download)}</Metric>
        <Metric icon={<RefreshCw size={20} />} label="本次内核上传" tone="amber">{bytes(usage?.session_upload)}</Metric>
        <Metric icon={<RefreshCw size={20} />} label="本次内核下载" tone="purple">{bytes(usage?.session_download)}</Metric>
      </div>
      <p className={`mt-4 text-xs ${muted}`}>约每 2 秒采样，适合查看趋势，不作为运营商计费依据。9090 面板自身仍显示本次内核会话；历史累计请以此处为准。{usage?.updated_at && `最近采样：${new Date(usage.updated_at).toLocaleString()}`}</p>
    </Section>

    <Section icon={<Globe2 size={21} />} title="DNS 监控" subtitle={dns?.running ? 'DNS 服务运行中' : 'DNS 服务未运行或未启用'}>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Metric icon={<BarChart3 size={20} />} label="总查询数">{stats?.total || 0}</Metric>
        <Metric icon={<Activity size={20} />} label="成功查询" tone="green">{stats?.success || 0}</Metric>
        <Metric icon={<Activity size={20} />} label="失败查询" tone="purple">{stats?.failed || 0}</Metric>
        <Metric icon={<RefreshCw size={20} />} label="平均响应" tone="amber">{(stats?.average_response_ms || 0).toFixed(1)} ms</Metric>
      </div>
    </Section>
    <Modal isOpen={!!error} onClose={() => setError(null)}><ModalContent><ModalHeader>{error?.title}</ModalHeader><ModalBody><p className="whitespace-pre-wrap">{error?.message}</p></ModalBody><ModalFooter><Button onPress={() => setError(null)}>关闭</Button></ModalFooter></ModalContent></Modal>
  </div>;
}
