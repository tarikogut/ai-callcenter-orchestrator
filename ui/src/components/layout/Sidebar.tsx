import React from 'react';
import {
  Users,
  Hash,
  Activity,
  GitBranch,
  Bot,
  PhoneCall,
  Headphones,
  FileSpreadsheet,
  Server,
  Zap,
} from 'lucide-react';
import { PortalMode } from '../../types';

export type AdminTab = 'tenants' | 'dids' | 'diameter';
export type CustomerTab = 'flow' | 'persona' | 'extensions' | 'softphone' | 'cdr';

interface SidebarProps {
  mode: PortalMode;
  adminTab: AdminTab;
  onAdminTabChange: (tab: AdminTab) => void;
  customerTab: CustomerTab;
  onCustomerTabChange: (tab: CustomerTab) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  mode,
  adminTab,
  onAdminTabChange,
  customerTab,
  onCustomerTabChange,
}) => {
  return (
    <aside className="w-64 border-r border-border bg-card/60 flex flex-col shrink-0 select-none">
      <div className="p-4 border-b border-border/50">
        <div className="text-xs font-semibold uppercase tracking-wider text-muted-foreground px-2">
          {mode === 'admin' ? 'Admin Central' : 'Workspace'}
        </div>
      </div>

      <nav className="p-3 space-y-1 flex-1">
        {mode === 'admin' ? (
          <>
            <button
              onClick={() => onAdminTabChange('tenants')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                adminTab === 'tenants'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Users className="w-4 h-4" />
              <span>Tenant Management</span>
            </button>

            <button
              onClick={() => onAdminTabChange('dids')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                adminTab === 'dids'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Hash className="w-4 h-4" />
              <span>Global DID Pool</span>
            </button>

            <button
              onClick={() => onAdminTabChange('diameter')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                adminTab === 'diameter'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Activity className="w-4 h-4" />
              <span>Diameter Ro Billing</span>
            </button>
          </>
        ) : (
          <>
            <button
              onClick={() => onCustomerTabChange('flow')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'flow'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <GitBranch className="w-4 h-4" />
              <span>Visual Workflow Builder</span>
            </button>

            <button
              onClick={() => onCustomerTabChange('persona')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'persona'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Bot className="w-4 h-4" />
              <span>AI Persona & Settings</span>
            </button>

            <button
              onClick={() => onCustomerTabChange('extensions')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'extensions'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <PhoneCall className="w-4 h-4" />
              <span>Extensions & DIDs</span>
            </button>

            <button
              onClick={() => onCustomerTabChange('softphone')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'softphone'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <Headphones className="w-4 h-4" />
              <span>Softphone & Live Monitor</span>
            </button>

            <button
              onClick={() => onCustomerTabChange('cdr')}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all ${
                customerTab === 'cdr'
                  ? 'bg-primary text-primary-foreground shadow-sm'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
              }`}
            >
              <FileSpreadsheet className="w-4 h-4" />
              <span>CDR & Reports</span>
            </button>
          </>
        )}
      </nav>

      {/* Footer System Status Badge */}
      <div className="p-4 border-t border-border/50 text-xs">
        <div className="p-3 rounded-xl bg-secondary/60 border border-border flex flex-col gap-1.5">
          <div className="flex items-center justify-between">
            <span className="font-semibold text-foreground flex items-center gap-1.5">
              <Server className="w-3.5 h-3.5 text-primary" />
              Orchestrator
            </span>
            <span className="inline-flex items-center gap-1 text-[10px] text-emerald-500 font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
              Live (Go)
            </span>
          </div>
          <div className="text-[11px] text-muted-foreground flex justify-between">
            <span>Kamailio + FreeSWITCH</span>
            <span className="font-mono text-emerald-500">200 OK</span>
          </div>
          <div className="text-[11px] text-muted-foreground flex justify-between">
            <span>Diameter Ro (CCR/CCA)</span>
            <span className="font-mono text-primary">Prepaid</span>
          </div>
        </div>
      </div>
    </aside>
  );
};
