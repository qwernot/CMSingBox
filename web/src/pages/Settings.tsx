import { useEffect, useState, useRef } from 'react';
import { Card, CardBody, CardHeader, Input, Button, Switch, Chip, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, Select, SelectItem, Progress, Textarea, useDisclosure } from '@nextui-org/react';
import { Save, Download, Upload, Terminal, CheckCircle, AlertCircle, Plus, Pencil, Trash2, Server, Eye, EyeOff, Copy, RefreshCw, Wifi, ShieldCheck, Database, ShoppingCart } from 'lucide-react';
import { useStore } from '../store';
import type { Settings as SettingsType, HostEntry } from '../store';
import { authApi, backupApi, daemonApi, firewallApi, kernelApi, licenseApi, maintenanceApi, settingsApi } from '../api';
import { toast } from '../components/Toast';

// 内核信息类型
interface KernelInfo {
  installed: boolean;
  version: string;
  path: string;
  os: string;
  arch: string;
}

// 下载进度类型
interface DownloadProgress {
  status: 'idle' | 'preparing' | 'downloading' | 'extracting' | 'installing' | 'completed' | 'error';
  progress: number;
  message: string;
  downloaded?: number;
  total?: number;
}

// GitHub Release 类型
interface GithubRelease {
  tag_name: string;
  name: string;
}

interface LicenseStatus {
  is_valid: boolean;
  status: 'licensed' | 'unlicensed';
  reason?: string;
  device_code: string;
  license_id?: string;
  max_subscriptions: number;
  used_subscriptions: number;
  remaining_subscriptions: number;
  expires_at?: number;
}

export default function Settings() {
  const { settings, fetchSettings, updateSettings } = useStore();
  const [formData, setFormData] = useState<SettingsType | null>(null);
  const [daemonStatus, setDaemonStatus] = useState<{ installed: boolean; running: boolean; supported: boolean } | null>(null);

  // 内核相关状态
  const [kernelInfo, setKernelInfo] = useState<KernelInfo | null>(null);
  const [releases, setReleases] = useState<GithubRelease[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<string>('');
  const [showDownloadModal, setShowDownloadModal] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [downloadProgress, setDownloadProgress] = useState<DownloadProgress | null>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Hosts 相关状态
  const [systemHosts, setSystemHosts] = useState<HostEntry[]>([]);
  const { isOpen: isHostModalOpen, onOpen: onHostModalOpen, onClose: onHostModalClose } = useDisclosure();
  const [editingHost, setEditingHost] = useState<HostEntry | null>(null);
  const [hostFormData, setHostFormData] = useState({ domain: '', enabled: true });
  const [ipsText, setIpsText] = useState('');

  // 密钥显示状态
  const [showSecret, setShowSecret] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [changingPassword, setChangingPassword] = useState(false);
  const [backupBusy, setBackupBusy] = useState(false);
  const [firewallStatus, setFirewallStatus] = useState<{ supported: boolean; active: boolean } | null>(null);
  const [cleanupPreview, setCleanupPreview] = useState<{ logs_bytes: number; temporary_bytes: number; files: number } | null>(null);
  const backupInputRef = useRef<HTMLInputElement | null>(null);
  const [licenseStatus, setLicenseStatus] = useState<LicenseStatus | null>(null);
  const [licenseCode, setLicenseCode] = useState('');
  const [licenseBusy, setLicenseBusy] = useState(false);

  useEffect(() => {
    fetchSettings();
    fetchDaemonStatus();
    fetchKernelInfo();
    fetchSystemHosts();
    firewallApi.status().then((response) => setFirewallStatus(response.data.data)).catch(() => undefined);
    maintenanceApi.preview().then((response) => setCleanupPreview(response.data.data)).catch(() => undefined);
    fetchLicenseStatus();
  }, []);

  const fetchLicenseStatus = async () => {
    try {
      const response = await licenseApi.status();
      setLicenseStatus(response.data.data);
    } catch (error) {
      console.error('获取授权状态失败:', error);
    }
  };

  const handleActivateLicense = async () => {
    setLicenseBusy(true);
    try {
      const response = await licenseApi.activate(licenseCode.trim());
      setLicenseStatus(response.data.data);
      setLicenseCode('');
      toast.success('授权成功');
    } catch (error: any) {
      toast.error(error.response?.data?.error || '授权失败');
    } finally {
      setLicenseBusy(false);
    }
  };

  const handleClearLicense = async () => {
    if (!confirm('清除后将恢复为最多 1 条订阅链接，确定继续吗？')) return;
    setLicenseBusy(true);
    try {
      const response = await licenseApi.clear();
      setLicenseStatus(response.data.data);
      toast.success('授权已清除');
    } catch (error: any) {
      toast.error(error.response?.data?.error || '清除授权失败');
    } finally {
      setLicenseBusy(false);
    }
  };

  useEffect(() => {
    if (settings) {
      setFormData(settings);
    }
  }, [settings]);

  // 清理轮询定时器
  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
      }
    };
  }, []);

  const fetchKernelInfo = async () => {
    try {
      const res = await kernelApi.getInfo();
      setKernelInfo(res.data.data);
    } catch (error) {
      console.error('获取内核信息失败:', error);
    }
  };

  const fetchSystemHosts = async () => {
    try {
      const res = await settingsApi.getSystemHosts();
      setSystemHosts(res.data.data || []);
    } catch (error) {
      console.error('获取系统 hosts 失败:', error);
    }
  };

  // Hosts 处理函数
  const handleAddHost = () => {
    setEditingHost(null);
    setHostFormData({ domain: '', enabled: true });
    setIpsText('');
    onHostModalOpen();
  };

  const handleEditHost = (host: HostEntry) => {
    setEditingHost(host);
    setHostFormData({ domain: host.domain, enabled: host.enabled });
    setIpsText(host.ips.join('\n'));
    onHostModalOpen();
  };

  const handleDeleteHost = (id: string) => {
    if (!formData?.hosts) return;
    setFormData({
      ...formData,
      hosts: formData.hosts.filter(h => h.id !== id)
    });
  };

  const handleToggleHost = (id: string, enabled: boolean) => {
    if (!formData?.hosts) return;
    setFormData({
      ...formData,
      hosts: formData.hosts.map(h => h.id === id ? { ...h, enabled } : h)
    });
  };

  const handleSubmitHost = () => {
    const ips = ipsText.split('\n').map(ip => ip.trim()).filter(ip => ip);

    // 验证 IP 格式
    const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
    const ipv6Regex = /^([a-fA-F0-9:]+)$/;
    const invalidIps = ips.filter(ip => !ipv4Regex.test(ip) && !ipv6Regex.test(ip));
    if (invalidIps.length > 0) {
      toast.error(`无效的 IP 地址: ${invalidIps.join(', ')}`);
      return;
    }

    // 验证域名格式
    const domainRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/;
    if (!domainRegex.test(hostFormData.domain)) {
      toast.error('无效的域名格式');
      return;
    }

    if (ips.length === 0) {
      toast.error('请输入至少一个 IP 地址');
      return;
    }

    const hosts = formData?.hosts || [];

    if (editingHost) {
      // 编辑模式
      setFormData({
        ...formData!,
        hosts: hosts.map(h => h.id === editingHost.id
          ? { ...h, domain: hostFormData.domain, ips, enabled: hostFormData.enabled }
          : h
        )
      });
    } else {
      // 新增模式
      const newHost: HostEntry = {
        id: crypto.randomUUID(),
        domain: hostFormData.domain,
        ips,
        enabled: hostFormData.enabled,
      };
      setFormData({
        ...formData!,
        hosts: [...hosts, newHost]
      });
    }

    onHostModalClose();
  };

  const fetchDaemonStatus = async () => {
    try {
      const res = await daemonApi.status();
      setDaemonStatus(res.data.data);
    } catch (error) {
      console.error('获取守护进程状态失败:', error);
    }
  };

  const fetchReleases = async () => {
    try {
      const res = await kernelApi.getReleases();
      setReleases(res.data.data || []);
      if (res.data.data && res.data.data.length > 0) {
        setSelectedVersion(res.data.data[0].tag_name);
      }
    } catch (error) {
      console.error('获取版本列表失败:', error);
    }
  };

  // 复制密钥到剪贴板（兼容非HTTPS环境）
  const handleCopySecret = () => {
    if (!formData?.clash_api_secret) return;

    const text = formData.clash_api_secret;

    // 优先尝试现代 API
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(() => {
        toast.success('密钥已复制到剪贴板');
      }).catch(() => {
        fallbackCopy(text);
      });
    } else {
      fallbackCopy(text);
    }
  };

  // 兼容性复制方法（支持非HTTPS环境）
  const fallbackCopy = (text: string) => {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.left = '-9999px';
    textarea.style.top = '-9999px';
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();

    try {
      const success = document.execCommand('copy');
      if (success) {
        toast.success('密钥已复制到剪贴板');
      } else {
        toast.error('复制失败');
      }
    } catch {
      toast.error('复制失败');
    } finally {
      document.body.removeChild(textarea);
    }
  };

  // 生成新的随机密钥
  const handleGenerateSecret = () => {
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let secret = '';
    for (let i = 0; i < 16; i++) {
      secret += charset.charAt(Math.floor(Math.random() * charset.length));
    }
    setFormData({ ...formData!, clash_api_secret: secret });
    toast.success('已生成新密钥，请保存设置');
  };

  const handleSave = async () => {
    if (formData) {
      try {
        await updateSettings(formData);
        toast.success('设置已保存');
      } catch (error: any) {
        toast.error(error.response?.data?.error || '保存设置失败');
      }
    }
  };

  const handleChangePassword = async () => {
    if (newPassword.length < 8) {
      toast.error('新密码至少需要 8 个字符');
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error('两次输入的新密码不一致');
      return;
    }
    setChangingPassword(true);
    try {
      await authApi.changePassword(currentPassword, newPassword);
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      toast.success('密码已更新');
    } catch (error: unknown) {
      const message = typeof error === 'object' && error !== null && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error
        : undefined;
      toast.error(message || '修改密码失败');
    } finally {
      setChangingPassword(false);
    }
  };

  const handleExportBackup = async () => {
    setBackupBusy(true);
    try {
      const response = await backupApi.export();
      const disposition = response.headers['content-disposition'] || '';
      const match = disposition.match(/filename="?([^";]+)"?/);
      const filename = match?.[1] || 'cmsingbox-backup.zip';
      const url = URL.createObjectURL(response.data);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = filename;
      anchor.click();
      URL.revokeObjectURL(url);
      toast.success('备份已导出');
    } catch {
      toast.error('导出备份失败');
    } finally {
      setBackupBusy(false);
    }
  };

  const handleImportBackup = async (file?: File) => {
    if (!file) return;
    if (!confirm('导入会覆盖当前的订阅、节点、规则和设置，确定继续吗？')) {
      if (backupInputRef.current) backupInputRef.current.value = '';
      return;
    }
    setBackupBusy(true);
    try {
      await backupApi.import(file);
      await fetchSettings();
      toast.success('备份恢复成功，请检查配置后重新应用');
    } catch (error: unknown) {
      const message = typeof error === 'object' && error !== null && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error
        : undefined;
      toast.error(message || '导入备份失败');
    } finally {
      setBackupBusy(false);
      if (backupInputRef.current) backupInputRef.current.value = '';
    }
  };

  const handleFirewall = async (enable: boolean) => {
    if (!confirm(enable ? '即将修改本机 nftables 和策略路由，确定应用透明代理规则吗？' : '确定停用本机透明代理规则吗？')) return;
    try { if (enable) await firewallApi.apply(); else await firewallApi.disable(); const response = await firewallApi.status(); setFirewallStatus(response.data.data); toast.success(enable ? '透明代理已启用' : '透明代理已停用'); }
    catch (error: unknown) { const message = typeof error === 'object' && error !== null && 'response' in error ? (error as { response?: { data?: { error?: string } } }).response?.data?.error : undefined; toast.error(message || '防火墙操作失败'); }
  };

  const formatBytes = (value = 0) => value > 1024 * 1024 ? `${(value / 1024 / 1024).toFixed(1)} MB` : `${(value / 1024).toFixed(1)} KB`;
  const handleCleanup = async () => { if (!confirm('确定清空应用日志和临时下载文件吗？此操作不可撤销。')) return; try { const response = await maintenanceApi.clean(true, true); setCleanupPreview(response.data.data); toast.success('系统垃圾已清理'); } catch { toast.error('清理失败'); } };

  const handleInstallDaemon = async () => {
    try {
      const res = await daemonApi.install();
      const data = res.data;
      if (data.action === 'exit') {
        toast.success(data.message);
      } else if (data.action === 'manual') {
        toast.info(data.message);
      } else {
        toast.success(data.message || '服务已安装');
      }
      await fetchDaemonStatus();
    } catch (error: any) {
      console.error('安装守护进程服务失败:', error);
      toast.error(error.response?.data?.error || '安装服务失败');
    }
  };

  const handleUninstallDaemon = async () => {
    if (confirm('确定要卸载后台服务吗？卸载后 CMSingBox 将不再开机自启。')) {
      try {
        await daemonApi.uninstall();
        toast.success('服务已卸载');
        await fetchDaemonStatus();
      } catch (error: any) {
        console.error('卸载守护进程服务失败:', error);
        toast.error(error.response?.data?.error || '卸载服务失败');
      }
    }
  };

  const handleRestartDaemon = async () => {
    try {
      await daemonApi.restart();
      toast.success('服务已重启');
      await fetchDaemonStatus();
    } catch (error: any) {
      console.error('重启守护进程服务失败:', error);
      toast.error(error.response?.data?.error || '重启服务失败');
    }
  };

  const openDownloadModal = async () => {
    await fetchReleases();
    setDownloadProgress(null);
    setShowDownloadModal(true);
  };

  const startDownload = async () => {
    if (!selectedVersion) return;

    setDownloading(true);
    setDownloadProgress({ status: 'preparing', progress: 0, message: '正在准备下载...' });

    try {
      await kernelApi.download(selectedVersion);

      // 开始轮询进度
      pollIntervalRef.current = setInterval(async () => {
        try {
          const res = await kernelApi.getProgress();
          const progress = res.data.data;
          setDownloadProgress(progress);

          if (progress.status === 'completed' || progress.status === 'error') {
            if (pollIntervalRef.current) {
              clearInterval(pollIntervalRef.current);
              pollIntervalRef.current = null;
            }
            setDownloading(false);

            if (progress.status === 'completed') {
              await fetchKernelInfo();
              setTimeout(() => setShowDownloadModal(false), 1500);
            }
          }
        } catch (error) {
          console.error('获取进度失败:', error);
        }
      }, 500);
    } catch (error: any) {
      setDownloading(false);
      setDownloadProgress({
        status: 'error',
        progress: 0,
        message: error.response?.data?.error || '下载失败',
      });
    }
  };

  if (!formData) {
    return <div>加载中...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div><p className="text-sm font-medium text-blue-600">系统配置</p><h1 className="text-2xl font-bold text-gray-800 dark:text-white">设置</h1></div>
        <Button
          color="primary"
          startContent={<Save className="w-4 h-4" />}
          onPress={handleSave}
        >
          保存设置
        </Button>
      </div>

      <Card>
        <CardHeader className="flex justify-between items-center">
          <div className="flex items-center"><ShieldCheck className="w-5 h-5 mr-2" /><h2 className="text-lg font-semibold">软件授权</h2></div>
          <Chip color={licenseStatus?.is_valid ? 'success' : 'warning'} variant="flat">
            {licenseStatus?.is_valid ? '已授权' : '未授权'}
          </Chip>
        </CardHeader>
        <CardBody className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-lg bg-default-100 p-3"><p className="text-xs text-default-500">设备码</p><p className="font-mono text-lg">{licenseStatus?.device_code || '------'}</p></div>
            <div className="rounded-lg bg-default-100 p-3"><p className="text-xs text-default-500">订阅链接</p><p className="text-lg">{licenseStatus?.used_subscriptions ?? 0} / {licenseStatus?.max_subscriptions ?? 1}</p></div>
            <div className="rounded-lg bg-default-100 p-3"><p className="text-xs text-default-500">有效期</p><p className="text-lg">{licenseStatus?.expires_at ? new Date(licenseStatus.expires_at * 1000).toLocaleDateString() : (licenseStatus?.is_valid ? '永久' : '未授权')}</p></div>
          </div>
          <p className="text-sm text-default-500">未授权版本最多添加 1 条订阅链接；订阅内节点数和手动节点数不限制。将此设备码交给授权方获取离线授权码。</p>
          {licenseStatus?.reason && <p className="text-sm text-warning">{licenseStatus.reason}</p>}
          <Input label="授权码" placeholder="CMS1..." value={licenseCode} onChange={(event) => setLicenseCode(event.target.value)} />
          <div className="flex flex-wrap gap-2">
            <Button color="primary" isLoading={licenseBusy} isDisabled={!licenseCode.trim()} onPress={handleActivateLicense}>激活授权</Button>
            <Button as="a" href="https://666228.xyz" target="_blank" rel="noopener noreferrer" color="secondary" variant="flat" startContent={<ShoppingCart className="h-4 w-4" />}>购买授权</Button>
            {licenseStatus?.is_valid && <Button color="danger" variant="flat" isDisabled={licenseBusy} onPress={handleClearLicense}>清除授权</Button>}
          </div>
        </CardBody>
      </Card>

      {/* sing-box 配置 */}
      <Card>
        <CardHeader>
          <Terminal className="w-5 h-5 mr-2" />
          <h2 className="text-lg font-semibold">sing-box 配置</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          {/* 内核状态 */}
          <div className="flex items-center justify-between p-4 rounded-lg bg-default-100">
            <div className="flex items-center gap-3">
              {kernelInfo?.installed ? (
                <>
                  <CheckCircle className="w-5 h-5 text-success" />
                  <div>
                    <p className="font-medium">sing-box 已安装</p>
                    <p className="text-sm text-gray-500">
                      版本: {kernelInfo.version || '未知'} | 平台: {kernelInfo.os}/{kernelInfo.arch}
                    </p>
                  </div>
                </>
              ) : (
                <>
                  <AlertCircle className="w-5 h-5 text-warning" />
                  <div>
                    <p className="font-medium text-warning">sing-box 未安装</p>
                    <p className="text-sm text-gray-500">
                      需要下载 sing-box 内核才能使用代理功能
                    </p>
                  </div>
                </>
              )}
            </div>
            <Button
              color={kernelInfo?.installed ? 'default' : 'primary'}
              variant={kernelInfo?.installed ? 'flat' : 'solid'}
              startContent={<Download className="w-4 h-4" />}
              onPress={openDownloadModal}
            >
              {kernelInfo?.installed ? '更新内核' : '下载内核'}
            </Button>
          </div>

          <Input
            label="配置文件路径"
            placeholder="generated/config.json"
            value={formData.config_path}
            onChange={(e) => setFormData({ ...formData, config_path: e.target.value })}
          />
          <Input
            label="GitHub 代理地址"
            placeholder="如 https://ghproxy.com/"
            description="用于加速 GitHub 下载，留空则直连"
            value={formData.github_proxy || ''}
            onChange={(e) => setFormData({ ...formData, github_proxy: e.target.value })}
          />
        </CardBody>
      </Card>

      {/* 入站配置 */}
      <Card>
        <CardHeader>
          <Download className="w-5 h-5 mr-2" />
          <h2 className="text-lg font-semibold">入站配置</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          <Input
            type="number"
            label="混合代理端口"
            placeholder="2080"
            value={String(formData.mixed_port)}
            onChange={(e) => setFormData({ ...formData, mixed_port: parseInt(e.target.value) || 2080 })}
          />
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">TUN 模式</p>
              <p className="text-sm text-gray-500">启用 TUN 模式进行透明代理</p>
            </div>
            <Switch
              isSelected={formData.tun_enabled}
              onValueChange={(enabled) => setFormData({ ...formData, tun_enabled: enabled })}
            />
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium flex items-center gap-2">
                <Wifi className="w-4 h-4" />
                开放 HTTP / SOCKS5 端口
              </p>
              <p className="text-sm text-gray-500">监听 0.0.0.0，允许其他设备通过服务器 IP 和端口连接</p>
            </div>
            <Chip color="success" variant="flat">默认开启</Chip>
          </div>

          {formData.allow_lan && (
            <div className={`space-y-4 rounded-xl border p-4 ${formData.mixed_auth_enabled ? 'border-success-200 bg-success-50 dark:border-success-800 dark:bg-success-900/20' : 'border-danger-200 bg-danger-50 dark:border-danger-800 dark:bg-danger-900/20'}`}>
              <div className="flex items-center justify-between gap-4">
                <div><p className="font-medium">HTTP / SOCKS5 连接认证</p><p className="text-sm text-gray-500">同一端口同时支持 HTTP 和 SOCKS5，认证可选。</p></div>
                <Switch isSelected={Boolean(formData.mixed_auth_enabled)} onValueChange={(enabled) => setFormData({ ...formData, mixed_auth_enabled: enabled })} />
              </div>
              {formData.mixed_auth_enabled ? (
                <div className="grid gap-3 sm:grid-cols-2">
                  <Input label="代理用户名" value={formData.mixed_username || ''} onChange={(e) => setFormData({ ...formData, mixed_username: e.target.value })} />
                  <Input label="代理密码" type={showSecret ? 'text' : 'password'} value={formData.mixed_password || ''} onChange={(e) => setFormData({ ...formData, mixed_password: e.target.value })} />
                </div>
              ) : <p className="text-sm text-danger-600 dark:text-danger-400">认证已关闭。任何能访问该端口的人都可以使用代理，公网环境不建议关闭。</p>}
            </div>
          )}

          {formData.allow_lan && (
            <div className="p-4 rounded-lg bg-warning-50 dark:bg-warning-900/20 border border-warning-200 dark:border-warning-800">
              <div className="flex items-center gap-2 mb-2">
                <p className="font-medium text-warning-700 dark:text-warning-400">ClashAPI 密钥</p>
                <Chip size="sm" color="warning" variant="flat">安全</Chip>
              </div>
              <p className="text-sm text-warning-600 dark:text-warning-500 mb-3">
                此密钥用于 zashboard 等外部 UI 连接时的认证，请妥善保管
              </p>
              <div className="flex items-center gap-2">
                <Input
                  type={showSecret ? "text" : "password"}
                  value={formData.clash_api_secret || ''}
                  onChange={(e) => setFormData({ ...formData, clash_api_secret: e.target.value })}
                  placeholder="保存设置后将自动生成"
                  size="sm"
                  className="flex-1"
                  endContent={
                    <Button
                      isIconOnly
                      size="sm"
                      variant="light"
                      onPress={() => setShowSecret(!showSecret)}
                    >
                      {showSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </Button>
                  }
                />
                <Button
                  isIconOnly
                  size="sm"
                  variant="flat"
                  onPress={handleCopySecret}
                  isDisabled={!formData.clash_api_secret}
                  title="复制密钥"
                >
                  <Copy className="w-4 h-4" />
                </Button>
                <Button
                  isIconOnly
                  size="sm"
                  variant="flat"
                  onPress={handleGenerateSecret}
                  title="重新生成"
                >
                  <RefreshCw className="w-4 h-4" />
                </Button>
              </div>
            </div>
          )}
        </CardBody>
      </Card>

      {/* DNS 配置 */}
      <Card>
        <CardHeader>
          <Upload className="w-5 h-5 mr-2" />
          <h2 className="text-lg font-semibold">DNS 配置</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          <Input
            label="代理 DNS"
            placeholder="https://1.1.1.1/dns-query"
            value={formData.proxy_dns}
            onChange={(e) => setFormData({ ...formData, proxy_dns: e.target.value })}
          />
          <Input
            label="直连 DNS"
            placeholder="udp://192.168.1.1:53"
            value={formData.direct_dns}
            onChange={(e) => setFormData({ ...formData, direct_dns: e.target.value })}
          />
          <div className="mt-4 pt-4 border-t border-divider space-y-4">
            <div className="flex items-center justify-between"><div><p className="font-medium">独立 DNS 分流服务</p><p className="text-sm text-gray-500">按客户端来源 IP 选择代理或直连上游，并记录查询统计</p></div><Switch isSelected={formData.dns_enabled} onValueChange={(value) => setFormData({ ...formData, dns_enabled: value })} /></div>
            <div className="grid gap-4 sm:grid-cols-2">
              <Input
                label="DNS 监听 IP"
                description="0.0.0.0 表示接受所有网卡的请求"
                value={(formData.dns_listen || '0.0.0.0:53').replace(/:\d+$/, '') || '0.0.0.0'}
                onChange={(e) => {
                  const port = (formData.dns_listen || '0.0.0.0:53').match(/:(\d+)$/)?.[1] || '53';
                  setFormData({ ...formData, dns_listen: `${e.target.value || '0.0.0.0'}:${port}` });
                }}
              />
              <Input
                type="number"
                min={1}
                max={65535}
                label="DNS 监听端口"
                description="标准 DNS 端口为 53，可按需修改"
                value={(formData.dns_listen || '0.0.0.0:53').match(/:(\d+)$/)?.[1] || '53'}
                onChange={(e) => {
                  const address = (formData.dns_listen || '0.0.0.0:53').replace(/:\d+$/, '') || '0.0.0.0';
                  const port = Math.min(65535, Math.max(1, Number(e.target.value) || 53));
                  setFormData({ ...formData, dns_listen: `${address}:${port}` });
                }}
              />
            </div>
            <div className="grid md:grid-cols-2 gap-4"><Input label="代理 DNS 上游" value={formData.dns_proxy_upstream || '127.0.0.1:1053'} onChange={(e) => setFormData({ ...formData, dns_proxy_upstream: e.target.value })} /><Input label="直连 DNS 上游" value={formData.dns_direct_upstream || '192.168.1.1:53'} onChange={(e) => setFormData({ ...formData, dns_direct_upstream: e.target.value })} /></div>
            <Select label="DNS 分流模式" selectedKeys={[formData.dns_routing_mode || 'default_proxy']} onSelectionChange={(keys) => setFormData({ ...formData, dns_routing_mode: String(Array.from(keys)[0]) })}><SelectItem key="default_proxy">默认代理，例外设备直连</SelectItem><SelectItem key="default_direct">默认直连，例外设备代理</SelectItem></Select>
            <Textarea label="例外设备" description="每行一个 IP 或 CIDR，可在 # 后添加备注" value={(formData.dns_exceptions || []).join('\n')} onChange={(e) => setFormData({ ...formData, dns_exceptions: e.target.value.split('\n').map((item) => item.trim()).filter(Boolean) })} />
          </div>

          {/* Hosts 映射 */}
          <div className="mt-6 pt-4 border-t border-divider">
            <div className="flex justify-between items-center mb-4">
              <div>
                <h3 className="font-medium">Hosts 映射</h3>
                <p className="text-sm text-gray-500">自定义域名解析（仅对 Sing-Box 生效）</p>
              </div>
              <Button
                color="primary"
                size="sm"
                startContent={<Plus className="w-4 h-4" />}
                onPress={handleAddHost}
              >
                添加
              </Button>
            </div>

            {/* 用户自定义 hosts */}
            {formData.hosts && formData.hosts.length > 0 && (
              <div className="mb-4">
                <p className="text-sm text-gray-500 mb-2">自定义映射</p>
                {formData.hosts.map((host) => (
                  <div
                    key={host.id}
                    className="flex items-center justify-between p-3 bg-default-100 rounded-lg mb-2"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <Server className="w-4 h-4 text-gray-500" />
                        <span className="font-medium">{host.domain}</span>
                        {!host.enabled && <Chip size="sm" variant="flat">已禁用</Chip>}
                      </div>
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {host.ips.map((ip, idx) => (
                          <Chip key={idx} size="sm" variant="bordered">{ip}</Chip>
                        ))}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <Button isIconOnly size="sm" variant="light" onPress={() => handleEditHost(host)}>
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button isIconOnly size="sm" variant="light" color="danger" onPress={() => handleDeleteHost(host.id)}>
                        <Trash2 className="w-4 h-4" />
                      </Button>
                      <Switch
                        size="sm"
                        isSelected={host.enabled}
                        onValueChange={(enabled) => handleToggleHost(host.id, enabled)}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* 系统 hosts（只读） */}
            {systemHosts.length > 0 && (
              <div>
                <p className="text-sm text-gray-500 mb-2">
                  系统 hosts <Chip size="sm" variant="flat">只读</Chip>
                </p>
                {systemHosts.map((host) => (
                  <div
                    key={host.id}
                    className="flex items-center justify-between p-3 bg-default-100 rounded-lg mb-2"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <Server className="w-4 h-4 text-gray-500" />
                        <span className="font-medium">{host.domain}</span>
                        <Chip size="sm" color="secondary" variant="flat">系统</Chip>
                      </div>
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {host.ips.map((ip, idx) => (
                          <Chip key={idx} size="sm" variant="bordered">{ip}</Chip>
                        ))}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* 空状态 */}
            {(!formData.hosts || formData.hosts.length === 0) && systemHosts.length === 0 && (
              <p className="text-gray-500 text-center py-4">暂无 hosts 映射</p>
            )}
          </div>
        </CardBody>
      </Card>

      {/* 控制面板配置 */}
      <Card>
        <CardHeader>
          <h2 className="text-lg font-semibold">控制面板</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          <Input
            type="number"
            label="Web 管理端口"
            placeholder="9090"
            disabled
            value={String(formData.web_port)}
            onChange={(e) => setFormData({ ...formData, web_port: parseInt(e.target.value) || 9090 })}
          />
          <Input
            type="number"
            label="Clash API 端口"
            placeholder="9090"
            value={String(formData.clash_api_port)}
            onChange={(e) => setFormData({ ...formData, clash_api_port: parseInt(e.target.value) || 9090 })}
          />
          <Input
            label="漏网规则出站"
            placeholder="Proxy"
            value={formData.final_outbound}
            onChange={(e) => setFormData({ ...formData, final_outbound: e.target.value })}
          />
        </CardBody>
      </Card>

      <Card>
        <CardHeader className="flex justify-between"><div><h2 className="text-lg font-semibold">透明代理</h2><p className="text-sm text-gray-500">使用 nftables TProxy 接管旁路由转发流量</p></div><Chip color={firewallStatus?.active ? 'success' : 'default'} variant="flat">{firewallStatus?.active ? '已应用' : '未应用'}</Chip></CardHeader>
        <CardBody className="space-y-4"><div className="flex justify-between"><div><p className="font-medium">启用透明代理配置</p><p className="text-sm text-gray-500">保存后会在 Sing-box 配置中生成 TProxy 入站</p></div><Switch isSelected={formData.transparent_proxy} onValueChange={(value) => setFormData({ ...formData, transparent_proxy: value })} /></div><Input type="number" label="TProxy 端口" value={String(formData.tproxy_port || 7893)} onChange={(e) => setFormData({ ...formData, tproxy_port: Number(e.target.value) })} /><Textarea label="绕过网段" value={(formData.bypass_cidrs || []).join('\n')} onChange={(e) => setFormData({ ...formData, bypass_cidrs: e.target.value.split('\n').map((item) => item.trim()).filter(Boolean) })} /><div className="flex gap-2"><Button color="primary" isDisabled={!formData.transparent_proxy || !firewallStatus?.supported} onPress={() => handleFirewall(true)}>应用 nftables</Button><Button color="danger" variant="flat" isDisabled={!firewallStatus?.active} onPress={() => handleFirewall(false)}>停用规则</Button>{firewallStatus && !firewallStatus.supported && <Chip color="warning" variant="flat">系统未安装 nft</Chip>}</div></CardBody>
      </Card>

      {/* 自动化设置 */}
      <Card>
        <CardHeader>
          <h2 className="text-lg font-semibold">自动化</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">配置变更后自动应用</p>
              <p className="text-sm text-gray-500">订阅刷新或规则变更后自动重启 sing-box</p>
            </div>
            <Switch
              isSelected={formData.auto_apply}
              onValueChange={(enabled) => setFormData({ ...formData, auto_apply: enabled })}
            />
          </div>
          <Input
            type="number"
            label="订阅自动更新间隔 (分钟)"
            placeholder="60"
            description="设置为 0 表示禁用自动更新"
            value={String(formData.subscription_interval)}
            onChange={(e) => setFormData({ ...formData, subscription_interval: parseInt(e.target.value) || 0 })}
          />
        </CardBody>
      </Card>

      {/* 账户安全 */}
      <Card>
        <CardHeader>
          <ShieldCheck className="w-5 h-5 mr-2" />
          <h2 className="text-lg font-semibold">账户安全</h2>
        </CardHeader>
        <CardBody className="space-y-4">
          <Input type="password" label="当前密码" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} autoComplete="current-password" />
          <Input type="password" label="新密码" description="至少 8 个字符" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} autoComplete="new-password" />
          <Input type="password" label="确认新密码" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} autoComplete="new-password" />
          <div>
            <Button color="primary" variant="flat" isLoading={changingPassword} isDisabled={!currentPassword || !newPassword || !confirmPassword} onPress={handleChangePassword}>
              修改密码
            </Button>
          </div>
        </CardBody>
      </Card>

      {/* 数据管理 */}
      <Card>
        <CardHeader>
          <Database className="w-5 h-5 mr-2" />
          <h2 className="text-lg font-semibold">数据管理</h2>
        </CardHeader>
        <CardBody>
          <p className="text-sm text-gray-500 mb-4">导出或恢复订阅、节点、筛选、规则和系统设置。密码及登录会话不会包含在备份中。</p>
          <div className="flex flex-wrap gap-2">
            <Button startContent={<Download className="w-4 h-4" />} isLoading={backupBusy} onPress={handleExportBackup}>导出数据</Button>
            <Button color="primary" variant="flat" startContent={<Upload className="w-4 h-4" />} isDisabled={backupBusy} onPress={() => backupInputRef.current?.click()}>导入数据</Button>
            <input ref={backupInputRef} type="file" accept=".zip,application/zip" className="hidden" onChange={(event) => handleImportBackup(event.target.files?.[0])} />
          </div>
        </CardBody>
      </Card>

      <Card><CardHeader><h2 className="text-lg font-semibold">系统清理</h2></CardHeader><CardBody><p className="text-sm text-gray-500 mb-4">可清理日志 {formatBytes(cleanupPreview?.logs_bytes)}、临时文件 {formatBytes(cleanupPreview?.temporary_bytes)}，共 {cleanupPreview?.files || 0} 个文件。</p><div><Button color="danger" variant="flat" isDisabled={!cleanupPreview || cleanupPreview.files === 0} onPress={handleCleanup}>清理日志与临时文件</Button></div></CardBody></Card>

      {/* 后台服务管理 */}
      {daemonStatus?.supported && (
        <Card>
          <CardHeader className="flex justify-between items-center">
            <h2 className="text-lg font-semibold">后台服务</h2>
            {daemonStatus && (
              <Chip
                color={daemonStatus.installed ? 'success' : 'default'}
                variant="flat"
                size="sm"
              >
                {daemonStatus.installed ? '已安装' : '未安装'}
              </Chip>
            )}
          </CardHeader>
          <CardBody>
            <p className="text-sm text-gray-500 mb-4">
              安装后台服务可让 CMSingBox 管理程序在后台运行，关闭终端后仍可访问 Web 管理界面。服务会开机自启并在崩溃后自动重启。
            </p>
            <div className="flex gap-2">
              {daemonStatus?.installed ? (
                <>
                  <Button
                    color="primary"
                    variant="flat"
                    onPress={handleRestartDaemon}
                  >
                    重启服务
                  </Button>
                  <Button
                    color="danger"
                    variant="flat"
                    onPress={handleUninstallDaemon}
                  >
                    卸载服务
                  </Button>
                </>
              ) : (
                <Button
                  color="primary"
                  onPress={handleInstallDaemon}
                >
                  安装后台服务
                </Button>
              )}
            </div>
          </CardBody>
        </Card>
      )}

      {/* 下载内核弹窗 */}
      <Modal isOpen={showDownloadModal} onClose={() => !downloading && setShowDownloadModal(false)}>
        <ModalContent>
          <ModalHeader>下载 sing-box 内核</ModalHeader>
          <ModalBody>
            <Select
              label="选择版本"
              placeholder="选择要下载的版本"
              selectedKeys={selectedVersion ? [selectedVersion] : []}
              onSelectionChange={(keys) => {
                const selected = Array.from(keys)[0] as string;
                if (selected) setSelectedVersion(selected);
              }}
              isDisabled={downloading}
            >
              {releases.map((release) => (
                <SelectItem key={release.tag_name} textValue={release.tag_name}>
                  {release.tag_name} {release.name && `- ${release.name}`}
                </SelectItem>
              ))}
            </Select>

            {kernelInfo && (
              <p className="text-sm text-gray-500">
                将下载适用于 {kernelInfo.os}/{kernelInfo.arch} 的版本
              </p>
            )}

            {downloadProgress && (
              <div className="mt-4 space-y-2">
                <Progress
                  value={downloadProgress.progress}
                  color={downloadProgress.status === 'error' ? 'danger' : downloadProgress.status === 'completed' ? 'success' : 'primary'}
                  showValueLabel
                />
                <p className={`text-sm ${downloadProgress.status === 'error' ? 'text-danger' : downloadProgress.status === 'completed' ? 'text-success' : 'text-gray-600'}`}>
                  {downloadProgress.message}
                </p>
              </div>
            )}
          </ModalBody>
          <ModalFooter>
            <Button
              variant="flat"
              onPress={() => setShowDownloadModal(false)}
              isDisabled={downloading}
            >
              取消
            </Button>
            <Button
              color="primary"
              onPress={startDownload}
              isLoading={downloading}
              isDisabled={!selectedVersion || downloading}
            >
              开始下载
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Hosts 编辑弹窗 */}
      <Modal isOpen={isHostModalOpen} onClose={onHostModalClose}>
        <ModalContent>
          <ModalHeader>{editingHost ? '编辑 Host' : '添加 Host'}</ModalHeader>
          <ModalBody className="gap-4">
            <Input
              label="域名"
              placeholder="例如：example.com"
              value={hostFormData.domain}
              onChange={(e) => setHostFormData({ ...hostFormData, domain: e.target.value })}
            />
            <Textarea
              label="IP 地址"
              placeholder={"每行一个 IP 地址\n例如：\n192.168.1.1\n192.168.1.2"}
              value={ipsText}
              onChange={(e) => setIpsText(e.target.value)}
              minRows={3}
            />
            <div className="flex items-center justify-between">
              <span>启用</span>
              <Switch
                isSelected={hostFormData.enabled}
                onValueChange={(enabled) => setHostFormData({ ...hostFormData, enabled })}
              />
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={onHostModalClose}>取消</Button>
            <Button
              color="primary"
              onPress={handleSubmitHost}
              isDisabled={!hostFormData.domain || !ipsText.trim()}
            >
              {editingHost ? '保存' : '添加'}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}
