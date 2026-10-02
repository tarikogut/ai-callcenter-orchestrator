import React, { useState, useEffect } from 'react';
import {
  Shield,
  Users,
  Hash,
  Activity,
  LogOut,
  Radio,
  Sun,
  Moon,
} from 'lucide-react';
import { TenantManagement } from './components/admin/TenantManagement';
import { GlobalDidPool } from './components/admin/GlobalDidPool';
import { DiameterBilling } from './components/admin/DiameterBilling';
import { AdminLogin } from './components/AdminLogin';
import { Button } from './components/ui/Button';
import { api } from './api/client';
import { Tenant, DIDNumber, DiameterMetric } from './types';

type AdminTab = 'tenants' | 'dids' | 'diameter';

export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(() => {
    return localStorage.getItem('admin_auth') === 'true';
  });

  const [activeTab, setActiveTab] = useState<AdminTab>('tenants');
  const [isDark, setIsDark] = useState<boolean>(() => localStorage.getItem('theme') !== 'light');

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [dids, setDids] = useState<DIDNumber[]>([]);
  const [metrics, setMetrics] = useState<DiameterMetric | null>(null);

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    }
  }, [isDark]);

  useEffect(() => {
    if (!isAuthenticated) return;
    const load = async () => {
      const [tList, dList, mData] = await Promise.all([
        api.getTenants(),
        api.getDids(),
        api.getDiameterMetrics(),
      ]);
      setTenants(tList);
      setDids(dList);
      setMetrics(mData);
    };
    load();
  }, [isAuthenticated]);

  const handleLogin = (_user: string, pass: string): boolean => {
    if (pass === 'admin123' || pass === '') {
      setIsAuthenticated(true);
      localStorage.setItem('admin_auth', 'true');
      return true;
    }
    return false;
  };

  const handleLogout = () => {
    setIsAuthenticated(false);
    localStorage.removeItem('admin_auth');
  };

  if (!isAuthenticated) {
    return <AdminLogin onLogin={handleLogin} />;
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col font-sans">
      {/* Super Admin Top Header */}
      <header className="sticky top-0 z-40 w-full border-b border-border bg-card/90 backdrop-blur px-6 h-16 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="w-10 h-10 rounded-xl bg-rose-600 flex items-center justify-center text-white shadow-md shadow-rose-900/30">
            <Shield className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-base tracking-tight">Super Admin Platform</span>
              <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-rose-500/10 text-rose-500 border border-rose-500/20">
                Operator Tier-1
              </span>
            </div>
            <p className="text-[11px] text-muted-foreground font-medium">
              Multi-Tenant CPaaS Telecom Backbone & OCS
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl border border-border bg-secondary/30 text-xs font-mono">
            <Radio className="w-3.5 h-3.5 text-emerald-500 animate-pulse" />
            <span className="text-muted-foreground">FreeSWITCH / Kamailio:</span>
            <span className="text-emerald-500 font-bold">ONLINE</span>
          </div>

          <button
            onClick={() => setIsDark(!isDark)}
            className="p-2 rounded-xl border border-border bg-secondary/50 hover:bg-secondary text-muted-foreground transition-colors"
          >
            {isDark ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4" />}
          </button>

          <Button
            variant="outline"
            size="sm"
            onClick={handleLogout}
            className="gap-1.5 text-xs text-rose-500 hover:bg-rose-500/10 border-border"
          >
            <LogOut className="w-3.5 h-3.5" />
            <span>Oturumu Kapat</span>
          </Button>
        </div>
      </header>

      {/* Main Admin Content */}
      <div className="flex flex-1 overflow-hidden">
        <aside className="w-64 border-r border-border bg-card/60 flex flex-col shrink-0 select-none">
          <div className="p-4 border-b border-border/50">
            <div className="text-xs font-bold uppercase tracking-wider text-rose-500 px-2 flex items-center gap-1.5">
              <Shield className="w-3.5 h-3.5" />
              <span>Admin Modülleri</span>
            </div>
          </div>
          <nav className="p-3 space-y-1.5 flex-1">
            <button
              onClick={() => setActiveTab('tenants')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                activeTab === 'tenants'
                  ? 'bg-rose-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Users className="w-4 h-4" />
              <span>Tenant (Müşteri) Yönetimi</span>
            </button>

            <button
              onClick={() => setActiveTab('dids')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                activeTab === 'dids'
                  ? 'bg-rose-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Hash className="w-4 h-4" />
              <span>Global DID Numara Havuzu</span>
            </button>

            <button
              onClick={() => setActiveTab('diameter')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                activeTab === 'diameter'
                  ? 'bg-rose-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Activity className="w-4 h-4" />
              <span>Diameter Ro (RFC 4006) Şarj</span>
            </button>
          </nav>
        </aside>

        <main className="flex-1 overflow-y-auto p-6 max-w-7xl mx-auto w-full">
          {activeTab === 'tenants' && (
            <TenantManagement
              tenants={tenants}
              onToggleStatus={async (id) => {
                const updated = await api.toggleTenantStatus(id);
                setTenants(updated);
              }}
              onSaveTenant={async (tenant) => {
                const updated = await api.saveTenant(tenant);
                setTenants(updated);
              }}
            />
          )}

          {activeTab === 'dids' && (
            <GlobalDidPool
              dids={dids}
              tenants={tenants}
              onAssignDid={async (didId, tenantId) => {
                const updated = await api.assignDid(didId, tenantId);
                setDids(updated);
                const fresh = await api.getTenants();
                setTenants(fresh);
              }}
              onAddDid={async (newDid) => {
                const updated = await api.addDid(newDid);
                setDids(updated);
              }}
            />
          )}

          {activeTab === 'diameter' && metrics && (
            <DiameterBilling metrics={metrics} />
          )}
        </main>
      </div>
    </div>
  );
};
