import React, { useState } from 'react';
import {
  Users,
  Plus,
  Lock,
  Unlock,
  Sliders,
  PhoneCall,
  Search,
  CheckCircle2,
  AlertTriangle,
  Building,
} from 'lucide-react';
import { Tenant } from '../../types';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/Card';
import { Modal } from '../ui/Modal';

interface TenantManagementProps {
  tenants: Tenant[];
  onToggleStatus: (tenantId: string) => void;
  onSaveTenant: (tenant: Tenant) => void;
}

export const TenantManagement: React.FC<TenantManagementProps> = ({
  tenants,
  onToggleStatus,
  onSaveTenant,
}) => {
  const [search, setSearch] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isQuotaModalOpen, setIsQuotaModalOpen] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);

  // Form states for new tenant
  const [newId, setNewId] = useState('');
  const [newName, setNewName] = useState('');
  const [newDomain, setNewDomain] = useState('');
  const [newIndustry, setNewIndustry] = useState<Tenant['industry']>('pharmacy');
  const [newEmail, setNewEmail] = useState('');
  const [newMaxCalls, setNewMaxCalls] = useState(10);
  const [newSpendLimit, setNewSpendLimit] = useState(5000);
  const [newInitialBalance, setNewInitialBalance] = useState(500);

  // Form states for quota editing
  const [editMaxCalls, setEditMaxCalls] = useState(10);
  const [editSpendLimit, setEditSpendLimit] = useState(5000);

  const filteredTenants = tenants.filter(
    (t) =>
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.id.toLowerCase().includes(search.toLowerCase()) ||
      t.domain.toLowerCase().includes(search.toLowerCase())
  );

  const handleCreateTenant = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newId || !newName) return;

    const tenant: Tenant = {
      id: newId.toLowerCase().replace(/[^a-z0-9_]/g, '_'),
      name: newName,
      domain: newDomain || `${newId}.icell.cloud`,
      industry: newIndustry,
      status: 'active',
      balance: Number(newInitialBalance),
      currency: 'TRY',
      maxConcurrentCalls: Number(newMaxCalls),
      monthlySpendLimit: Number(newSpendLimit),
      currentConcurrentCalls: 0,
      assignedDids: [],
      createdAt: new Date().toISOString().split('T')[0],
      contactEmail: newEmail || `admin@${newDomain}`,
    };

    onSaveTenant(tenant);
    setIsCreateModalOpen(false);
    // Reset
    setNewId('');
    setNewName('');
    setNewDomain('');
    setNewEmail('');
  };

  const openQuotaModal = (tenant: Tenant) => {
    setSelectedTenant(tenant);
    setEditMaxCalls(tenant.maxConcurrentCalls);
    setEditSpendLimit(tenant.monthlySpendLimit);
    setIsQuotaModalOpen(true);
  };

  const handleUpdateQuota = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedTenant) return;
    const updated: Tenant = {
      ...selectedTenant,
      maxConcurrentCalls: Number(editMaxCalls),
      monthlySpendLimit: Number(editSpendLimit),
    };
    onSaveTenant(updated);
    setIsQuotaModalOpen(false);
  };

  return (
    <div className="space-y-6">
      {/* Top Header & Actions */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Users className="w-6 h-6 text-primary" />
            Tenant Management
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Configure multi-tenant CPaaS accounts, real-time call limits, and prepaid status.
          </p>
        </div>

        <Button onClick={() => setIsCreateModalOpen(true)} className="gap-2 shadow-sm">
          <Plus className="w-4 h-4" />
          <span>New Tenant</span>
        </Button>
      </div>

      {/* Stats Quick Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Total Tenants</p>
              <p className="text-2xl font-bold text-foreground mt-0.5">{tenants.length}</p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary">
              <Building className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Active Accounts</p>
              <p className="text-2xl font-bold text-emerald-600 dark:text-emerald-400 mt-0.5">
                {tenants.filter((t) => t.status === 'active').length}
              </p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 flex items-center justify-center text-emerald-500">
              <CheckCircle2 className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Active Live Channels</p>
              <p className="text-2xl font-bold text-blue-600 dark:text-blue-400 mt-0.5">
                {tenants.reduce((acc, t) => acc + t.currentConcurrentCalls, 0)}
              </p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center text-blue-500">
              <PhoneCall className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Frozen Accounts</p>
              <p className="text-2xl font-bold text-rose-600 dark:text-rose-400 mt-0.5">
                {tenants.filter((t) => t.status === 'frozen').length}
              </p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-rose-500/10 flex items-center justify-center text-rose-500">
              <AlertTriangle className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filter and Table */}
      <Card>
        <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <CardTitle>Tenants Directory</CardTitle>
            <CardDescription>View, activate/freeze, or modify call capacity quotas.</CardDescription>
          </div>
          <div className="relative w-full sm:w-64">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search tenant name or ID..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 text-xs rounded-lg border border-border bg-background focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </CardHeader>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-muted/40 text-muted-foreground uppercase font-semibold border-b border-border">
              <tr>
                <th className="px-5 py-3">Tenant Name & ID</th>
                <th className="px-5 py-3">Industry</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Prepaid Balance</th>
                <th className="px-5 py-3">Concurrent Calls</th>
                <th className="px-5 py-3">Assigned DIDs</th>
                <th className="px-5 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {filteredTenants.map((t) => (
                <tr key={t.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-5 py-4">
                    <div className="font-semibold text-foreground text-sm">{t.name}</div>
                    <div className="text-[11px] font-mono text-muted-foreground">ID: {t.id}</div>
                    <div className="text-[11px] text-muted-foreground">{t.domain}</div>
                  </td>
                  <td className="px-5 py-4">
                    <span className="capitalize px-2 py-0.5 rounded bg-secondary font-medium">
                      {t.industry}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    {t.status === 'active' ? (
                      <Badge variant="success">Active</Badge>
                    ) : (
                      <Badge variant="destructive">Frozen</Badge>
                    )}
                  </td>
                  <td className="px-5 py-4 font-mono font-medium">
                    <span className={t.balance < 50 ? 'text-rose-500 font-bold' : 'text-foreground'}>
                      {t.balance.toFixed(2)} {t.currency}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    <div className="flex items-center gap-2">
                      <div className="w-20 bg-secondary h-2 rounded-full overflow-hidden">
                        <div
                          className="bg-primary h-full transition-all"
                          style={{
                            width: `${Math.min(100, (t.currentConcurrentCalls / t.maxConcurrentCalls) * 100)}%`,
                          }}
                        />
                      </div>
                      <span className="font-mono text-[11px] text-muted-foreground">
                        {t.currentConcurrentCalls}/{t.maxConcurrentCalls}
                      </span>
                    </div>
                  </td>
                  <td className="px-5 py-4">
                    {t.assignedDids.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {t.assignedDids.map((did) => (
                          <span key={did} className="px-1.5 py-0.5 bg-primary/10 text-primary font-mono text-[11px] rounded">
                            {did}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="text-muted-foreground italic">None</span>
                    )}
                  </td>
                  <td className="px-5 py-4 text-right space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openQuotaModal(t)}
                      className="h-7 text-xs"
                    >
                      <Sliders className="w-3 h-3" />
                      Quota
                    </Button>
                    <Button
                      variant={t.status === 'active' ? 'destructive' : 'success'}
                      size="sm"
                      onClick={() => onToggleStatus(t.id)}
                      className="h-7 text-xs"
                    >
                      {t.status === 'active' ? (
                        <>
                          <Lock className="w-3 h-3" />
                          Freeze
                        </>
                      ) : (
                        <>
                          <Unlock className="w-3 h-3" />
                          Activate
                        </>
                      )}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Create Tenant Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Multi-Tenant Account"
        description="Provision an isolated tenant environment with Diameter Ro billing credentials."
      >
        <form onSubmit={handleCreateTenant} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground">Tenant ID (Identifier)</label>
              <input
                type="text"
                required
                placeholder="e.g. klinik_umut"
                value={newId}
                onChange={(e) => setNewId(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground">Display Name</label>
              <input
                type="text"
                required
                placeholder="e.g. Umut Polikliniği"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground">Industry / Sector</label>
              <select
                value={newIndustry}
                onChange={(e) => setNewIndustry(e.target.value as any)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
              >
                <option value="pharmacy">Pharmacy / Eczane</option>
                <option value="logistics">Logistics / Kargo</option>
                <option value="clinic">Clinic / Sağlık</option>
                <option value="ecommerce">E-Commerce / Perakende</option>
                <option value="other">Other / Genel</option>
              </select>
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground">Domain / Host</label>
              <input
                type="text"
                placeholder="e.g. umutklinik.com"
                value={newDomain}
                onChange={(e) => setNewDomain(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="text-xs font-semibold text-foreground">Max Channels</label>
              <input
                type="number"
                min="1"
                max="100"
                value={newMaxCalls}
                onChange={(e) => setNewMaxCalls(Number(e.target.value))}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground">Spend Limit (TRY)</label>
              <input
                type="number"
                step="100"
                value={newSpendLimit}
                onChange={(e) => setNewSpendLimit(Number(e.target.value))}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground">Initial Deposit</label>
              <input
                type="number"
                step="50"
                value={newInitialBalance}
                onChange={(e) => setNewInitialBalance(Number(e.target.value))}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-4 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsCreateModalOpen(false)}>
              Cancel
            </Button>
            <Button type="submit">Create Tenant</Button>
          </div>
        </form>
      </Modal>

      {/* Edit Quota Modal */}
      <Modal
        isOpen={isQuotaModalOpen}
        onClose={() => setIsQuotaModalOpen(false)}
        title={`Set Global Quotas: ${selectedTenant?.name}`}
        description="Limit concurrent call slots and maximum monthly credit spending."
      >
        <form onSubmit={handleUpdateQuota} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-foreground">
              Maximum Concurrent Calls (Kamailio / FreeSWITCH Channels)
            </label>
            <input
              type="number"
              min="1"
              max="200"
              value={editMaxCalls}
              onChange={(e) => setEditMaxCalls(Number(e.target.value))}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
            <p className="text-[11px] text-muted-foreground mt-1">
              Calls exceeding this threshold will receive SIP 486 Busy Here or queue waiting tone.
            </p>
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">
              Monthly Credit Spending Cap ({selectedTenant?.currency})
            </label>
            <input
              type="number"
              step="500"
              value={editSpendLimit}
              onChange={(e) => setEditSpendLimit(Number(e.target.value))}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
            <p className="text-[11px] text-muted-foreground mt-1">
              When reached, Diameter Ro CCR returns DIAMETER_CREDIT_LIMIT_REACHED.
            </p>
          </div>

          <div className="flex justify-end gap-2 pt-4 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsQuotaModalOpen(false)}>
              Cancel
            </Button>
            <Button type="submit">Save Quotas</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
