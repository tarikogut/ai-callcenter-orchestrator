import React, { useState } from 'react';
import {
  Hash,
  Plus,
  ArrowRightLeft,
  Search,
  CheckCircle,
  XCircle,
  Radio,
  Building,
} from 'lucide-react';
import { DIDNumber, Tenant } from '../../types';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/Card';
import { Modal } from '../ui/Modal';

interface GlobalDidPoolProps {
  dids: DIDNumber[];
  tenants: Tenant[];
  onAssignDid: (didId: string, tenantId: string | null) => void;
  onAddDid: (did: DIDNumber) => void;
}

export const GlobalDidPool: React.FC<GlobalDidPoolProps> = ({
  dids,
  tenants,
  onAssignDid,
  onAddDid,
}) => {
  const [search, setSearch] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'assigned' | 'available'>('all');
  const [isAssignModalOpen, setIsAssignModalOpen] = useState(false);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [selectedDid, setSelectedDid] = useState<DIDNumber | null>(null);
  const [selectedTenantId, setSelectedTenantId] = useState<string>('');

  // Add DID form states
  const [newNumber, setNewNumber] = useState('');
  const [newCarrier, setNewCarrier] = useState('Turkcell Superonline SIP Trunk');
  const [newCost, setNewCost] = useState(45.0);

  const filteredDids = dids.filter((d) => {
    const matchesSearch =
      d.number.toLowerCase().includes(search.toLowerCase()) ||
      d.carrier.toLowerCase().includes(search.toLowerCase()) ||
      (d.tenantName && d.tenantName.toLowerCase().includes(search.toLowerCase()));

    if (filterStatus === 'assigned') return matchesSearch && d.status === 'assigned';
    if (filterStatus === 'available') return matchesSearch && d.status === 'available';
    return matchesSearch;
  });

  const openAssignModal = (did: DIDNumber) => {
    setSelectedDid(did);
    setSelectedTenantId(did.tenantId || '');
    setIsAssignModalOpen(true);
  };

  const handleAssignSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDid) return;
    onAssignDid(selectedDid.id, selectedTenantId || null);
    setIsAssignModalOpen(false);
  };

  const handleAddDidSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newNumber) return;

    const did: DIDNumber = {
      id: `did_${Date.now()}`,
      number: newNumber.trim(),
      countryCode: 'TR',
      carrier: newCarrier,
      tenantId: null,
      tenantName: null,
      assignedAt: null,
      status: 'available',
      monthlyCost: Number(newCost),
    };

    onAddDid(did);
    setIsAddModalOpen(false);
    setNewNumber('');
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Hash className="w-6 h-6 text-primary" />
            Global Telecom DID Pool
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Manage carrier phone numbers, SIP trunks, and map inbound numbers to customer tenants.
          </p>
        </div>

        <Button onClick={() => setIsAddModalOpen(true)} className="gap-2 shadow-sm">
          <Plus className="w-4 h-4" />
          <span>Add Number to Pool</span>
        </Button>
      </div>

      {/* Metrics Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Total DID Numbers</p>
              <p className="text-2xl font-bold text-foreground mt-0.5">{dids.length}</p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary">
              <Hash className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Assigned to Tenants</p>
              <p className="text-2xl font-bold text-emerald-600 dark:text-emerald-400 mt-0.5">
                {dids.filter((d) => d.status === 'assigned').length}
              </p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-emerald-500/10 flex items-center justify-center text-emerald-500">
              <CheckCircle className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <p className="text-xs font-medium text-muted-foreground">Available in Pool</p>
              <p className="text-2xl font-bold text-blue-600 dark:text-blue-400 mt-0.5">
                {dids.filter((d) => d.status === 'available').length}
              </p>
            </div>
            <div className="w-10 h-10 rounded-xl bg-blue-500/10 flex items-center justify-center text-blue-500">
              <Radio className="w-5 h-5" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* DIDs Table */}
      <Card>
        <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <CardTitle>Carrier DID Inventory</CardTitle>
            <div className="flex rounded-lg bg-secondary p-0.5 text-xs">
              <button
                onClick={() => setFilterStatus('all')}
                className={`px-2.5 py-1 rounded-md transition-all ${
                  filterStatus === 'all' ? 'bg-card font-medium text-foreground shadow-xs' : 'text-muted-foreground'
                }`}
              >
                All
              </button>
              <button
                onClick={() => setFilterStatus('assigned')}
                className={`px-2.5 py-1 rounded-md transition-all ${
                  filterStatus === 'assigned' ? 'bg-card font-medium text-foreground shadow-xs' : 'text-muted-foreground'
                }`}
              >
                Assigned
              </button>
              <button
                onClick={() => setFilterStatus('available')}
                className={`px-2.5 py-1 rounded-md transition-all ${
                  filterStatus === 'available' ? 'bg-card font-medium text-foreground shadow-xs' : 'text-muted-foreground'
                }`}
              >
                Available
              </button>
            </div>
          </div>

          <div className="relative w-full sm:w-64">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search phone number or carrier..."
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
                <th className="px-5 py-3">Phone Number</th>
                <th className="px-5 py-3">Carrier / SIP Trunk</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Assigned Tenant</th>
                <th className="px-5 py-3">Monthly Cost</th>
                <th className="px-5 py-3 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {filteredDids.map((d) => (
                <tr key={d.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-5 py-3.5">
                    <span className="font-mono text-sm font-semibold text-foreground">
                      {d.number}
                    </span>
                  </td>
                  <td className="px-5 py-3.5 text-muted-foreground">
                    {d.carrier}
                  </td>
                  <td className="px-5 py-3.5">
                    {d.status === 'assigned' ? (
                      <Badge variant="success">Assigned</Badge>
                    ) : (
                      <Badge variant="secondary">Available</Badge>
                    )}
                  </td>
                  <td className="px-5 py-3.5">
                    {d.tenantName ? (
                      <div className="flex items-center gap-1.5 font-medium text-foreground">
                        <Building className="w-3.5 h-3.5 text-primary" />
                        <span>{d.tenantName}</span>
                      </div>
                    ) : (
                      <span className="text-muted-foreground italic">Unassigned (Pool)</span>
                    )}
                  </td>
                  <td className="px-5 py-3.5 font-mono text-muted-foreground">
                    {d.monthlyCost.toFixed(2)} TRY/mo
                  </td>
                  <td className="px-5 py-3.5 text-right">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openAssignModal(d)}
                      className="h-7 text-xs gap-1.5"
                    >
                      <ArrowRightLeft className="w-3 h-3" />
                      {d.status === 'assigned' ? 'Reassign' : 'Assign'}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Assign DID Modal */}
      <Modal
        isOpen={isAssignModalOpen}
        onClose={() => setIsAssignModalOpen(false)}
        title={`Assign DID: ${selectedDid?.number}`}
        description="Route incoming Kamailio SIP traffic from this number to a specific tenant workflow."
      >
        <form onSubmit={handleAssignSubmit} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-foreground">Target Tenant</label>
            <select
              value={selectedTenantId}
              onChange={(e) => setSelectedTenantId(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
            >
              <option value="">-- Release to Pool (Unassigned) --</option>
              {tenants.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} ({t.id})
                </option>
              ))}
            </select>
          </div>

          <div className="p-3 bg-muted/40 rounded-lg text-[11px] text-muted-foreground space-y-1">
            <p><strong>Carrier:</strong> {selectedDid?.carrier}</p>
            <p><strong>Routing Target:</strong> Inbound call trigger node matching <code>$did == {selectedDid?.number}</code></p>
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsAssignModalOpen(false)}>
              Cancel
            </Button>
            <Button type="submit">Update Assignment</Button>
          </div>
        </form>
      </Modal>

      {/* Add New DID Modal */}
      <Modal
        isOpen={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
        title="Add Carrier DID Number"
        description="Register a new external DID into the orchestrator gateway pool."
      >
        <form onSubmit={handleAddDidSubmit} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-foreground">Phone Number (E.164 Format)</label>
            <input
              type="text"
              required
              placeholder="+90 850 123 4567"
              value={newNumber}
              onChange={(e) => setNewNumber(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">Carrier / SIP Interconnect</label>
            <select
              value={newCarrier}
              onChange={(e) => setNewCarrier(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            >
              <option value="Turkcell Superonline SIP Trunk">Turkcell Superonline SIP Trunk</option>
              <option value="Türk Telekom PSTN Interconnect">Türk Telekom PSTN Interconnect</option>
              <option value="Vodafone Carrier SIP">Vodafone Carrier SIP</option>
              <option value="Netgsm SIP Gateway">Netgsm SIP Gateway</option>
              <option value="Verimor Telekom Cloud Trunk">Verimor Telekom Cloud Trunk</option>
            </select>
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">Monthly Carrier Fee (TRY)</label>
            <input
              type="number"
              step="5"
              value={newCost}
              onChange={(e) => setNewCost(Number(e.target.value))}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsAddModalOpen(false)}>
              Cancel
            </Button>
            <Button type="submit">Save DID to Pool</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
