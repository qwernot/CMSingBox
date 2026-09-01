import { useEffect, useState } from 'react';
import { Button, Card, CardBody, CardHeader, Input, Select, SelectItem, Switch, Textarea } from '@nextui-org/react';
import { Braces, Save } from 'lucide-react';
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
    <Button color="primary" startContent={<Save className="w-4 h-4" />} onPress={save} isLoading={saving}>
      保存并应用
    </Button>
  );

  return <div className="space-y-6">
    <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
      <div><h1 className="text-2xl font-bold">高级配置</h1><p className="text-sm text-gray-500">修改后点击“保存并应用”，不会在输入过程中自动提交</p></div>
      {saveButton}
    </div>

    <Card><CardHeader><h2 className="text-lg font-semibold">手机客户端配置</h2></CardHeader><CardBody className="space-y-4">
      <Input label="配置路径" value={formData.client_config_path || ''} onChange={(e) => patch({ client_config_path: e.target.value })} endContent={<Button size="sm" variant="light" onPress={() => patch({ client_config_path: crypto.randomUUID().replaceAll('-', '') })}>随机</Button>} />
      <Input isReadOnly label="订阅链接" value={`${window.location.origin}/client/${formData.client_config_path}`} />
      <div><Button as="a" href={`/client/${formData.client_config_path}`} target="_blank" color="primary" variant="flat">下载客户端配置</Button></div>
    </CardBody></Card>

    <Card><CardHeader><h2 className="text-lg font-semibold">回家配置</h2></CardHeader><CardBody className="space-y-4">
      <div className="flex justify-between"><div><p className="font-medium">启用 Hysteria2 回家服务</p><p className="text-sm text-gray-500">生成服务端入站及手机客户端“回家”出站</p></div><Switch isSelected={formData.backhome_enabled} onValueChange={(value) => patch({ backhome_enabled: value })} /></div>
      <Input label="DDNS 域名 / 公网 IP" value={formData.backhome_server || ''} onChange={(e) => patch({ backhome_server: e.target.value })} />
      <Input type="number" label="回家端口" value={String(formData.backhome_port || 8443)} onChange={(e) => patch({ backhome_port: Number(e.target.value) })} />
      <Input type="password" label="回家密码" value={formData.backhome_password || ''} onChange={(e) => patch({ backhome_password: e.target.value })} />
      <div className="grid gap-4 md:grid-cols-2"><Input label="TLS 证书路径" value={formData.backhome_cert_path || ''} onChange={(e) => patch({ backhome_cert_path: e.target.value })} /><Input label="TLS 私钥路径" value={formData.backhome_key_path || ''} onChange={(e) => patch({ backhome_key_path: e.target.value })} /></div>
    </CardBody></Card>

    <Card><CardHeader><h2 className="text-lg font-semibold">DNS 与日志</h2></CardHeader><CardBody className="space-y-4">
      <Input label="FakeIP 网段" value={formData.fakeip_range || '198.18.0.0/15'} onChange={(e) => patch({ fakeip_range: e.target.value })} />
      <div className="flex justify-between"><div><p className="font-medium">启用日志</p><p className="text-sm text-gray-500">控制 Sing-box 核心日志输出</p></div><Switch isSelected={formData.log_enabled} onValueChange={(value) => patch({ log_enabled: value })} /></div>
      <div className="flex justify-between"><span>显示时间戳</span><Switch isSelected={formData.log_timestamp} onValueChange={(value) => patch({ log_timestamp: value })} /></div>
      <Select label="日志级别" selectedKeys={[formData.log_level || 'info']} onSelectionChange={(keys) => patch({ log_level: String(Array.from(keys)[0]) })}>{['trace','debug','info','warn','error','fatal','panic'].map((level) => <SelectItem key={level}>{level}</SelectItem>)}</Select>
      <Input label="日志路径" placeholder="留空使用默认路径" value={formData.log_path || ''} onChange={(e) => patch({ log_path: e.target.value })} />
    </CardBody></Card>

    <Card><CardHeader><Braces className="w-5 h-5 mr-2" /><h2 className="text-lg font-semibold">Sing-box 配置扩展</h2></CardHeader><CardBody className="space-y-5">
      <Textarea label="额外入站" description="JSON 数组；内容追加到系统生成的入站之后" minRows={10} value={inbounds} onChange={(e) => setInbounds(e.target.value)} classNames={{ input: 'font-mono text-xs' }} />
      <Textarea label="额外出站" description="JSON 数组；可用于绑定指定网卡或自定义出口" minRows={10} value={outbounds} onChange={(e) => setOutbounds(e.target.value)} classNames={{ input: 'font-mono text-xs' }} />
    </CardBody></Card>

    <div className="flex justify-end pb-4">{saveButton}</div>
  </div>;
}
