import { useEffect, useState } from 'react';
import { Button, Card, CardBody, CardHeader, Input, Select, SelectItem, Switch, Textarea } from '@nextui-org/react';
import { Braces, Save } from 'lucide-react';
import { useStore } from '../store';
import { toast } from '../components/Toast';

export default function Advanced() {
  const { settings, fetchSettings, updateSettings } = useStore();
  const [inbounds, setInbounds] = useState('[]');
  const [outbounds, setOutbounds] = useState('[]');
  useEffect(() => { if (!settings) fetchSettings(); }, [settings, fetchSettings]);
  useEffect(() => { if (settings) { setInbounds(JSON.stringify(settings.extra_inbounds || [], null, 2)); setOutbounds(JSON.stringify(settings.extra_outbounds || [], null, 2)); } }, [settings]);
  if (!settings) return <div>加载中...</div>;
  const save = async () => {
    try {
      const parsedInbounds = JSON.parse(inbounds); const parsedOutbounds = JSON.parse(outbounds);
      if (!Array.isArray(parsedInbounds) || !Array.isArray(parsedOutbounds)) throw new Error('额外入站和出站必须是 JSON 数组');
      await updateSettings({ ...settings, extra_inbounds: parsedInbounds, extra_outbounds: parsedOutbounds }); toast.success('高级配置已保存');
    } catch (error) { toast.error(error instanceof Error ? error.message : 'JSON 格式错误'); }
  };
  const patch = (values: Partial<typeof settings>) => updateSettings({ ...settings, ...values });
  return <div className="space-y-6"><div className="flex justify-between"><div><h1 className="text-2xl font-bold">高级配置</h1><p className="text-sm text-gray-500">自定义 Sing-box 核心配置</p></div><Button color="primary" startContent={<Save className="w-4 h-4" />} onPress={save}>保存 JSON 配置</Button></div>
    <Card><CardHeader><h2 className="text-lg font-semibold">手机客户端配置</h2></CardHeader><CardBody className="space-y-4"><Input label="配置路径" value={settings.client_config_path || ''} onChange={(e) => patch({ client_config_path: e.target.value })} endContent={<Button size="sm" variant="light" onPress={() => patch({ client_config_path: crypto.randomUUID().replaceAll('-', '') })}>随机</Button>} /><Input isReadOnly label="订阅链接" value={`${window.location.origin}/client/${settings.client_config_path}`} /><div><Button as="a" href={`/client/${settings.client_config_path}`} target="_blank" color="primary" variant="flat">下载客户端配置</Button></div></CardBody></Card>
    <Card><CardHeader><h2 className="text-lg font-semibold">回家配置</h2></CardHeader><CardBody className="space-y-4"><div className="flex justify-between"><div><p className="font-medium">启用 Hysteria2 回家服务</p><p className="text-sm text-gray-500">生成服务端入站及手机客户端“回家”出站</p></div><Switch isSelected={settings.backhome_enabled} onValueChange={(value) => patch({ backhome_enabled: value })} /></div><Input label="DDNS 域名 / 公网 IP" value={settings.backhome_server || ''} onChange={(e) => patch({ backhome_server: e.target.value })} /><Input type="number" label="回家端口" value={String(settings.backhome_port || 8443)} onChange={(e) => patch({ backhome_port: Number(e.target.value) })} /><Input type="password" label="回家密码" value={settings.backhome_password || ''} onChange={(e) => patch({ backhome_password: e.target.value })} /><div className="grid md:grid-cols-2 gap-4"><Input label="TLS 证书路径" value={settings.backhome_cert_path || ''} onChange={(e) => patch({ backhome_cert_path: e.target.value })} /><Input label="TLS 私钥路径" value={settings.backhome_key_path || ''} onChange={(e) => patch({ backhome_key_path: e.target.value })} /></div></CardBody></Card>
    <Card><CardHeader><h2 className="text-lg font-semibold">DNS 与日志</h2></CardHeader><CardBody className="space-y-4"><Input label="FakeIP 网段" value={settings.fakeip_range || '198.18.0.0/15'} onChange={(e) => patch({ fakeip_range: e.target.value })} /><div className="flex justify-between"><div><p className="font-medium">启用日志</p><p className="text-sm text-gray-500">控制 Sing-box 核心日志输出</p></div><Switch isSelected={settings.log_enabled} onValueChange={(value) => patch({ log_enabled: value })} /></div><div className="flex justify-between"><span>显示时间戳</span><Switch isSelected={settings.log_timestamp} onValueChange={(value) => patch({ log_timestamp: value })} /></div><Select label="日志级别" selectedKeys={[settings.log_level || 'info']} onSelectionChange={(keys) => patch({ log_level: String(Array.from(keys)[0]) })}>{['trace','debug','info','warn','error','fatal','panic'].map((level) => <SelectItem key={level}>{level}</SelectItem>)}</Select><Input label="日志路径" placeholder="留空使用默认路径" value={settings.log_path || ''} onChange={(e) => patch({ log_path: e.target.value })} /></CardBody></Card>
    <Card><CardHeader><Braces className="w-5 h-5 mr-2" /><h2 className="text-lg font-semibold">Sing-box 配置扩展</h2></CardHeader><CardBody className="space-y-5"><Textarea label="额外入站" description="JSON 数组；内容追加到系统生成的入站之后" minRows={10} value={inbounds} onChange={(e) => setInbounds(e.target.value)} classNames={{ input: 'font-mono text-xs' }} /><Textarea label="额外出站" description="JSON 数组；可用于绑定指定网卡或自定义出口" minRows={10} value={outbounds} onChange={(e) => setOutbounds(e.target.value)} classNames={{ input: 'font-mono text-xs' }} /></CardBody></Card>
  </div>;
}
