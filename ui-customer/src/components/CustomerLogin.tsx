import React, { useState } from 'react';
import { Building2, Key, ArrowRight, PhoneCall, Sparkles } from 'lucide-react';
import { Button } from './ui/Button';
import { Tenant } from '../types';

interface CustomerLoginProps {
  tenants: Tenant[];
  onLogin: (tenantId: string, apiKey: string) => void;
}

export const CustomerLogin: React.FC<CustomerLoginProps> = ({ tenants, onLogin }) => {
  const [selectedTenantId, setSelectedTenantId] = useState<string>(tenants[0]?.id || 'eczane_hayat');
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTenantId) {
      setError('Lütfen bir müşteri hesabı seçiniz.');
      return;
    }
    onLogin(selectedTenantId, apiKey);
  };

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-slate-950 text-slate-100 p-4">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 shadow-2xl rounded-2xl p-8 flex flex-col gap-6">
        <div className="flex flex-col items-center text-center gap-3">
          <div className="w-14 h-14 rounded-2xl bg-indigo-600/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400 shadow-lg shadow-indigo-950">
            <PhoneCall className="w-7 h-7" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">
              Customer Cloud Studio
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Visual Workflow Builder, AI Voice Persona & WebRTC Softphone
            </p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
              <Building2 className="w-3.5 h-3.5 text-slate-400" />
              <span>İşletme / Tenant Seçin</span>
            </label>
            <select
              value={selectedTenantId}
              onChange={(e) => setSelectedTenantId(e.target.value)}
              className="w-full h-11 px-3.5 rounded-xl border border-slate-800 bg-slate-950 text-sm font-medium text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
            >
              {tenants.map((t) => (
                <option key={t.id} value={t.id} className="bg-slate-900 text-white">
                  {t.name} ({t.id})
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
              <Key className="w-3.5 h-3.5 text-slate-400" />
              <span>Müşteri API Anahtarı / Şifresi</span>
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="API Key veya boş bırakın (Demo)"
              className="w-full h-11 px-3.5 rounded-xl border border-slate-800 bg-slate-950 text-sm text-white focus:outline-none focus:ring-2 focus:ring-indigo-500/50 placeholder:text-slate-600"
            />
            <p className="text-[11px] text-slate-500">
              Demo hesabınızla doğrudan giriş yapabilirsiniz.
            </p>
          </div>

          {error && (
            <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs font-medium">
              {error}
            </div>
          )}

          <Button type="submit" variant="primary" className="w-full h-11 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold gap-2 mt-2 shadow-lg shadow-indigo-900/30">
            <span>Müşteri Paneline Giriş Yap</span>
            <ArrowRight className="w-4 h-4" />
          </Button>
        </form>

        <div className="pt-3 border-t border-slate-800/80 text-center flex items-center justify-center gap-1.5 text-[11px] text-slate-500">
          <Sparkles className="w-3.5 h-3.5 text-indigo-400" />
          <span>Powered by Gemini 2.0 Live & FreeSWITCH</span>
        </div>
      </div>
    </div>
  );
};
