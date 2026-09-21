import { useState } from 'react';
import type { FormEvent } from 'react';
import { ArrowRight, Eye, EyeOff, LockKeyhole, ShieldCheck, User } from 'lucide-react';
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

  return <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#087e9f] p-4 sm:p-6">
    <div className="absolute inset-0 bg-cover bg-center" style={{ backgroundImage: "url('/api/login-background'), url('/login-ocean.svg')" }} />
    <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_42%,transparent_0%,rgba(1,24,40,.08)_52%,rgba(1,20,34,.38)_100%)]" />

    <main className="relative w-full max-w-[448px] rounded-[28px] border border-white/20 bg-[#073246]/80 px-7 py-9 shadow-[0_28px_90px_rgba(0,25,43,.45)] backdrop-blur-2xl sm:px-9 sm:py-10">
      <div className="mb-8 text-center">
        <img src="/favicon.svg?v=1.1.3" alt="CMSingBox" className="mx-auto mb-4 h-16 w-16" />
        <h1 className="text-[28px] font-semibold tracking-tight text-white">CMSingBox</h1>
        <p className="mt-1.5 text-sm tracking-wide text-white/45">Sing-box Network Proxy Box</p>
      </div>

      <form onSubmit={submit} className="space-y-4">
        <label className="block rounded-2xl border border-white/10 bg-[#061f30]/45 px-4 py-2.5 transition focus-within:border-blue-400/65 focus-within:bg-[#061f30]/65">
          <span className="block pl-7 text-[11px] text-white/35">用户名</span>
          <span className="flex items-center gap-3"><span className="grid h-7 w-7 place-items-center rounded-full bg-blue-500/15"><User className="h-4 w-4 text-blue-400" /></span><input aria-label="用户名" value={username} onChange={(e) => setUsername(e.target.value)} className="min-w-0 flex-1 bg-transparent py-1 text-[15px] text-white outline-none placeholder:text-white/25" autoComplete="username" placeholder="请输入用户名" required /></span>
        </label>
        <label className="block rounded-2xl border border-white/10 bg-[#061f30]/45 px-4 py-2.5 transition focus-within:border-blue-400/65 focus-within:bg-[#061f30]/65">
          <span className="block pl-7 text-[11px] text-white/35">密码</span>
          <span className="flex items-center gap-3"><LockKeyhole className="h-4 w-4 text-white/30" /><input aria-label="密码" type={showPassword ? 'text' : 'password'} value={password} onChange={(e) => setPassword(e.target.value)} className="min-w-0 flex-1 bg-transparent py-1 text-[15px] text-white outline-none placeholder:text-white/25" autoComplete="current-password" placeholder="请输入密码" required autoFocus /><button type="button" aria-label={showPassword ? '隐藏密码' : '显示密码'} onClick={() => setShowPassword(!showPassword)} className="rounded-lg p-1.5 text-white/30 transition hover:bg-white/10 hover:text-white">{showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}</button></span>
        </label>

        <label className="flex cursor-pointer items-center gap-2 px-1 text-sm text-white/50"><input type="checkbox" checked={rememberName} onChange={(e) => setRememberName(e.target.checked)} className="h-4 w-4 rounded border-white/30 bg-transparent accent-blue-500" /><span>记住用户名</span></label>
        {error && <p role="alert" className="rounded-xl border border-rose-500/25 bg-rose-500/10 px-4 py-3 text-sm text-rose-300">{error}</p>}
        <button type="submit" disabled={loading} className="group flex w-full items-center justify-center gap-2 rounded-[14px] bg-[#087cff] px-4 py-3.5 font-semibold text-white shadow-[0_10px_28px_rgba(0,93,255,.28)] transition hover:bg-[#1988ff] disabled:cursor-not-allowed disabled:opacity-60">{loading ? '正在登录…' : '登录'}{!loading && <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />}</button>
      </form>

      <div className="mt-8 flex items-center gap-4 text-white/55"><span className="h-px flex-1 bg-white/15" /><ShieldCheck className="h-4 w-4" /><span className="h-px flex-1 bg-white/15" /></div>
      <div className="mt-4 flex items-center justify-center text-xs"><a href="https://666228.xyz/CM/" target="_blank" rel="noopener noreferrer" className="text-white/40 transition hover:text-white/75">用户使用文档</a></div>
    </main>
  </div>;
}
