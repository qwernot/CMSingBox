import { useState } from 'react';
import type { FormEvent } from 'react';
import { Activity, ArrowRight, Eye, EyeOff, LockKeyhole, ShieldCheck, User } from 'lucide-react';
import { authApi } from '../api';

interface LoginProps { onSuccess: () => void; }

export default function Login({ onSuccess }: LoginProps) {
  const [username, setUsername] = useState(() => localStorage.getItem('cmsingbox-login-name') || 'admin');
  const [password, setPassword] = useState('');
  const [rememberName, setRememberName] = useState(() => Boolean(localStorage.getItem('cmsingbox-login-name')));
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError('');
    try {
      await authApi.login(username.trim(), password);
      if (rememberName) localStorage.setItem('cmsingbox-login-name', username.trim());
      else localStorage.removeItem('cmsingbox-login-name');
      onSuccess();
    } catch (err: unknown) {
      const message = typeof err === 'object' && err !== null && 'response' in err
        ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
        : undefined;
      setError(message || '登录失败，请检查账号、密码或网络连接');
    } finally { setLoading(false); }
  };

  return <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#061b2b] p-4 sm:p-6">
    <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_15%,rgba(6,182,212,.42),transparent_31%),radial-gradient(circle_at_82%_70%,rgba(37,99,235,.34),transparent_34%),linear-gradient(135deg,#052f46_0%,#061522_48%,#063c51_100%)]" />
    <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(rgba(255,255,255,.045)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.045)_1px,transparent_1px)] [background-size:46px_46px]" />
    <div className="absolute -left-20 top-16 h-72 w-72 rounded-full bg-cyan-400/20 blur-3xl" />
    <div className="absolute -right-20 bottom-10 h-80 w-80 rounded-full bg-blue-500/20 blur-3xl" />

    <main className="relative w-full max-w-[460px] rounded-[30px] border border-white/15 bg-slate-950/55 px-6 py-8 shadow-[0_30px_90px_rgba(0,0,0,.45)] backdrop-blur-2xl sm:px-10 sm:py-10">
      <div className="mb-8 text-center">
        <div className="mx-auto mb-5 grid h-16 w-16 place-items-center rounded-2xl bg-gradient-to-br from-blue-500 to-cyan-400 shadow-xl shadow-blue-500/25">
          <Activity className="h-9 w-9 text-white" />
        </div>
        <h1 className="text-3xl font-bold tracking-tight text-white">CMSingBox</h1>
        <p className="mt-2 text-sm tracking-wide text-slate-400">Sing-box Network Management Console</p>
      </div>

      <form onSubmit={submit} className="space-y-4">
        <label className="block rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-2 transition focus-within:border-blue-400/70 focus-within:bg-slate-950/70">
          <span className="block text-[11px] text-slate-500">用户名</span>
          <span className="flex items-center gap-3"><User className="h-4 w-4 text-blue-400" /><input aria-label="用户名" value={username} onChange={(e) => setUsername(e.target.value)} className="min-w-0 flex-1 bg-transparent py-1.5 text-white outline-none placeholder:text-slate-600" autoComplete="username" placeholder="请输入用户名" required /></span>
        </label>
        <label className="block rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-2 transition focus-within:border-blue-400/70 focus-within:bg-slate-950/70">
          <span className="block text-[11px] text-slate-500">密码</span>
          <span className="flex items-center gap-3"><LockKeyhole className="h-4 w-4 text-blue-400" /><input aria-label="密码" type={showPassword ? 'text' : 'password'} value={password} onChange={(e) => setPassword(e.target.value)} className="min-w-0 flex-1 bg-transparent py-1.5 text-white outline-none placeholder:text-slate-600" autoComplete="current-password" placeholder="请输入密码" required autoFocus /><button type="button" aria-label={showPassword ? '隐藏密码' : '显示密码'} onClick={() => setShowPassword(!showPassword)} className="rounded-lg p-1 text-slate-500 hover:bg-white/10 hover:text-white">{showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}</button></span>
        </label>

        <label className="flex cursor-pointer items-center gap-2 px-1 text-sm text-slate-400"><input type="checkbox" checked={rememberName} onChange={(e) => setRememberName(e.target.checked)} className="h-4 w-4 rounded border-slate-600 bg-transparent accent-blue-500" /><span>记住用户名</span></label>
        {error && <p role="alert" className="rounded-xl border border-rose-500/25 bg-rose-500/10 px-4 py-3 text-sm text-rose-300">{error}</p>}
        <button type="submit" disabled={loading} className="group flex w-full items-center justify-center gap-2 rounded-2xl bg-blue-600 px-4 py-3.5 font-semibold text-white shadow-lg shadow-blue-950/40 transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-60">{loading ? '正在登录…' : '登录'}{!loading && <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />}</button>
      </form>

      <div className="mt-8 flex items-center gap-4 text-slate-600"><span className="h-px flex-1 bg-white/10" /><ShieldCheck className="h-4 w-4" /><span className="h-px flex-1 bg-white/10" /></div>
      <p className="mt-4 text-center text-xs text-slate-500">安全管理入口 · 登录后可在系统设置中修改密码</p>
      <div className="mt-4 flex items-center justify-center gap-4 text-xs"><a href="/docs/user" className="text-cyan-400/80 transition hover:text-cyan-300">用户文档</a><span className="text-white/15">·</span><a href="/docs/admin" className="text-cyan-400/80 transition hover:text-cyan-300">管理员文档</a></div>
    </main>
    <p className="absolute bottom-5 text-[11px] tracking-wide text-white/30">CMSingBox · Private Network Console</p>
  </div>;
}
