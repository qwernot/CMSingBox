import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button, Card, CardBody, CardHeader, Input, Select, SelectItem, Switch } from '@nextui-org/react';
import { AlertTriangle, CheckCircle2, Code2, FileJson, House, PanelsTopLeft, Save, ScrollText, Server, Smartphone } from 'lucide-react';
import { useStore } from '../store';
import type { Settings } from '../store';
import { toast } from '../components/Toast';

type EditorName = 'inbounds' | 'outbounds' | null;

export default function Advanced() {
  const { settings, fetchSettings, updateSettings } = useStore();
  const [formData, setFormData] = useState<Settings | null>(null);
  const [inbounds, setInbounds] = useState('[]');
  const [outbounds, setOutbounds] = useState('[]');
  const [editor, setEditor] = useState<EditorName>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { if (!settings) fetchSettings(); }, [settings, fetchSettings]);
  useEffect(() => {
    if (!settings) return;
    setFormData({ ...settings });
    setInbounds(JSON.stringify(settings.extra_inbounds || [], null, 2));
    setOutbounds(JSON.stringify(settings.extra_outbounds || [], null, 2));
  }, [settings]);

  if (!formData) return <div>加载中...</div>;

  const patch = (values: Partial<Settings>) => setFormData((current) => current ? { ...current, ...values } : current);
  const clashUIURL = (() => {
    const host = window.location.hostname;
    const port = formData.clash_api_port || 9090;
    const params = new URLSearchParams({ hostname: host, port: String(port) });
    if (formData.clash_api_secret) params.set('secret', formData.clash_api_secret);
    return `${window.location.protocol}//${host}:${port}/ui/#/setup?${params.toString()}`;
  })();
  const isJSONArray = (value: string) => { try { return Array.isArray(JSON.parse(value)); } catch { return false; } };
  const countItems = (value: string) => { try { const parsed = JSON.parse(value); return Array.isArray(parsed) ? parsed.length : 0; } catch { return 0; } };
  const save = async () => {
    setSaving(true);
    try {
      const parsedInbounds = JSON.parse(inbounds);
      const parsedOutbounds = JSON.parse(outbounds);
      if (!Array.isArray(parsedInbounds) || !Array.isArray(parsedOutbounds)) throw new Error('额外入站和出站必须是 JSON 数组');
      await updateSettings({ ...formData, extra_inbounds: parsedInbounds, extra_outbounds: parsedOutbounds });
      toast.success('高级设置已保存并应用');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存高级设置失败');
    } finally { setSaving(false); }
  };

  const saveButton = <Button color="primary" radius="lg" className="font-medium shadow-lg shadow-blue-500/15" startContent={<Save className="h-4 w-4" />} onPress={save} isLoading={saving}>保存并应用</Button>;
  const codeEditor = (title: string, description: string, value: string, onChange: (value: string) => void) => {
    const valid = isJSONArray(value);
    return <div className="overflow-hidden rounded-2xl border border-slate-700/70 bg-[#0a1220] shadow-inner">
      <div className="flex items-center justify-between border-b border-white/[0.07] bg-white/[0.025] px-4 py-3"><div className="flex items-center gap-2.5"><FileJson className="h-4 w-4 text-blue-400" /><div><p className="text-sm font-medium text-slate-100">{title}</p><p className="mt-0.5 text-[11px] text-slate-500">{description}</p></div></div><span className={`inline-flex items-center gap-1 rounded-full px-2 py-1 text-[10px] font-medium ${valid ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'}`}>{valid ? <CheckCircle2 className="h-3 w-3" /> : <AlertTriangle className="h-3 w-3" />}{valid ? 'JSON 有效' : '格式错误'}</span></div>
      <div className="relative px-4 py-3"><span className="pointer-events-none absolute left-4 top-4 select-none text-[11px] text-slate-700">JSON</span><textarea aria-label={title} spellCheck={false} value={value} onChange={(event) => onChange(event.target.value)} className="code-editor pt-7" /></div>
    </div>;
  };

  return <div className="space-y-7">
    <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end"><div><p className="page-kicker">Advanced</p><h1 className="mt-1 text-2xl font-semibold tracking-tight sm:text-[28px]">高级配置</h1><p className="mt-1.5 text-sm text-slate-500">客户端、回家、DNS、控制面板与 sing-box 扩展配置。</p></div>{saveButton}</div>

    <section><div className="mb-4 flex items-center gap-3"><Smartphone className="h-5 w-5 text-blue-500" /><div><h2 className="text-lg font-semibold">客户端配置</h2><p className="text-xs text-slate-500">手机客户端订阅和回家功能</p></div></div><div className="grid gap-5 xl:grid-cols-2">
      <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><h3 className="font-semibold">手机客户端配置</h3></CardHeader><CardBody className="space-y-4 p-5"><Input label="配置路径" value={formData.client_config_path || ''} onChange={(event) => patch({ client_config_path: event.target.value })} endContent={<Button size="sm" variant="light" onPress={() => patch({ client_config_path: crypto.randomUUID().replaceAll('-', '') })}>随机</Button>} /><Input isReadOnly label="订阅链接" value={`${window.location.origin}/client/${formData.client_config_path}`} /><Button as="a" href={`/client/${formData.client_config_path}`} target="_blank" color="primary" variant="flat">下载客户端配置</Button></CardBody></Card>
      <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><div className="flex items-center gap-2"><House className="h-4 w-4 text-orange-500" /><h3 className="font-semibold">回家配置</h3></div></CardHeader><CardBody className="space-y-4 p-5"><div className="flex justify-between gap-4"><div><p className="font-medium">启用 Hysteria2 回家服务</p><p className="text-sm text-slate-500">生成服务端入站及手机客户端“回家”出站</p></div><Switch isSelected={formData.backhome_enabled} onValueChange={(value) => patch({ backhome_enabled: value })} /></div><Input isDisabled={!formData.backhome_enabled} label="DDNS 域名 / 公网 IP" value={formData.backhome_server || ''} onChange={(event) => patch({ backhome_server: event.target.value })} /><div className="grid gap-4 sm:grid-cols-2"><Input isDisabled={!formData.backhome_enabled} type="number" label="回家端口" value={String(formData.backhome_port || 8443)} onChange={(event) => patch({ backhome_port: Number(event.target.value) })} /><Input isDisabled={!formData.backhome_enabled} type="password" label="回家密码" value={formData.backhome_password || ''} onChange={(event) => patch({ backhome_password: event.target.value })} /></div><p className="text-xs leading-5 text-slate-500">首次启用会自动生成并保存在数据目录的自签名证书，无需手动上传。手机客户端配置会自动兼容该证书；请妥善保护回家密码。</p></CardBody></Card>
    </div></section>

    <section><div className="mb-4 flex items-center gap-3"><PanelsTopLeft className="h-5 w-5 text-violet-500" /><div><h2 className="text-lg font-semibold">系统配置</h2><p className="text-xs text-slate-500">控制面板、DNS 策略与核心日志</p></div></div><div className="grid gap-5 xl:grid-cols-2">
      <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><h3 className="font-semibold">Clash 控制面板</h3></CardHeader><CardBody className="space-y-4 p-5"><Input type="number" label="Clash API 端口" value={String(formData.clash_api_port || 9090)} onChange={(event) => patch({ clash_api_port: Number(event.target.value) || 9090 })} /><Input type="password" label="控制面板密码" description="留空则无需密码；开放到公网时建议设置密码" value={formData.clash_api_secret || ''} onChange={(event) => patch({ clash_api_secret: event.target.value })} /><Input label="自定义面板 ZIP 下载地址（可选）" placeholder="留空使用内置 Zashboard" value={formData.clash_ui_url || ''} onChange={(event) => patch({ clash_ui_url: event.target.value, clash_ui_revision: Date.now() })} /><Input label="UI 下载出站" placeholder="DIRECT 或 Proxy" isDisabled={!formData.clash_ui_url} value={formData.clash_ui_detour || 'DIRECT'} onChange={(event) => patch({ clash_ui_detour: event.target.value })} /><p className="text-xs leading-5 text-slate-500">自定义面板下载到独立目录，不会覆盖内置 Zashboard。清空链接后恢复内置面板；这里只更新控制 UI，不会更新 sing-box 内核。</p><div className="flex flex-wrap gap-2"><Button as="a" href={clashUIURL} target="_blank" color="primary" variant="flat">打开控制 UI</Button><Button variant="flat" isDisabled={!formData.clash_ui_url} onPress={() => patch({ clash_ui_revision: Date.now() })}>重新下载 UI</Button><span className="self-center text-xs text-slate-500">点击重新下载后仍需保存并应用</span></div></CardBody></Card>
      <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><h3 className="font-semibold">DNS 配置</h3></CardHeader><CardBody className="space-y-4 p-5"><Select label="DNS 策略（全局）" selectedKeys={[formData.dns_strategy || 'prefer_ipv4']} onSelectionChange={(keys) => patch({ dns_strategy: String(Array.from(keys)[0] || 'prefer_ipv4') })}><SelectItem key="prefer_ipv4">优先 IPv4</SelectItem><SelectItem key="prefer_ipv6">优先 IPv6</SelectItem><SelectItem key="ipv4_only">仅 IPv4</SelectItem><SelectItem key="ipv6_only">仅 IPv6</SelectItem></Select><Input label="代理 DNS" placeholder="https://1.1.1.1/dns-query" value={formData.proxy_dns || ''} onChange={(event) => patch({ proxy_dns: event.target.value })} /><Input label="直连 DNS" placeholder="https://dns.alidns.com/dns-query" value={formData.direct_dns || ''} onChange={(event) => patch({ direct_dns: event.target.value })} /><Input label="FakeIP 网段" value={formData.fakeip_range || '198.18.0.0/15'} onChange={(event) => patch({ fakeip_range: event.target.value })} /><p className="text-xs leading-5 text-slate-500">独立 DNS 服务监听、设备例外和 Hosts 映射仍在“系统设置 → DNS 配置”中管理。</p></CardBody></Card>
      <Card className="app-panel xl:col-span-2"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><div className="flex items-center gap-2"><ScrollText className="h-4 w-4 text-cyan-500" /><h3 className="font-semibold">日志配置</h3></div></CardHeader><CardBody className="grid gap-4 p-5 sm:grid-cols-2 xl:grid-cols-4"><div className="flex items-center justify-between rounded-xl bg-slate-50 px-4 dark:bg-slate-900/70"><span className="text-sm">启用日志</span><Switch isSelected={formData.log_enabled} onValueChange={(value) => patch({ log_enabled: value })} /></div><div className="flex items-center justify-between rounded-xl bg-slate-50 px-4 dark:bg-slate-900/70"><span className="text-sm">显示时间戳</span><Switch isSelected={formData.log_timestamp} onValueChange={(value) => patch({ log_timestamp: value })} /></div><Select label="日志级别" selectedKeys={[formData.log_level || 'info']} onSelectionChange={(keys) => patch({ log_level: String(Array.from(keys)[0]) })}>{['trace','debug','info','warn','error','fatal','panic'].map((level) => <SelectItem key={level}>{level}</SelectItem>)}</Select><Input label="日志路径" placeholder="留空使用默认路径" value={formData.log_path || ''} onChange={(event) => patch({ log_path: event.target.value })} /></CardBody></Card>
    </div></section>

    <section className="app-panel overflow-hidden"><div className="flex flex-col gap-3 border-b border-slate-200/80 px-5 py-5 dark:border-slate-800 sm:flex-row sm:items-center sm:justify-between"><div className="flex items-start gap-3"><span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400"><Code2 className="h-5 w-5" /></span><div><h2 className="text-base font-semibold">Sing-box 配置扩展</h2><p className="mt-1 text-xs leading-5 text-slate-500">系统没有提供的字段才需要在这里追加，默认保持为空。</p></div></div><span className="w-fit rounded-full bg-amber-50 px-3 py-1 text-[11px] font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">修改前建议备份</span></div>
      <div className="grid gap-4 p-5 md:grid-cols-3"><div className="rounded-2xl border border-slate-200 p-4 dark:border-slate-800"><div className="mb-5 flex items-center justify-between"><Server className="h-5 w-5 text-blue-500" /><span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs dark:bg-slate-800">{countItems(inbounds)} 项配置</span></div><h3 className="font-semibold">额外入站</h3><p className="mt-1 text-sm text-slate-500">追加到系统生成的入站之后</p><Button className="mt-5 w-full" variant="flat" onPress={() => setEditor(editor === 'inbounds' ? null : 'inbounds')}>{editor === 'inbounds' ? '收起' : '编辑'}</Button></div><div className="rounded-2xl border border-slate-200 p-4 dark:border-slate-800"><div className="mb-5 flex items-center justify-between"><FileJson className="h-5 w-5 text-violet-500" /><span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs dark:bg-slate-800">{countItems(outbounds)} 项配置</span></div><h3 className="font-semibold">额外出站</h3><p className="mt-1 text-sm text-slate-500">自定义出口或绑定指定网卡</p><Button className="mt-5 w-full" variant="flat" onPress={() => setEditor(editor === 'outbounds' ? null : 'outbounds')}>{editor === 'outbounds' ? '收起' : '编辑'}</Button></div><div className="rounded-2xl border border-slate-200 p-4 dark:border-slate-800"><div className="mb-5 flex items-center justify-between"><Server className="h-5 w-5 text-cyan-500" /><span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs dark:bg-slate-800">{formData.hosts?.filter((host) => host.enabled).length || 0} 条映射</span></div><h3 className="font-semibold">静态 DNS</h3><p className="mt-1 text-sm text-slate-500">域名到 IP 的 Hosts 映射</p><Button as={Link} to="/settings" className="mt-5 w-full" variant="flat">前往编辑</Button></div></div>
      {editor && <div className="border-t border-slate-200 p-5 dark:border-slate-800">{editor === 'inbounds' ? codeEditor('额外入站', 'JSON 数组 · 追加到系统入站之后', inbounds, setInbounds) : codeEditor('额外出站', 'JSON 数组 · 自定义出口或绑定网卡', outbounds, setOutbounds)}</div>}
      <div className="flex items-start gap-2 border-t border-slate-100 bg-slate-50/70 px-5 py-3 text-xs leading-5 text-slate-500 dark:border-slate-800 dark:bg-slate-950/30"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" /><span>保存前会校验 JSON 和完整 sing-box 配置；失败时不会替换当前可用配置，也不会中断正在运行的代理。</span></div>
    </section>
    <div className="flex justify-end pb-4">{saveButton}</div>
  </div>;
}
