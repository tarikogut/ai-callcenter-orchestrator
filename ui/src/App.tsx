import React, { useState, useEffect } from 'react';
import { Navbar } from './components/layout/Navbar';
import { Sidebar, AdminTab, CustomerTab } from './components/layout/Sidebar';
import { TenantManagement } from './components/admin/TenantManagement';
import { GlobalDidPool } from './components/admin/GlobalDidPool';
import { DiameterBilling } from './components/admin/DiameterBilling';
import { Canvas as WorkflowCanvas } from './components/customer/WorkflowBuilder/Canvas';
import { AiPersonaSettings } from './components/customer/AiPersonaSettings';
import { ExtensionsManager } from './components/customer/ExtensionsManager';
import { WebRtcSoftphone } from './components/customer/WebRtcSoftphone';
import { CdrReports } from './components/customer/CdrReports';
import { Modal } from './components/ui/Modal';
import { LoginScreen } from './components/auth/LoginScreen';
import { api } from './api/client';
import {
  Tenant,
  DIDNumber,
  DiameterMetric,
  Extension,
  AiPersonaConfig,
  McpServerConfig,
  FaqItem,
  CdrRecord,
  PortalMode,
} from './types';

export const App: React.FC = () => {
  // Session authentication state
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(() => {
    return localStorage.getItem('cpaas_auth') === 'true';
  });

  const [mode, setMode] = useState<PortalMode>(() => {
    const savedRole = localStorage.getItem('cpaas_role') as PortalMode;
    if (savedRole === 'admin' || savedRole === 'customer') return savedRole;
    return window.location.pathname.startsWith('/admin') ? 'admin' : 'customer';
  });

  const [activeTenantId, setActiveTenantId] = useState<string>(() => {
    return localStorage.getItem('cpaas_active_tenant') || 'eczane_hayat';
  });

  const [adminTab, setAdminTab] = useState<AdminTab>('tenants');
  const [customerTab, setCustomerTab] = useState<CustomerTab>('flow');

  // Theme state
  const [isDark, setIsDark] = useState<boolean>(() => {
    return localStorage.getItem('theme') !== 'light';
  });

  // Softphone quick drawer modal
  const [isSoftphoneModalOpen, setIsSoftphoneModalOpen] = useState(false);

  // Core Data States
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [dids, setDids] = useState<DIDNumber[]>([]);
  const [metrics, setMetrics] = useState<DiameterMetric | null>(null);
  const [extensions, setExtensions] = useState<Extension[]>([]);
  const [persona, setPersona] = useState<AiPersonaConfig | null>(null);
  const [mcpServers, setMcpServers] = useState<McpServerConfig[]>([]);
  const [faqs, setFaqs] = useState<FaqItem[]>([]);
  const [cdrs, setCdrs] = useState<CdrRecord[]>([]);

  // Auth Handlers
  const handleAdminLogin = (password: string): boolean => {
    if (password === 'admin123' || password === '') {
      setIsAuthenticated(true);
      setMode('admin');
      localStorage.setItem('cpaas_auth', 'true');
      localStorage.setItem('cpaas_role', 'admin');
      window.history.pushState(null, '', '/admin');
      return true;
    }
    return false;
  };

  const handleCustomerLogin = (tenantId: string, _authKey: string): boolean => {
    setIsAuthenticated(true);
    setMode('customer');
    setActiveTenantId(tenantId);
    localStorage.setItem('cpaas_auth', 'true');
    localStorage.setItem('cpaas_role', 'customer');
    localStorage.setItem('cpaas_active_tenant', tenantId);
    window.history.pushState(null, '', '/customer');
    return true;
  };

  const handleLogout = () => {
    setIsAuthenticated(false);
    localStorage.removeItem('cpaas_auth');
    localStorage.removeItem('cpaas_role');
    window.history.pushState(null, '', '/');
  };

  // Sync theme to HTML class
  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    }
  }, [isDark]);

  // Load initial dataset
  useEffect(() => {
    const loadData = async () => {
      const t = await api.getTenants();
      setTenants(t);
      if (t.length > 0) {
        setActiveTenantId((prev) => prev || t[0].id);
      }

      const d = await api.getDids();
      setDids(d);

      const m = await api.getDiameterMetrics();
      setMetrics(m);
    };
    loadData();
  }, []);

  // Reload tenant-specific data when activeTenantId changes
  useEffect(() => {
    if (!activeTenantId) return;

    const loadTenantData = async () => {
      const exts = await api.getExtensions(activeTenantId);
      setExtensions(exts);

      const p = await api.getAiPersona(activeTenantId);
      setPersona(p);

      const mcps = await api.getMcpServers(activeTenantId);
      setMcpServers(mcps);

      const f = await api.getFaqs(activeTenantId);
      setFaqs(f);

      const c = await api.getCdrs(activeTenantId);
      setCdrs(c);
    };

    loadTenantData();
  }, [activeTenantId]);

  // Listen for browser popstate
  useEffect(() => {
    const handlePopState = () => {
      const isAdm = window.location.pathname.startsWith('/admin');
      setMode(isAdm ? 'admin' : 'customer');
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const activeTenant = tenants.find((t) => t.id === activeTenantId) || tenants[0];

  // Render Login Screen if not authenticated
  if (!isAuthenticated) {
    return (
      <LoginScreen
        tenants={tenants}
        onAdminLogin={handleAdminLogin}
        onCustomerLogin={handleCustomerLogin}
      />
    );
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col font-sans">
      {/* Top Navigation */}
      <Navbar
        mode={mode}
        tenants={tenants}
        activeTenantId={activeTenantId}
        isDark={isDark}
        onToggleTheme={() => setIsDark(!isDark)}
        onToggleSoftphone={() => setIsSoftphoneModalOpen(!isSoftphoneModalOpen)}
        isSoftphoneOpen={isSoftphoneModalOpen}
        onLogout={handleLogout}
      />

      {/* Main Workspace with Sidebar & Content */}
      <div className="flex flex-1 overflow-hidden">
        <Sidebar
          mode={mode}
          adminTab={adminTab}
          onAdminTabChange={setAdminTab}
          customerTab={customerTab}
          onCustomerTabChange={setCustomerTab}
        />

        <main className="flex-1 overflow-y-auto">
          {mode === 'admin' ? (
            <div className="p-6 max-w-7xl mx-auto">
              {adminTab === 'tenants' && (
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

              {adminTab === 'dids' && (
                <GlobalDidPool
                  dids={dids}
                  tenants={tenants}
                  onAssignDid={async (didId, tenantId) => {
                    const updated = await api.assignDid(didId, tenantId);
                    setDids(updated);
                    const freshTenants = await api.getTenants();
                    setTenants(freshTenants);
                  }}
                  onAddDid={async (newDid) => {
                    const updated = await api.addDid(newDid);
                    setDids(updated);
                  }}
                />
              )}

              {adminTab === 'diameter' && metrics && (
                <DiameterBilling metrics={metrics} />
              )}
            </div>
          ) : (
            <div>
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

              {customerTab === 'softphone' && activeTenant && (
                <div className="p-6 max-w-7xl mx-auto">
                  <WebRtcSoftphone
                    activeExtension={extensions[0]}
                    tenantName={activeTenant.name}
                  />
                </div>
              )}

              {customerTab === 'cdr' && (
                <div className="p-6 max-w-7xl mx-auto">
                  <CdrReports
                    cdrs={cdrs}
                    tenantCurrency={activeTenant?.currency || 'TRY'}
                  />
                </div>
              )}
            </div>
          )}
        </main>
      </div>

      {/* Floating Quick Softphone Modal */}
      <Modal
        isOpen={isSoftphoneModalOpen}
        onClose={() => setIsSoftphoneModalOpen(false)}
        title="WebRTC Agent Softphone Quick Launcher"
        description="Active Extension: 101 • Connected to Kamailio WSS / FreeSWITCH mod_audio_fork"
        maxWidth="2xl"
      >
        <div className="pt-2">
          <WebRtcSoftphone
            activeExtension={extensions[0]}
            tenantName={activeTenant?.name || 'Customer Portal'}
          />
        </div>
      </Modal>
    </div>
  );
};

export default App;
