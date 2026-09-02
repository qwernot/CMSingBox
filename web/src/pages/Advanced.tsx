import { useEffect, useState } from 'react';
import { Button, Card, CardBody, CardHeader, Input, Select, SelectItem, Switch } from '@nextui-org/react';
import { AlertTriangle, CheckCircle2, Code2, FileJson, Save } from 'lucide-react';
import { useStore } from '../store';
import type { Settings } from '../store';
import { toast } from '../components/Toast';

export default function Advanced() {
  const { settings, fetchSettings, updateSettings } = useStore();
  const [formData, setFormData] = useState<Settings | null>(null);
  const [inbounds, setInbounds] = useState('[]');
  const [outbounds, setOutbounds] = useState('[]');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!settings) fetchSettings();
  }, [settings, fetchSettings]);

  useEffect(() => {
    if (!settings) return;
    setFormData({ ...settings });
    setInbounds(JSON.stringify(settings.extra_inbounds || [], null, 2));
    setOutbounds(JSON.stringify(settings.extra_outbounds || [], null, 2));
  }, [settings]);

  if (!formData) return <div>加载中...</div>;

  const patch = (values: Partial<Settings>) => setFormData((current) => current ? { ...current, ...values } : current);
  const save = async () => {
    setSaving(true);
    try {
      const parsedInbounds = JSON.parse(inbounds);
      const parsedOutbounds = JSON.parse(outbounds);
      if (!Array.isArray(parsedInbounds) || !Array.isArray(parsedOutbounds)) {
        throw new Error('额外入站和出站必须是 JSON 数组');
      }
      await updateSettings({ ...formData, extra_inbounds: parsedInbounds, extra_outbounds: parsedOutbounds });
      toast.success('高级设置已保存并应用');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存高级设置失败');
    } finally {
      setSaving(false);
    }
  };

  const saveButton = (
    <Button color="primary" radius="lg" className="font-medium shadow-lg shadow-blue-500/15" startContent={<Save className="w-4 h-4" />} onPress={save} isLoading={saving}>
      保存并应用
    </Button>
  );

  const isJSONArray = (value: string) => {
    try { return Array.isArray(JSON.parse(value)); } catch { return false; }
  };

  const codeEditor = (title: string, description: string, value: string, onChange: (value: string) => void) => {
    const valid = isJSONArray(value);
    return <div className="overflow-hidden rounded-2xl border border-slate-700/70 bg-[#0a1220] shadow-inner">
      <div className="flex items-center justify-between border-b border-white/[0.07] bg-white/[0.025] px-4 py-3">
        <div className="flex items-center gap-2.5"><FileJson className="h-4 w-4 text-blue-400" /><div><p className="text-sm font-medium text-slate-100">{title}</p><p className="mt-0.5 text-[11px] text-slate-500">{description}</p></div></div>
        <span className={`inline-flex items-center gap-1 rounded-full px-2 py-1 text-[10px] font-medium ${valid ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'}`}>{valid ? <CheckCircle2 className="h-3 w-3" /> : <AlertTriangle className="h-3 w-3" />}{valid ? 'JSON 有效' : '格式错误'}</span>
      </div>
      <div className="relative px-4 py-3"><span className="pointer-events-none absolute left-4 top-4 select-none text-[11px] text-slate-700">JSON</span><textarea aria-label={title} spellCheck={false} value={value} onChange={(event) => onChange(event.target.value)} className="code-editor pt-7" /></div>
    </div>;
  };

  return <div className="space-y-6">
    <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div><p className="page-kicker">Advanced</p><h1 className="mt-1 text-2xl font-semibold tracking-tight sm:text-[28px]">高级配置</h1><p className="mt-1.5 text-sm text-slate-500">集中管理客户端、回家服务与 sing-box 扩展；所有修改仅在保存后生效。</p></div>
      {saveButton}
    </div>

    <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><h2 className="text-base font-semibold">手机客户端配置</h2></CardHeader><CardBody className="space-y-4 p-5">
      <Input label="配置路径" value={formData.client_config_path || ''} onChange={(e) => patch({ client_config_path: e.target.value })} endContent={<Button size="sm" variant="light" onPress={() => patch({ client_config_path: crypto.randomUUID().replaceAll('-', '') })}>随机</Button>} />
      <Input isReadOnly label="订阅链接" value={`${window.location.origin}/client/${formData.client_config_path}`} />
      <div><Button as="a" href={`/client/${formData.client_config_path}`} target="_blank" color="primary" variant="flat">下载客户端配置</Button></div>
    </CardBody></Card>

    <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><h2 className="text-base font-semibold">回家配置</h2></CardHeader><CardBody className="space-y-4 p-5">
      <div className="flex justify-between"><div><p className="font-medium">启用 Hysteria2 回家服务</p><p className="text-sm text-gray-500">生成服务端入站及手机客户端“回家”出站</p></div><Switch isSelected={formData.backhome_enabled} onValueChange={(value) => patch({ backhome_enabled: value })} /></div>
      <Input label="DDNS 域名 / 公网 IP" value={formData.backhome_server || ''} onChange={(e) => patch({ backhome_server: e.target.value })} />
      <Input type="number" label="回家端口" value={String(formData.backhome_port || 8443)} onChange={(e) => patch({ backhome_port: Number(e.target.value) })} />
      <Input type="password" label="回家密码" value={formData.backhome_password || ''} onChange={(e) => patch({ backhome_password: e.target.value })} />
      <div className="grid gap-4 md:grid-cols-2"><Input label="TLS 证书路径" value={formData.backhome_cert_path || ''} onChange={(e) => patch({ backhome_cert_path: e.target.value })} /><Input label="TLS 私钥路径" value={formData.backhome_key_path || ''} onChange={(e) => patch({ backhome_key_path: e.target.value })} /></div>
    </CardBody></Card>

    <Card className="app-panel"><CardHeader className="border-b border-slate-100 px-5 py-4 dark:border-slate-800"><h2 className="text-base font-semibold">DNS 与日志</h2></CardHeader><CardBody className="space-y-4 p-5">
      <Input label="FakeIP 网段" value={formData.fakeip_range || '198.18.0.0/15'} onChange={(e) => patch({ fakeip_range: e.target.value })} />
      <div className="flex justify-between"><div><p className="font-medium">启用日志</p><p className="text-sm text-gray-500">控制 Sing-box 核心日志输出</p></div><Switch isSelected={formData.log_enabled} onValueChange={(value) => patch({ log_enabled: value })} /></div>
      <div className="flex justify-between"><span>显示时间戳</span><Switch isSelected={formData.log_timestamp} onValueChange={(value) => patch({ log_timestamp: value })} /></div>
      <Select label="日志级别" selectedKeys={[formData.log_level || 'info']} onSelectionChange={(keys) => patch({ log_level: String(Array.from(keys)[0]) })}>{['trace','debug','info','warn','error','fatal','panic'].map((level) => <SelectItem key={level}>{level}</SelectItem>)}</Select>
      <Input label="日志路径" placeholder="留空使用默认路径" value={formData.log_path || ''} onChange={(e) => patch({ log_path: e.target.value })} />
    </CardBody></Card>

    <section className="app-panel overflow-hidden">
      <div className="flex flex-col gap-3 border-b border-slate-200/80 px-5 py-5 dark:border-slate-800 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-3"><span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400"><Code2 className="h-5 w-5" /></span><div><h2 className="text-base font-semibold">Sing-box 配置扩展</h2><p className="mt-1 text-xs leading-5 text-slate-500">仅用于系统未提供的高级字段。内容会追加到自动生成的配置中，不会覆盖基础代理设置。</p></div></div>
        <span className="w-fit rounded-full bg-amber-50 px-3 py-1 text-[11px] font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">修改前建议先备份</span>
      </div>
      <div className="grid gap-4 p-4 sm:p-5 xl:grid-cols-2">
        {codeEditor('额外入站', 'JSON 数组 · 追加到系统入站之后', inbounds, setInbounds)}
        {codeEditor('额外出站', 'JSON 数组 · 自定义出口或绑定网卡', outbounds, setOutbounds)}
      </div>
      <div className="flex items-start gap-2 border-t border-slate-100 bg-slate-50/70 px-5 py-3 text-xs leading-5 text-slate-500 dark:border-slate-800 dark:bg-slate-950/30"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" /><span>保存前会校验 JSON 和完整 sing-box 配置；校验失败时不会替换当前可用配置，也不会中断正在运行的代理。</span></div>
    </section>

    <div className="flex justify-end pb-4">{saveButton}</div>
  </div>;
}
