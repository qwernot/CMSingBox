import { useEffect, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  Activity, ChevronRight, FileText, Globe, LayoutDashboard, LogOut, Menu,
  Moon, Radar, ScrollText, Settings, ShieldCheck, SlidersHorizontal, Sun, X,
} from 'lucide-react';
import { useStore } from '../store';
import { authApi } from '../api';

const menuGroups = [
  { label: '监控', items: [
    { path: '/', icon: LayoutDashboard, label: '仪表盘' },
    { path: '/dns', icon: Radar, label: 'DNS 监控' },
  ] },
  { label: '配置', items: [
    { path: '/subscriptions', icon: Globe, label: '节点与订阅' },
    { path: '/rules', icon: FileText, label: '路由规则' },
    { path: '/advanced', icon: SlidersHorizontal, label: '高级配置' },
  ] },
  { label: '系统', items: [
    { path: '/settings', icon: Settings, label: '系统设置' },
    { path: '/logs', icon: ScrollText, label: '运行日志' },
  ] },
];

interface LayoutProps { children: React.ReactNode; onLogout: () => void; }

export default function Layout({ children, onLogout }: LayoutProps) {
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [darkMode, setDarkMode] = useState(() => localStorage.getItem('sbm-theme') === 'dark' || (!localStorage.getItem('sbm-theme') && matchMedia('(prefers-color-scheme: dark)').matches));
  const { settings, fetchSettings, serviceStatus, fetchServiceStatus } = useStore();

  useEffect(() => {
    if (!settings) fetchSettings();
    if (!serviceStatus) fetchServiceStatus();
  }, []);
  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode);
    localStorage.setItem('sbm-theme', darkMode ? 'dark' : 'light');
  }, [darkMode]);
  useEffect(() => setMobileOpen(false), [location.pathname]);

  const clashApiPort = settings?.clash_api_port || 9090;
  const zashboardURL = (() => {
    const host = window.location.hostname;
    const baseUrl = `${window.location.protocol}//${host}:${clashApiPort}/ui/`;
    const params = new URLSearchParams({ hostname: host, port: String(clashApiPort) });
    if (settings?.clash_api_secret) params.set('secret', settings.clash_api_secret);
    return `${baseUrl}#/setup?${params.toString()}`;
  })();

  const logout = async () => {
    try { await authApi.logout(); } finally { onLogout(); }
  };

  const sidebar = (
    <div className="flex h-full flex-col bg-slate-950 text-slate-200">
      <div className="flex h-20 items-center gap-3 border-b border-white/10 px-6">
        <div className="grid h-10 w-10 place-items-center rounded-xl bg-gradient-to-br from-blue-500 to-cyan-400 shadow-lg shadow-blue-500/20">
          <Activity className="h-6 w-6 text-white" />
        </div>
        <div><p className="text-lg font-bold tracking-tight text-white">CMSingBox</p><p className="text-[11px] tracking-widest text-slate-500">NETWORK CONSOLE</p></div>
        <button className="ml-auto rounded-lg p-2 text-slate-400 hover:bg-white/10 lg:hidden" onClick={() => setMobileOpen(false)} aria-label="关闭菜单"><X className="h-5 w-5" /></button>
      </div>

      <nav className="flex-1 overflow-y-auto px-4 py-5">
        {menuGroups.map((group) => (
          <div key={group.label} className="mb-6">
            <p className="mb-2 px-3 text-[11px] font-semibold tracking-[0.2em] text-slate-500">{group.label}</p>
            <div className="space-y-1">
              {group.items.map((item) => {
                const active = location.pathname === item.path;
                const Icon = item.icon;
                return <Link key={item.path} to={item.path} className={`group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition-all ${active ? 'bg-blue-600 text-white shadow-lg shadow-blue-950/40' : 'text-slate-400 hover:bg-white/[0.07] hover:text-white'}`}>
                  <Icon className="h-[18px] w-[18px]" /><span className="flex-1">{item.label}</span>{active && <ChevronRight className="h-4 w-4 opacity-70" />}
                </Link>;
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="border-t border-white/10 p-4">
        <div className="mb-3 flex items-center gap-3 rounded-xl bg-white/[0.05] p-3">
          <span className={`h-2.5 w-2.5 rounded-full ${serviceStatus?.running ? 'bg-emerald-400 shadow-[0_0_12px_rgba(52,211,153,.8)]' : 'bg-rose-400'}`} />
          <div className="min-w-0 flex-1"><p className="text-xs font-medium text-white">sing-box {serviceStatus?.running ? '运行中' : '已停止'}</p><p className="truncate text-[11px] text-slate-500">CMSingBox v{serviceStatus?.sbm_version || '-'}</p></div>
        </div>
        <a href={zashboardURL} target="_blank" rel="noopener noreferrer" className="mb-1 flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-400 hover:bg-white/[0.07] hover:text-white"><ShieldCheck className="h-4 w-4" />打开代理控制台</a>
        <button onClick={() => setDarkMode(!darkMode)} className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-400 hover:bg-white/[0.07] hover:text-white">{darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}{darkMode ? '浅色主题' : '深色主题'}</button>
        <button onClick={logout} className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-400 hover:bg-rose-500/10 hover:text-rose-400"><LogOut className="h-4 w-4" />退出登录</button>
      </div>
    </div>
  );

  return <div className="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-900 dark:text-slate-100">
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-72 lg:block">{sidebar}</aside>
    {mobileOpen && <><button className="fixed inset-0 z-40 bg-slate-950/60 backdrop-blur-sm lg:hidden" onClick={() => setMobileOpen(false)} aria-label="关闭菜单遮罩" /><aside className="fixed inset-y-0 left-0 z-50 w-[min(86vw,320px)] shadow-2xl lg:hidden">{sidebar}</aside></>}
    <header className="fixed inset-x-0 top-0 z-30 flex h-16 items-center border-b border-slate-200/80 bg-white/90 px-4 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/85 lg:hidden">
      <button onClick={() => setMobileOpen(true)} className="rounded-xl border border-slate-200 p-2 text-slate-700 dark:border-slate-700 dark:text-slate-200" aria-label="打开菜单"><Menu className="h-5 w-5" /></button>
      <div className="ml-3"><p className="font-bold">CMSingBox</p><p className="text-[10px] text-slate-500">网络管理控制台</p></div>
      <span className={`ml-auto h-2.5 w-2.5 rounded-full ${serviceStatus?.running ? 'bg-emerald-500' : 'bg-rose-500'}`} />
    </header>
    <main className="min-w-0 pt-16 lg:ml-72 lg:pt-0"><div className="mx-auto w-full max-w-[1600px] p-4 sm:p-6 lg:p-8">{children}</div></main>
  </div>;
}
