import { useEffect, useMemo, useState } from 'react';
import { Card, CardBody, Chip, Input } from '@nextui-org/react';
import { CheckCircle, Clock, Database, Search, XCircle } from 'lucide-react';
import { monitorApi } from '../api';

interface DNSData {
  running: boolean;
  stats: { total: number; success: number; failed: number; cache_hits: number; average_response_ms: number };
  logs: Array<{ time: string; domain: string; type: string; upstream: string; source_ip: string; result: string; duration_ms: number; cached: boolean }>;
}

export default function DNSMonitor() {
  const [data, setData] = useState<DNSData | null>(null);
  const [query, setQuery] = useState('');
  useEffect(() => {
    const refresh = () => monitorApi.dns().then((response) => setData(response.data.data)).catch(() => undefined);
    refresh(); const timer = setInterval(refresh, 2000); return () => clearInterval(timer);
  }, []);
  const logs = useMemo(() => (data?.logs || []).filter((item) => Object.values(item).some((value) => String(value).toLowerCase().includes(query.toLowerCase()))), [data, query]);
  const rankings = useMemo(() => {
    const domains = new Map<string, number>(); const sources = new Map<string, number>();
    for (const item of data?.logs || []) { domains.set(item.domain, (domains.get(item.domain) || 0) + 1); sources.set(item.source_ip, (sources.get(item.source_ip) || 0) + 1); }
    const top = (values: Map<string, number>) => [...values.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10);
    return { domains: top(domains), sources: top(sources) };
  }, [data]);
  const stats = data?.stats || { total: 0, success: 0, failed: 0, cache_hits: 0, average_response_ms: 0 };
  const cards = [
    ['总查询数', stats.total, Database, 'text-blue-500'], ['成功查询', stats.success, CheckCircle, 'text-emerald-500'],
    ['失败查询', stats.failed, XCircle, 'text-red-500'], ['平均响应', `${stats.average_response_ms.toFixed(2)} ms`, Clock, 'text-purple-500'],
  ] as const;
  return <div className="space-y-6">
    <div className="flex items-center justify-between gap-3"><div><p className="text-sm font-medium text-blue-600">实时分析</p><h1 className="text-2xl font-bold">DNS 监控</h1></div><Chip color={data?.running ? 'success' : 'default'} variant="flat">{data?.running ? '运行中' : '未启用'}</Chip></div>
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">{cards.map(([label, value, Icon, color]) => <Card key={label}><CardBody className="flex-row items-center gap-3"><Icon className={`w-6 h-6 ${color}`} /><div><p className="text-sm text-gray-500">{label}</p><p className="text-2xl font-bold">{value}</p></div></CardBody></Card>)}</div>
    <div className="grid md:grid-cols-2 gap-4"><Card><CardBody><h2 className="font-semibold mb-3">域名请求排行</h2>{rankings.domains.map(([name,count]) => <div key={name} className="flex justify-between py-2 border-b border-divider text-sm"><span className="truncate">{name}</span><Chip size="sm" variant="flat">{count}</Chip></div>)}{!rankings.domains.length && <p className="text-gray-500">暂无数据</p>}</CardBody></Card><Card><CardBody><h2 className="font-semibold mb-3">来源 IP 排行</h2>{rankings.sources.map(([name,count]) => <div key={name} className="flex justify-between py-2 border-b border-divider text-sm"><span>{name}</span><Chip size="sm" variant="flat">{count}</Chip></div>)}{!rankings.sources.length && <p className="text-gray-500">暂无数据</p>}</CardBody></Card></div>
    <Card><CardBody className="space-y-4"><div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-center"><div><h2 className="text-lg font-semibold">DNS 查询日志</h2><p className="text-sm text-gray-500">最新 {logs.length} / 1000 条，缓存命中 {stats.cache_hits}</p></div><Input className="w-full sm:max-w-sm" startContent={<Search className="w-4 h-4" />} placeholder="搜索域名 / 类型 / 上游 / 来源 IP" value={query} onChange={(e) => setQuery(e.target.value)} /></div>
      <div className="overflow-auto"><table className="min-w-[760px] w-full text-sm"><thead><tr className="border-b text-left text-gray-500"><th className="p-3">时间</th><th>域名</th><th>类型</th><th>上游</th><th>来源 IP</th><th>耗时</th><th>状态</th></tr></thead><tbody>{logs.map((item, index) => <tr key={`${item.time}-${index}`} className="border-b border-divider"><td className="p-3 whitespace-nowrap">{new Date(item.time).toLocaleTimeString()}</td><td>{item.domain}</td><td>{item.type}</td><td>{item.upstream}</td><td>{item.source_ip}</td><td>{item.cached ? '缓存' : `${item.duration_ms.toFixed(1)} ms`}</td><td><Chip size="sm" color={item.result === 'success' ? 'success' : 'danger'} variant="flat">{item.result === 'success' ? '成功' : '失败'}</Chip></td></tr>)}</tbody></table>{logs.length === 0 && <p className="text-center text-gray-500 py-10">暂无查询数据</p>}</div>
    </CardBody></Card>
  </div>;
}
