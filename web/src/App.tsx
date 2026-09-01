import { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Subscriptions from './pages/Subscriptions';
import Rules from './pages/Rules';
import Settings from './pages/Settings';
import Logs from './pages/Logs';
import { ToastContainer } from './components/Toast';
import Login from './pages/Login';
import DNSMonitor from './pages/DNSMonitor';
import Advanced from './pages/Advanced';
import { authApi } from './api';
import Docs from './pages/Docs';

function App() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null);

  useEffect(() => {
    authApi.status().then(() => setAuthenticated(true)).catch(() => setAuthenticated(false));
    const handleUnauthorized = () => setAuthenticated(false);
    window.addEventListener('sbm:unauthorized', handleUnauthorized);
    return () => window.removeEventListener('sbm:unauthorized', handleUnauthorized);
  }, []);

  const docsMode = window.location.pathname === '/docs/admin'
    ? 'admin'
    : window.location.pathname === '/docs/user' ? 'user' : null;

  if (docsMode) {
    return <BrowserRouter><Docs mode={docsMode} /></BrowserRouter>;
  }

  if (authenticated === null) {
    return <div className="min-h-screen bg-slate-950 flex items-center justify-center text-slate-400">正在加载…</div>;
  }

  if (!authenticated) {
    return <Login onSuccess={() => setAuthenticated(true)} />;
  }

  return (
    <BrowserRouter>
      <ToastContainer />
      <Layout onLogout={() => setAuthenticated(false)}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/subscriptions" element={<Subscriptions />} />
          <Route path="/dns" element={<DNSMonitor />} />
          <Route path="/advanced" element={<Advanced />} />
          <Route path="/rules" element={<Rules />} />
          <Route path="/logs" element={<Logs />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}

export default App;
