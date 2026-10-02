import React, { useState } from 'react';
import { Shield, Building2, Key, User, ArrowRight, PhoneCall } from 'lucide-react';
import { Tenant } from '../../types';
import { Button } from '../ui/Button';

interface LoginScreenProps {
  tenants: Tenant[];
  onAdminLogin: (password: string) => boolean;
  onCustomerLogin: (tenantId: string, apiKeyOrPassword: string) => boolean;
}

export const LoginScreen: React.FC<LoginScreenProps> = ({
  tenants,
  onAdminLogin,
  onCustomerLogin,
}) => {
  const [activeTab, setActiveTab] = useState<'customer' | 'admin'>('customer');
  
  // Admin form
  const [adminUsername, setAdminUsername] = useState('admin');
  const [adminPassword, setAdminPassword] = useState('');
  const [adminError, setAdminError] = useState('');

  // Customer form
  const [selectedTenantId, setSelectedTenantId] = useState<string>(tenants[0]?.id || 'eczane_hayat');
  const [customerAuthKey, setCustomerAuthKey] = useState('');
  const [customerError, setCustomerError] = useState('');

  const handleAdminSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setAdminError('');
    const success = onAdminLogin(adminPassword);
    if (!success) {
      setAdminError('Geçersiz yönetici şifresi! (Varsayılan: admin123)');
    }
  };

  const handleCustomerSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setCustomerError('');
    if (!selectedTenantId) {
      setCustomerError('Lütfen bir müşteri hesabı seçiniz.');
      return;
    }
    const success = onCustomerLogin(selectedTenantId, customerAuthKey);
    if (!success) {
      setCustomerError('Müşteri kimlik doğrulaması başarısız oldu.');
    }
  };

  return (
    <div className="min-h-screen w-full flex items-center justify-center bg-radial from-card to-background p-4 sm:p-6 lg:p-8">
      {/* Background Glow */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 left-1/2 -translate-x-1/2 w-[600px] h-[600px] bg-primary/10 rounded-full blur-3xl"></div>
      </div>

      <div className="relative w-full max-w-md bg-card/90 backdrop-blur-xl border border-border shadow-2xl rounded-2xl p-6 sm:p-8 flex flex-col gap-6">
        {/* Brand Header */}
        <div className="flex flex-col items-center text-center gap-2">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-tr from-primary to-indigo-500 flex items-center justify-center text-white shadow-lg shadow-primary/30">
            <PhoneCall className="w-6 h-6 animate-pulse" />
          </div>
          <div>
            <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-foreground">
              AI Call Center Platform
            </h1>
            <p className="text-xs sm:text-sm text-muted-foreground mt-0.5">
              Carrier-Grade Multi-Tenant CPaaS & Visual Studio
            </p>
          </div>
        </div>

        {/* Tab Switcher: Customer vs Admin */}
        <div className="grid grid-cols-2 p-1 bg-secondary/70 rounded-xl border border-border text-xs font-semibold">
          <button
            type="button"
            onClick={() => {
              setActiveTab('customer');
              setCustomerError('');
            }}
            className={`flex items-center justify-center gap-2 py-2.5 rounded-lg transition-all ${
              activeTab === 'customer'
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Building2 className="w-4 h-4" />
            <span>Müşteri Portalı</span>
          </button>
          <button
            type="button"
            onClick={() => {
              setActiveTab('admin');
              setAdminError('');
            }}
            className={`flex items-center justify-center gap-2 py-2.5 rounded-lg transition-all ${
              activeTab === 'admin'
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Shield className="w-4 h-4" />
            <span>Süper Admin</span>
          </button>
        </div>

        {/* Customer Login Form */}
        {activeTab === 'customer' && (
          <form onSubmit={handleCustomerSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Building2 className="w-3.5 h-3.5 text-muted-foreground" />
                <span>Müşteri / Tenant Seçin</span>
              </label>
              <select
                value={selectedTenantId}
                onChange={(e) => setSelectedTenantId(e.target.value)}
                className="w-full h-10 px-3 rounded-xl border border-border bg-background text-sm font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-primary/40"
              >
                {tenants.map((t) => (
                  <option key={t.id} value={t.id} className="bg-card text-foreground">
                    {t.name} ({t.id})
                  </option>
                ))}
              </select>
              <p className="text-[11px] text-muted-foreground">
                Giriş yapacağınız işletme hesabını seçin (örn: Hayat Eczanesi, Jet Kargo).
              </p>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Key className="w-3.5 h-3.5 text-muted-foreground" />
                <span>Erişim Şifresi veya API Key</span>
              </label>
              <input
                type="password"
                value={customerAuthKey}
                onChange={(e) => setCustomerAuthKey(e.target.value)}
                placeholder="İşletme şifresi veya boş bırakın (Demo)"
                className="w-full h-10 px-3 rounded-xl border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/40 placeholder:text-muted-foreground/50"
              />
              <p className="text-[10px] text-muted-foreground">
                Demo modunda doğrudan giriş yapabilirsiniz.
              </p>
            </div>

            {customerError && (
              <div className="p-2.5 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs font-medium">
                {customerError}
              </div>
            )}

            <Button type="submit" variant="primary" className="w-full h-10 gap-2 mt-2">
              <span>Müşteri Paneline Giriş Yap</span>
              <ArrowRight className="w-4 h-4" />
            </Button>
          </form>
        )}

        {/* Admin Login Form */}
        {activeTab === 'admin' && (
          <form onSubmit={handleAdminSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-foreground flex items-center gap-1.5">
                <User className="w-3.5 h-3.5 text-muted-foreground" />
                <span>Yönetici Kullanıcı Adı</span>
              </label>
              <input
                type="text"
                value={adminUsername}
                onChange={(e) => setAdminUsername(e.target.value)}
                className="w-full h-10 px-3 rounded-xl border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/40"
                placeholder="admin"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-foreground flex items-center gap-1.5">
                <Key className="w-3.5 h-3.5 text-muted-foreground" />
                <span>Admin Giriş Şifresi</span>
              </label>
              <input
                type="password"
                value={adminPassword}
                onChange={(e) => setAdminPassword(e.target.value)}
                placeholder="admin123"
                className="w-full h-10 px-3 rounded-xl border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/40"
              />
              <p className="text-[11px] text-muted-foreground">
                Varsayılan şifre: <code className="text-primary font-mono font-medium">admin123</code>
              </p>
            </div>

            {adminError && (
              <div className="p-2.5 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs font-medium">
                {adminError}
              </div>
            )}

            <Button type="submit" variant="primary" className="w-full h-10 gap-2 mt-2">
              <span>Süper Admin Paneline Giriş Yap</span>
              <ArrowRight className="w-4 h-4" />
            </Button>
          </form>
        )}

        {/* Footer info */}
        <div className="pt-2 border-t border-border/50 text-center">
          <p className="text-[11px] text-muted-foreground">
            Tarık Öğüt • icell.cloud Telecom Systems
          </p>
        </div>
      </div>
    </div>
  );
};
