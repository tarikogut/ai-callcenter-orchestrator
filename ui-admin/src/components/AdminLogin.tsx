import React, { useState } from 'react';
import { Shield, Key, User, ArrowRight, Lock } from 'lucide-react';
import { Button } from './ui/Button';

interface AdminLoginProps {
  onLogin: (username: string, pass: string) => boolean;
}

export const AdminLogin: React.FC<AdminLoginProps> = ({ onLogin }) => {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    const success = onLogin(username, password);
    if (!success) {
      setError('Geçersiz yönetici bilgileri! (Varsayılan: admin / admin123)');
    }
  };

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-slate-950 text-slate-100 p-4">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 shadow-2xl rounded-2xl p-8 flex flex-col gap-6">
        <div className="flex flex-col items-center text-center gap-3">
          <div className="w-14 h-14 rounded-2xl bg-rose-600/20 border border-rose-500/30 flex items-center justify-center text-rose-500 shadow-lg shadow-rose-950">
            <Shield className="w-7 h-7" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">
              Carrier Super Admin
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Telecom Infrastructure, Multi-Tenant OCS & Routing Console
            </p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
              <User className="w-3.5 h-3.5 text-slate-400" />
              <span>Yönetici Kullanıcı Adı</span>
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full h-11 px-3.5 rounded-xl border border-slate-800 bg-slate-950 text-sm text-white focus:outline-none focus:ring-2 focus:ring-rose-500/50"
              placeholder="admin"
            />
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
              <Key className="w-3.5 h-3.5 text-slate-400" />
              <span>Yönetici Şifresi</span>
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full h-11 px-3.5 rounded-xl border border-slate-800 bg-slate-950 text-sm text-white focus:outline-none focus:ring-2 focus:ring-rose-500/50"
              placeholder="admin123"
            />
            <p className="text-[11px] text-slate-500">
              Varsayılan kimlik: <code className="text-rose-400 font-mono">admin</code> / <code className="text-rose-400 font-mono">admin123</code>
            </p>
          </div>

          {error && (
            <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs font-medium flex items-center gap-2">
              <Lock className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <Button type="submit" variant="primary" className="w-full h-11 bg-rose-600 hover:bg-rose-700 text-white font-semibold gap-2 mt-2 shadow-lg shadow-rose-900/30">
            <span>Admin Konsoluna Giriş Yap</span>
            <ArrowRight className="w-4 h-4" />
          </Button>
        </form>

        <div className="pt-3 border-t border-slate-800/80 text-center">
          <p className="text-[11px] text-slate-500 font-mono">
            icell.cloud • Operator Infrastructure
          </p>
        </div>
      </div>
    </div>
  );
};
