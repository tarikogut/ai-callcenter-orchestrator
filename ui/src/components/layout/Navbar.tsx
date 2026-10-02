import React from 'react';
import {
  Shield,
  Building2,
  Moon,
  Sun,
  Headphones,
  Radio,
  LogOut,
} from 'lucide-react';
import { Tenant, PortalMode } from '../../types';
import { Button } from '../ui/Button';

interface NavbarProps {
  mode: PortalMode;
  tenants: Tenant[];
  activeTenantId: string;
  isDark: boolean;
  onToggleTheme: () => void;
  onToggleSoftphone: () => void;
  isSoftphoneOpen: boolean;
  onLogout: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  mode,
  tenants,
  activeTenantId,
  isDark,
  onToggleTheme,
  onToggleSoftphone,
  isSoftphoneOpen,
  onLogout,
}) => {
  const currentTenant = tenants.find((t) => t.id === activeTenantId) || tenants[0];

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border bg-card/90 backdrop-blur supports-[backdrop-filter]:bg-card/75 px-4 lg:px-6 h-16 flex items-center justify-between">
      {/* Left: Brand & Portal Mode Badge */}
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
              {mode === 'admin' ? 'Super Admin Management Console' : 'Tenant Customer Cloud Studio'}
            </p>
          </div>
        </div>

        {/* Portal Indicator */}
        <div className="flex items-center gap-2">
          {mode === 'admin' ? (
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold bg-rose-500/10 text-rose-500 border border-rose-500/20">
              <Shield className="w-3.5 h-3.5" />
              SÜPER ADMIN PORTALI
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold bg-primary/10 text-primary border border-primary/20">
              <Building2 className="w-3.5 h-3.5" />
              MÜŞTERİ PORTALI: {currentTenant?.name}
            </span>
          )}
        </div>
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-3">
        {/* Tenant Balance in Customer portal */}
        {mode === 'customer' && currentTenant && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl border border-border bg-background shadow-xs text-xs">
            <span className="text-muted-foreground font-medium hidden md:inline">Bakiye:</span>
            <div className="flex items-center gap-1 text-emerald-500 font-mono text-xs font-semibold">
              <span className="w-2 h-2 rounded-full bg-emerald-500 inline-block animate-ping"></span>
              <span>{currentTenant.balance.toFixed(2)} {currentTenant.currency}</span>
            </div>
          </div>
        )}

        {/* WebRTC Softphone Toggle (Only in Customer Portal) */}
        {mode === 'customer' && (
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
        )}

        {/* Theme Toggle */}
        <button
          onClick={onToggleTheme}
          aria-label="Toggle theme"
          className="p-2 rounded-xl border border-border bg-secondary/50 hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
        >
          {isDark ? <Sun className="w-4 h-4 text-amber-400" /> : <Moon className="w-4 h-4 text-slate-700" />}
        </button>

        {/* Logout Button */}
        <Button
          variant="outline"
          size="sm"
          onClick={onLogout}
          className="gap-1.5 text-xs text-destructive hover:bg-destructive/10 border-border hover:border-destructive/30"
        >
          <LogOut className="w-3.5 h-3.5" />
          <span className="hidden sm:inline">Çıkış Yap</span>
        </Button>
      </div>
    </header>
  );
};
