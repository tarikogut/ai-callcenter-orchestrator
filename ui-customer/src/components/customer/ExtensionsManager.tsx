import React, { useState } from 'react';
import {
  PhoneCall,
  Plus,
  Trash2,
  Edit,
  Globe,
  UserCheck,
  Hash,
} from 'lucide-react';
import { Extension, Tenant } from '../../types';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/Card';
import { Modal } from '../ui/Modal';

interface ExtensionsManagerProps {
  tenant: Tenant;
  extensions: Extension[];
  onSaveExtension: (ext: Extension) => void;
  onDeleteExtension: (extId: string) => void;
}

export const ExtensionsManager: React.FC<ExtensionsManagerProps> = ({
  tenant,
  extensions,
  onSaveExtension,
  onDeleteExtension,
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingExt, setEditingExt] = useState<Extension | null>(null);

  // Form states
  const [extNumber, setExtNumber] = useState('101');
  const [extName, setExtName] = useState('');
  const [extAgent, setExtAgent] = useState('');
  const [extPassword, setExtPassword] = useState('Secret123!');
  const [extWebRtc, setExtWebRtc] = useState(true);

  const openAddModal = () => {
    setEditingExt(null);
    setExtNumber(`${101 + extensions.length}`);
    setExtName('');
    setExtAgent('');
    setExtPassword('P@ssword' + (101 + extensions.length));
    setExtWebRtc(true);
    setIsModalOpen(true);
  };

  const openEditModal = (ext: Extension) => {
    setEditingExt(ext);
    setExtNumber(ext.extensionNumber);
    setExtName(ext.name);
    setExtAgent(ext.assignedAgent || '');
    setExtPassword(ext.sipSecret);
    setExtWebRtc(ext.webrtcEnabled);
    setIsModalOpen(true);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!extNumber || !extName) return;

    const ext: Extension = {
      id: editingExt ? editingExt.id : `ext_${Date.now()}`,
      extensionNumber: extNumber,
      name: extName,
      tenantId: tenant.id,
      sipUsername: `${extNumber}_${tenant.id}`,
      sipSecret: extPassword,
      webrtcEnabled: extWebRtc,
      status: editingExt ? editingExt.status : 'online',
      assignedAgent: extAgent,
    };

    onSaveExtension(ext);
    setIsModalOpen(false);
  };

  const toggleWebRtc = (ext: Extension) => {
    onSaveExtension({
      ...ext,
      webrtcEnabled: !ext.webrtcEnabled,
    });
  };

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <PhoneCall className="w-6 h-6 text-primary" />
            Extensions (Dahili Hatlar) & DIDs
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Manage SIP client accounts for human agents, WebRTC browser softphone access, and DID numbers.
          </p>
        </div>

        <Button onClick={openAddModal} className="gap-2">
          <Plus className="w-4 h-4" />
          <span>Yeni Dahili Hat Ekle</span>
        </Button>
      </div>

      {/* DIDs Assigned Banner */}
      <Card className="bg-gradient-to-r from-primary/5 via-card to-card border-primary/20">
        <CardContent className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary font-bold">
              <Hash className="w-5 h-5" />
            </div>
            <div>
              <p className="text-xs font-bold text-foreground">Kiracıya Tahsisli Dış Santral Numaraları (DIDs)</p>
              <div className="flex flex-wrap gap-2 mt-1">
                {tenant.assignedDids.length > 0 ? (
                  tenant.assignedDids.map((did) => (
                    <span
                      key={did}
                      className="px-2 py-0.5 rounded bg-primary/10 text-primary font-mono text-xs font-semibold"
                    >
                      {did}
                    </span>
                  ))
                ) : (
                  <span className="text-xs text-muted-foreground italic">Numara atanmamış</span>
                )}
              </div>
            </div>
          </div>
          <Badge variant="success">Kamailio Route: Inbound Workflow Match</Badge>
        </CardContent>
      </Card>

      {/* Extensions Table */}
      <Card>
        <CardHeader>
          <CardTitle>Dahili Hatlar Listesi (FreeSWITCH / SIP.js)</CardTitle>
          <CardDescription>
            Her dahili hat, web softphone veya masaüstü IP telefon (Yealink, Grandstream vb.) üzerinden tescil edilebilir.
          </CardDescription>
        </CardHeader>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-muted/40 text-muted-foreground uppercase font-semibold border-b border-border">
              <tr>
                <th className="px-5 py-3">Dahili No</th>
                <th className="px-5 py-3">Hat Tanımı & Temsilci</th>
                <th className="px-5 py-3">SIP Kullanıcı Adı</th>
                <th className="px-5 py-3">WebRTC Erişimi</th>
                <th className="px-5 py-3">Durum</th>
                <th className="px-5 py-3 text-right">İşlemler</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {extensions.map((ext) => (
                <tr key={ext.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-5 py-4">
                    <span className="font-mono text-base font-bold text-primary px-2.5 py-1 rounded bg-primary/10">
                      {ext.extensionNumber}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    <div className="font-semibold text-foreground text-sm">{ext.name}</div>
                    {ext.assignedAgent && (
                      <div className="text-[11px] text-muted-foreground flex items-center gap-1 mt-0.5">
                        <UserCheck className="w-3 h-3 text-emerald-500" />
                        <span>{ext.assignedAgent}</span>
                      </div>
                    )}
                  </td>
                  <td className="px-5 py-4 font-mono text-muted-foreground">
                    {ext.sipUsername}@{tenant.domain}
                  </td>
                  <td className="px-5 py-4">
                    <button
                      onClick={() => toggleWebRtc(ext)}
                      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold transition-all ${
                        ext.webrtcEnabled
                          ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                          : 'bg-muted text-muted-foreground border border-border'
                      }`}
                    >
                      <Globe className="w-3 h-3" />
                      <span>{ext.webrtcEnabled ? 'Açık (WebRTC)' : 'Kapalı (Yalnızca SIP)'}</span>
                    </button>
                  </td>
                  <td className="px-5 py-4">
                    {ext.status === 'online' && <Badge variant="success">Online (Kayıtlı)</Badge>}
                    {ext.status === 'busy' && <Badge variant="warning">Meşgul (Görüşmede)</Badge>}
                    {ext.status === 'offline' && <Badge variant="secondary">Çevrimdışı</Badge>}
                  </td>
                  <td className="px-5 py-4 text-right space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => openEditModal(ext)}
                      className="h-7 text-xs"
                    >
                      <Edit className="w-3 h-3" />
                      Düzenle
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => onDeleteExtension(ext.id)}
                      className="h-7 text-xs"
                    >
                      <Trash2 className="w-3 h-3" />
                      Sil
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Add / Edit Extension Modal */}
      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={editingExt ? `Dahili Düzenle: ${editingExt.extensionNumber}` : 'Yeni Dahili Hat Tanımla'}
        description="FreeSWITCH ve Kamailio veritabanına yeni SIP abonesi ekleyin."
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground">Dahili Numarası</label>
              <input
                type="text"
                required
                placeholder="101"
                value={extNumber}
                onChange={(e) => setExtNumber(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background font-mono"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground">Hat Tanımı / Masa</label>
              <input
                type="text"
                required
                placeholder="Danışma & Reçete Masası"
                value={extName}
                onChange={(e) => setExtName(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-semibold text-foreground">Temsilci Adı (Opsiyonel)</label>
              <input
                type="text"
                placeholder="Eczacı Mehmet Bey"
                value={extAgent}
                onChange={(e) => setExtAgent(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
              />
            </div>
            <div>
              <label className="text-xs font-semibold text-foreground">SIP Şifresi (Secret)</label>
              <input
                type="password"
                required
                value={extPassword}
                onChange={(e) => setExtPassword(e.target.value)}
                className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background font-mono"
              />
            </div>
          </div>

          <div className="flex items-center gap-3 p-3 bg-secondary/50 rounded-xl">
            <input
              type="checkbox"
              id="webrtc_chk"
              checked={extWebRtc}
              onChange={(e) => setExtWebRtc(e.target.checked)}
              className="w-4 h-4 rounded text-primary"
            />
            <label htmlFor="webrtc_chk" className="text-xs text-foreground font-medium cursor-pointer">
              WebRTC Tarayıcı Softphone Erişimini Aktif Et (WSS / SIP.js)
            </label>
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsModalOpen(false)}>
              İptal
            </Button>
            <Button type="submit">Kaydet</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
