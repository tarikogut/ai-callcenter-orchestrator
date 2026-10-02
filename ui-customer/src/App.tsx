import React, { useState, useEffect } from 'react';
import {
  GitBranch,
  Bot,
  PhoneCall,
  FileSpreadsheet,
  Headphones,
  LogOut,
  Sun,
  Moon,
} from 'lucide-react';
import { Canvas as WorkflowCanvas } from './components/customer/WorkflowBuilder/Canvas';
import { AiPersonaSettings } from './components/customer/AiPersonaSettings';
import { ExtensionsManager } from './components/customer/ExtensionsManager';
import { WebRtcSoftphone } from './components/customer/WebRtcSoftphone';
import { CdrReports } from './components/customer/CdrReports';
import { CustomerLogin } from './components/CustomerLogin';
import { Modal } from './components/ui/Modal';
import { Button } from './components/ui/Button';
import { api } from './api/client';
import {
  Tenant,
  Extension,
  AiPersonaConfig,
  McpServerConfig,
  FaqItem,
  CdrRecord,
} from './types';

type CustomerTab = 'flow' | 'persona' | 'extensions' | 'softphone' | 'cdr';

export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(() => {
    return localStorage.getItem('customer_auth') === 'true';
  });

  const [activeTenantId, setActiveTenantId] = useState<string>(() => {
    return localStorage.getItem('customer_tenant_id') || 'eczane_hayat';
  });

  const [customerTab, setCustomerTab] = useState<CustomerTab>('flow');
  const [isDark, setIsDark] = useState<boolean>(() => localStorage.getItem('theme') !== 'light');
  const [isSoftphoneModalOpen, setIsSoftphoneModalOpen] = useState(false);

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [extensions, setExtensions] = useState<Extension[]>([]);
  const [persona, setPersona] = useState<AiPersonaConfig | null>(null);
  const [mcpServers, setMcpServers] = useState<McpServerConfig[]>([]);
  const [faqs, setFaqs] = useState<FaqItem[]>([]);
  const [cdrs, setCdrs] = useState<CdrRecord[]>([]);

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
    const fetchTenants = async () => {
      const list = await api.getTenants();
      setTenants(list);
    };
    fetchTenants();
  }, []);

  useEffect(() => {
    if (!isAuthenticated || !activeTenantId) return;

    const loadTenantData = async () => {
      const [exts, p, mcps, f, c] = await Promise.all([
        api.getExtensions(activeTenantId),
        api.getAiPersona(activeTenantId),
        api.getMcpServers(activeTenantId),
        api.getFaqs(activeTenantId),
        api.getCdrs(activeTenantId),
      ]);
      setExtensions(exts);
      setPersona(p);
      setMcpServers(mcps);
      setFaqs(f);
      setCdrs(c);
    };
    loadTenantData();
  }, [isAuthenticated, activeTenantId]);

  const handleLogin = (tenantId: string, _apiKey: string) => {
    setIsAuthenticated(true);
    setActiveTenantId(tenantId);
    localStorage.setItem('customer_auth', 'true');
    localStorage.setItem('customer_tenant_id', tenantId);
  };

  const handleLogout = () => {
    setIsAuthenticated(false);
    localStorage.removeItem('customer_auth');
    localStorage.removeItem('customer_tenant_id');
  };

  const activeTenant = tenants.find((t) => t.id === activeTenantId) || tenants[0];

  if (!isAuthenticated) {
    return <CustomerLogin tenants={tenants} onLogin={handleLogin} />;
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col font-sans">
      {/* Customer Header */}
      <header className="sticky top-0 z-40 w-full border-b border-border bg-card/90 backdrop-blur px-6 h-16 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center text-white shadow-md shadow-indigo-900/30">
            <Bot className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-base tracking-tight">{activeTenant?.name || 'Müşteri Portalı'}</span>
              <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                Tenant: {activeTenantId}
              </span>
            </div>
            <p className="text-[11px] text-muted-foreground font-medium">
              DID: {activeTenant?.assignedDids[0] || 'Atanmadı'} • Domain: {activeTenant?.domain}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {activeTenant && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl border border-border bg-background shadow-xs text-xs">
              <span className="text-muted-foreground font-medium">Kalan Bakiye:</span>
              <div className="flex items-center gap-1 text-emerald-500 font-mono text-xs font-semibold">
                <span className="w-2 h-2 rounded-full bg-emerald-500 inline-block animate-ping"></span>
                <span>{activeTenant.balance.toFixed(2)} {activeTenant.currency}</span>
              </div>
            </div>
          )}

          <Button
            variant={isSoftphoneModalOpen ? 'primary' : 'outline'}
            size="sm"
            onClick={() => setIsSoftphoneModalOpen(!isSoftphoneModalOpen)}
            className="relative gap-2 bg-indigo-600 hover:bg-indigo-700 text-white"
          >
            <Headphones className="w-4 h-4" />
            <span>WebRTC Softphone</span>
            <span className="absolute -top-1 -right-1 flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500"></span>
            </span>
          </Button>

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
            className="gap-1.5 text-xs text-destructive hover:bg-destructive/10 border-border"
          >
            <LogOut className="w-3.5 h-3.5" />
            <span>Çıkış Yap</span>
          </Button>
        </div>
      </header>

      {/* Main Customer Studio Layout */}
      <div className="flex flex-1 overflow-hidden">
        <aside className="w-64 border-r border-border bg-card/60 flex flex-col shrink-0 select-none">
          <div className="p-4 border-b border-border/50">
            <div className="text-xs font-bold uppercase tracking-wider text-indigo-400 px-2 flex items-center gap-1.5">
              <GitBranch className="w-3.5 h-3.5" />
              <span>İşlem Menüsü</span>
            </div>
          </div>
          <nav className="p-3 space-y-1.5 flex-1">
            <button
              onClick={() => setCustomerTab('flow')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'flow'
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <GitBranch className="w-4 h-4" />
              <span>Görsel Workflow Builder</span>
            </button>

            <button
              onClick={() => setCustomerTab('persona')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'persona'
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Bot className="w-4 h-4" />
              <span>AI Persona, BYOK & MCP</span>
            </button>

            <button
              onClick={() => setCustomerTab('extensions')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'extensions'
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <PhoneCall className="w-4 h-4" />
              <span>Dahili Hatlar (SIP / WebRTC)</span>
            </button>

            <button
              onClick={() => setCustomerTab('cdr')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'cdr'
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <FileSpreadsheet className="w-4 h-4" />
              <span>CDR & Ses Kayıtları</span>
            </button>
          </nav>
        </aside>

        <main className="flex-1 overflow-y-auto w-full">
          {customerTab === 'flow' && (
            <WorkflowCanvas tenantId={activeTenantId} />
          )}

          {customerTab === 'persona' && persona && (
            <div className="p-6 max-w-7xl mx-auto">
              <AiPersonaSettings
                tenantId={activeTenantId}
                persona={persona}
                mcpServers={mcpServers}
                faqs={faqs}
                onSavePersona={async (newP) => {
                  const saved = await api.saveAiPersona(activeTenantId, newP);
                  setPersona(saved);
                }}
                onSaveMcpServer={async (server) => {
                  const updated = await api.saveMcpServer(activeTenantId, server);
                  setMcpServers(updated);
                }}
                onSaveFaq={async (newFaq) => {
                  const updated = await api.saveFaq(activeTenantId, newFaq);
                  setFaqs(updated);
                }}
                onDeleteFaq={async (faqId) => {
                  const updated = await api.deleteFaq(activeTenantId, faqId);
                  setFaqs(updated);
                }}
              />
            </div>
          )}

          {customerTab === 'extensions' && activeTenant && (
            <div className="p-6 max-w-7xl mx-auto">
              <ExtensionsManager
                tenant={activeTenant}
                extensions={extensions}
                onSaveExtension={async (ext) => {
                  const updated = await api.saveExtension(activeTenantId, ext);
                  setExtensions(updated);
                }}
                onDeleteExtension={async (extId) => {
                  const updated = await api.deleteExtension(activeTenantId, extId);
                  setExtensions(updated);
                }}
              />
            </div>
          )}

          {customerTab === 'cdr' && (
            <div className="p-6 max-w-7xl mx-auto">
              <CdrReports cdrs={cdrs} tenantCurrency={activeTenant?.currency || 'TRY'} />
            </div>
          )}
        </main>
      </div>

      {/* WebRTC Softphone Modal */}
      {isSoftphoneModalOpen && activeTenant && (
        <Modal
          isOpen={isSoftphoneModalOpen}
          onClose={() => setIsSoftphoneModalOpen(false)}
          title="WebRTC Softphone & Human Takeover Console"
          maxWidth="lg"
        >
          <WebRtcSoftphone
            tenantName={activeTenant.name}
            activeExtension={extensions[0]}
          />
        </Modal>
      )}
    </div>
  );
}
