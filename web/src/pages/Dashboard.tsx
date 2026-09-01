import { useEffect, useState } from 'react';
import { Card, CardBody, CardHeader, Button, Chip, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, Tooltip } from '@nextui-org/react';
import { Play, Square, RefreshCw, Cpu, HardDrive, Wifi, Info, Activity } from 'lucide-react';
import { useStore } from '../store';
import { serviceApi, configApi } from '../api';
import { toast } from '../components/Toast';

export default function Dashboard() {
  const { serviceStatus, subscriptions, systemInfo, fetchServiceStatus, fetchSubscriptions, fetchSystemInfo } = useStore();

  // 错误模态框状态
  const [errorModal, setErrorModal] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
  }>({
    isOpen: false,
    title: '',
    message: ''
  });

  // 显示错误的辅助函数
  const showError = (title: string, error: any) => {
    const message = error.response?.data?.error || error.message || '操作失败';
    setErrorModal({
      isOpen: true,
      title,
      message
    });
  };

  useEffect(() => {
    fetchServiceStatus();
    fetchSubscriptions();
    fetchSystemInfo();

    // 每 5 秒刷新状态和系统信息
    const interval = setInterval(() => {
      fetchServiceStatus();
      fetchSystemInfo();
    }, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleStart = async () => {
    try {
      await serviceApi.start();
      await fetchServiceStatus();
      toast.success('服务已启动');
    } catch (error) {
      showError('启动失败', error);
    }
  };

  const handleStop = async () => {
    try {
      await serviceApi.stop();
      await fetchServiceStatus();
      toast.success('服务已停止');
    } catch (error) {
      showError('停止失败', error);
    }
  };

  const handleRestart = async () => {
    try {
      await serviceApi.restart();
      await fetchServiceStatus();
      toast.success('服务已重启');
    } catch (error) {
      showError('重启失败', error);
    }
  };

  const handleApplyConfig = async () => {
    try {
      await configApi.apply();
      await fetchServiceStatus();
      toast.success('配置已应用');
    } catch (error) {
      showError('应用配置失败', error);
    }
  };

  const totalNodes = subscriptions.reduce((sum, sub) => sum + sub.node_count, 0);
  const enabledSubs = subscriptions.filter(sub => sub.enabled).length;
  const formatRate = (value = 0) => value >= 1024 * 1024 ? `${(value / 1024 / 1024).toFixed(1)} MB/s` : `${(value / 1024).toFixed(1)} KB/s`;

  return (
    <div className="space-y-6">
      <div><p className="text-sm font-medium text-blue-600">系统总览</p><h1 className="mt-1 text-2xl font-bold tracking-tight text-slate-900 dark:text-white sm:text-3xl">欢迎使用 CMSingBox</h1><p className="mt-1 text-sm text-slate-500">实时查看服务状态、系统资源与订阅概况</p></div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card><CardBody><p className="text-sm text-gray-500">主机名</p><p className="text-xl font-bold">{systemInfo?.host?.hostname || '-'}</p><p className="text-xs text-gray-400">{systemInfo?.host?.ip_address || ''}</p></CardBody></Card>
        <Card><CardBody><p className="text-sm text-gray-500">CPU 使用率</p><p className="text-xl font-bold">{systemInfo?.host?.cpu_percent?.toFixed(1) || '0.0'}%</p><p className="text-xs text-gray-400 truncate">{systemInfo?.host?.cpu_model || ''}</p></CardBody></Card>
        <Card><CardBody><p className="text-sm text-gray-500">内存 / 磁盘</p><p className="text-xl font-bold">{systemInfo?.host?.memory_percent?.toFixed(1) || '0.0'}% / {systemInfo?.host?.disk_percent?.toFixed(1) || '0.0'}%</p><p className="text-xs text-gray-400">系统资源使用率</p></CardBody></Card>
        <Card><CardBody><p className="text-sm text-gray-500">网络传输</p><p className="font-bold text-emerald-600">↑ {formatRate(systemInfo?.host?.network_up_bps)}</p><p className="font-bold text-blue-600">↓ {formatRate(systemInfo?.host?.network_down_bps)}</p></CardBody></Card>
      </div>

      {/* 服务状态卡片 */}
      <Card className="border border-slate-200/70 shadow-sm dark:border-slate-800">
        <CardHeader className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold">sing-box 服务</h2>
            <Chip
              color={serviceStatus?.running ? 'success' : 'danger'}
              variant="flat"
              size="sm"
            >
              {serviceStatus?.running ? '运行中' : '已停止'}
            </Chip>
          </div>
          <div className="flex w-full flex-wrap gap-2 sm:w-auto">
            {serviceStatus?.running ? (
              <>
                <Button
                  size="sm"
                  color="danger"
                  variant="flat"
                  startContent={<Square className="w-4 h-4" />}
                  onPress={handleStop}
                >
                  停止
                </Button>
                <Button
                  size="sm"
                  color="primary"
                  variant="flat"
                  startContent={<RefreshCw className="w-4 h-4" />}
                  onPress={handleRestart}
                >
                  重启
                </Button>
              </>
            ) : (
              <Button
                size="sm"
                color="success"
                startContent={<Play className="w-4 h-4" />}
                onPress={handleStart}
              >
                启动
              </Button>
            )}
            <Button
              size="sm"
              color="primary"
              onPress={handleApplyConfig}
            >
              应用配置
            </Button>
          </div>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div>
              <p className="text-sm text-gray-500">版本</p>
              <div className="flex items-center gap-1">
                <p className="font-medium">
                  {serviceStatus?.version?.match(/version\s+([\d.]+)/)?.[1] || serviceStatus?.version || '-'}
                </p>
                {serviceStatus?.version && (
                  <Tooltip
                    content={
                      <div className="max-w-xs whitespace-pre-wrap text-xs p-1">
                        {serviceStatus.version}
                      </div>
                    }
                    placement="bottom"
                  >
                    <Info className="w-3.5 h-3.5 text-gray-400 cursor-help" />
                  </Tooltip>
                )}
              </div>
            </div>
            <div>
              <p className="text-sm text-gray-500">进程 ID</p>
              <p className="font-medium">{serviceStatus?.pid || '-'}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">状态</p>
              <p className="font-medium">
                {serviceStatus?.running ? '正常运行' : '未运行'}
              </p>
            </div>
          </div>
        </CardBody>
      </Card>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardBody className="flex flex-row items-center gap-4">
            <div className="p-3 bg-blue-100 dark:bg-blue-900 rounded-lg">
              <Wifi className="w-6 h-6 text-blue-600 dark:text-blue-300" />
            </div>
            <div>
              <p className="text-sm text-gray-500">订阅数量</p>
              <p className="text-2xl font-bold">{enabledSubs} / {subscriptions.length}</p>
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardBody className="flex flex-row items-center gap-4">
            <div className="p-3 bg-green-100 dark:bg-green-900 rounded-lg">
              <HardDrive className="w-6 h-6 text-green-600 dark:text-green-300" />
            </div>
            <div>
              <p className="text-sm text-gray-500">节点总数</p>
              <p className="text-2xl font-bold">{totalNodes}</p>
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardBody className="flex flex-row items-center gap-4">
            <div className="p-3 bg-purple-100 dark:bg-purple-900 rounded-lg">
              <Cpu className="w-6 h-6 text-purple-600 dark:text-purple-300" />
            </div>
            <div>
              <p className="text-sm text-gray-500">CMSingBox 资源</p>
              <p className="text-lg font-bold">
                {systemInfo?.sbm ? (
                  <>
                    <span className="text-sm font-normal text-gray-500">CPU </span>
                    {systemInfo.sbm.cpu_percent.toFixed(1)}%
                    <span className="text-sm font-normal text-gray-500 ml-2">内存 </span>
                    {systemInfo.sbm.memory_mb.toFixed(1)}MB
                  </>
                ) : '-'}
              </p>
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardBody className="flex flex-row items-center gap-4">
            <div className="p-3 bg-orange-100 dark:bg-orange-900 rounded-lg">
              <Activity className="w-6 h-6 text-orange-600 dark:text-orange-300" />
            </div>
            <div>
              <p className="text-sm text-gray-500">sing-box 资源</p>
              <p className="text-lg font-bold">
                {serviceStatus?.running && systemInfo?.singbox ? (
                  <>
                    <span className="text-sm font-normal text-gray-500">CPU </span>
                    {systemInfo.singbox.cpu_percent.toFixed(1)}%
                    <span className="text-sm font-normal text-gray-500 ml-2">内存 </span>
                    {systemInfo.singbox.memory_mb.toFixed(1)}MB
                  </>
                ) : (
                  <span className="text-gray-400">未运行</span>
                )}
              </p>
            </div>
          </CardBody>
        </Card>
      </div>

      {/* 订阅列表预览 */}
      <Card>
        <CardHeader>
          <h2 className="text-lg font-semibold">订阅概览</h2>
        </CardHeader>
        <CardBody>
          {subscriptions.length === 0 ? (
            <p className="text-gray-500 text-center py-4">暂无订阅，请前往节点页面添加</p>
          ) : (
            <div className="space-y-3">
              {subscriptions.map((sub) => (
                <div
                  key={sub.id}
                  className="flex flex-col items-start justify-between gap-2 rounded-lg bg-gray-50 p-3 dark:bg-gray-800 sm:flex-row sm:items-center"
                >
                  <div className="flex items-center gap-3">
                    <Chip
                      size="sm"
                      color={sub.enabled ? 'success' : 'default'}
                      variant="dot"
                    >
                      {sub.name}
                    </Chip>
                    <span className="text-sm text-gray-500">
                      {sub.node_count} 个节点
                    </span>
                  </div>
                  <span className="text-sm text-gray-400">
                    更新于 {new Date(sub.updated_at).toLocaleString()}
                  </span>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>

      {/* 错误提示模态框 */}
      <Modal isOpen={errorModal.isOpen} onClose={() => setErrorModal({ ...errorModal, isOpen: false })}>
        <ModalContent>
          <ModalHeader className="text-danger">{errorModal.title}</ModalHeader>
          <ModalBody>
            <p className="whitespace-pre-wrap text-sm">{errorModal.message}</p>
          </ModalBody>
          <ModalFooter>
            <Button color="primary" onPress={() => setErrorModal({ ...errorModal, isOpen: false })}>
              确定
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}
