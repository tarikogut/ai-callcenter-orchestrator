import React from 'react';
import {
  PhoneCall,
  Shield,
  Building2,
  Moon,
  Sun,
  Headphones,
  Radio,
  ChevronDown,
} from 'lucide-react';
import { Tenant, PortalMode } from '../../types';
import { Button } from '../ui/Button';

interface NavbarProps {
  mode: PortalMode;
  onModeChange: (mode: PortalMode) => void;
  tenants: Tenant[];
  activeTenantId: string;
  onTenantChange: (id: string) => void;
  isDark: boolean;
  onToggleTheme: () => void;
  onToggleSoftphone: () => void;
  isSoftphoneOpen: boolean;
}

export const Navbar: React.FC<NavbarProps> = ({
  mode,
  onModeChange,
  tenants,
  activeTenantId,
  onTenantChange,
  isDark,
  onToggleTheme,
  onToggleSoftphone,
  isSoftphoneOpen,
}) => {
  const currentTenant = tenants.find((t) => t.id === activeTenantId) || tenants[0];

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border bg-card/90 backdrop-blur supports-[backdrop-filter]:bg-card/75 px-4 lg:px-6 h-16 flex items-center justify-between">
      {/* Left: Brand & Portal Mode Switcher */}
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-2.5">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-primary to-indigo-500 flex items-center justify-center text-white shadow-md shadow-primary/20">
            <Radio className="w-5 h-5 animate-pulse" />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="font-bold text-base tracking-tight text-foreground">
                ai-callcenter
              </span>
              <span className="text-xs px-1.5 py-0.5 rounded bg-primary/10 text-primary font-mono font-medium">
                v2.5
              </span>
            </div>
            <p className="text-[11px] text-muted-foreground font-medium hidden sm:block">
              Carrier-Grade Orchestrator & Visual Studio
            </p>
          </div>
        </div>

        {/* Portal Pill Selector */}
        <div className="flex items-center p-1 bg-secondary/80 rounded-xl border border-border text-xs font-medium">
          <button
            onClick={() => onModeChange('admin')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
              mode === 'admin'
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Shield className="w-3.5 h-3.5" />
            <span>Admin Portal</span>
          </button>
          <button
            onClick={() => onModeChange('customer')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg transition-all ${
              mode === 'customer'
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <Building2 className="w-3.5 h-3.5" />
            <span>Customer Portal</span>
          </button>
        </div>
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-3">
        {/* Tenant Selector (if in customer portal) */}
        {mode === 'customer' && (
          <div className="relative flex items-center">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl border border-border bg-background shadow-xs text-xs">
              <span className="text-muted-foreground font-medium hidden md:inline">Tenant:</span>
              <select
                value={activeTenantId}
                onChange={(e) => onTenantChange(e.target.value)}
                className="bg-transparent font-semibold text-foreground focus:outline-none cursor-pointer pr-4"
              >
                {tenants.map((t) => (
                  <option key={t.id} value={t.id} className="bg-card text-foreground">
                    {t.name} ({t.assignedDids[0] || 'No DID'})
                  </option>
                ))}
              </select>
              <div className="hidden lg:flex items-center gap-1 pl-2 border-l border-border text-emerald-500 font-mono text-[11px]">
                <span className="w-2 h-2 rounded-full bg-emerald-500 inline-block animate-ping"></span>
                <span>{currentTenant?.balance.toFixed(2)} {currentTenant?.currency}</span>
              </div>
            </div>
          </div>
        )}

        {/* WebRTC Softphone Toggle */}
        <Button
          variant={isSoftphoneOpen ? 'primary' : 'outline'}
          size="sm"
          onClick={onToggleSoftphone}
          className="relative gap-2"
        >
          <Headphones className="w-4 h-4" />
          <span className="hidden sm:inline">Softphone</span>
          <span className="absolute -top-1 -right-1 flex h-2.5 w-2.5">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500"></span>
          </span>
        </Button>

        {/* Theme Toggle */}
        <button
          onClick={onToggleTheme}
          aria-label="Toggle theme"
          className="p-2 rounded-xl border border-border bg-secondary/50 hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
        >
          {isDark ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4 text-slate-700" />}
        </button>
      </div>
    </header>
  );
};
