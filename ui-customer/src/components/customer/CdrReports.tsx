import React, { useState } from 'react';
import {
  FileSpreadsheet,
  Search,
  Play,
  Pause,
  Download,
  ChevronRight,
} from 'lucide-react';
import { CdrRecord } from '../../types';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
import { Card, CardHeader, CardTitle, CardContent } from '../ui/Card';
import { Modal } from '../ui/Modal';

interface CdrReportsProps {
  cdrs: CdrRecord[];
  tenantCurrency: string;
}

export const CdrReports: React.FC<CdrReportsProps> = ({ cdrs, tenantCurrency: _tenantCurrency }) => {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [selectedCdr, setSelectedCdr] = useState<CdrRecord | null>(null);

  // Audio Player Mockup state
  const [playingCdrId, setPlayingCdrId] = useState<string | null>(null);
  const playProgress = 35; // 35%

  const filteredCdrs = cdrs.filter((c) => {
    const matchesSearch =
      c.caller.toLowerCase().includes(search.toLowerCase()) ||
      c.did.toLowerCase().includes(search.toLowerCase()) ||
      c.summary.toLowerCase().includes(search.toLowerCase());

    if (statusFilter === 'all') return matchesSearch;
    return matchesSearch && c.resolutionStatus === statusFilter;
  });

  const togglePlayAudio = (cdrId: string) => {
    if (playingCdrId === cdrId) {
      setPlayingCdrId(null);
    } else {
      setPlayingCdrId(cdrId);
    }
  };

  const getStatusBadge = (status: CdrRecord['resolutionStatus']) => {
    switch (status) {
      case 'ai_resolved':
        return <Badge variant="success">AI Çözüldü (%100)</Badge>;
      case 'transferred_agent':
        return <Badge variant="default">Temsilciye Aktarıldı</Badge>;
      case 'user_hangup':
        return <Badge variant="secondary">Kullanıcı Kapattı</Badge>;
      case 'credit_exhausted':
        return <Badge variant="destructive">Kredi Bitti</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <FileSpreadsheet className="w-6 h-6 text-primary" />
            CDR & Çağrı Raporları
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Geçmiş çağrı kayıtları, konuşma süreleri, Diameter Ro maliyetleri ve yapay zeka başarı oranları.
          </p>
        </div>

        <Button
          variant="outline"
          onClick={() => alert('CDR Raporu CSV olarak dışa aktarıldı!')}
          className="gap-2 text-xs"
        >
          <Download className="w-4 h-4" />
          <span>Export CSV / Excel</span>
        </Button>
      </div>

      {/* Floating Audio Player Bar (if active) */}
      {playingCdrId && (
        <Card className="border-2 border-primary/40 bg-card/95 backdrop-blur shadow-lg animate-in slide-in-from-top-2">
          <CardContent className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <button
                onClick={() => setPlayingCdrId(null)}
                className="w-10 h-10 rounded-full bg-primary text-primary-foreground flex items-center justify-center hover:scale-105 transition-transform"
              >
                <Pause className="w-5 h-5" />
              </button>
              <div>
                <div className="text-xs font-bold text-foreground flex items-center gap-2">
                  <span>Ses Kaydı Çalınıyor: {playingCdrId}</span>
                  <Badge variant="outline" className="font-mono text-[10px]">FreeSWITCH Stereo WAV</Badge>
                </div>
                <div className="text-[11px] text-muted-foreground">
                  Sol Kanal: Arayan • Sağ Kanal: AI Agent Ayşe
                </div>
              </div>
            </div>

            {/* Simulated Waveform & Progress */}
            <div className="flex-1 max-w-md flex items-center gap-2">
              <span className="text-[11px] font-mono text-muted-foreground">00:32</span>
              <div className="flex-1 h-3 bg-secondary rounded-full overflow-hidden relative cursor-pointer">
                <div
                  className="h-full bg-primary rounded-full transition-all"
                  style={{ width: `${playProgress}%` }}
                />
              </div>
              <span className="text-[11px] font-mono text-muted-foreground">01:34</span>
            </div>

            <div className="flex items-center gap-2">
              <Button size="sm" variant="ghost" className="h-8 text-xs">
                1.0x
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => alert('Ses dosyası indiriliyor...')}
                className="h-8 text-xs gap-1"
              >
                <Download className="w-3.5 h-3.5" />
                İndir
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Filters & Table */}
      <Card>
        <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <CardTitle>Çağrı Detay Kayıtları (CDR)</CardTitle>
            <div className="flex rounded-lg bg-secondary p-0.5 text-xs">
              <button
                onClick={() => setStatusFilter('all')}
                className={`px-2.5 py-1 rounded-md transition-all ${
                  statusFilter === 'all' ? 'bg-card font-medium text-foreground shadow-xs' : 'text-muted-foreground'
                }`}
              >
                Tümü
              </button>
              <button
                onClick={() => setStatusFilter('ai_resolved')}
                className={`px-2.5 py-1 rounded-md transition-all ${
                  statusFilter === 'ai_resolved' ? 'bg-card font-medium text-foreground shadow-xs' : 'text-muted-foreground'
                }`}
              >
                AI Çözüldü
              </button>
              <button
                onClick={() => setStatusFilter('transferred_agent')}
                className={`px-2.5 py-1 rounded-md transition-all ${
                  statusFilter === 'transferred_agent' ? 'bg-card font-medium text-foreground shadow-xs' : 'text-muted-foreground'
                }`}
              >
                Temsilciye Aktarıldı
              </button>
            </div>
          </div>

          <div className="relative w-full sm:w-64">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              placeholder="Numara veya özet ara..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 text-xs rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
            />
          </div>
        </CardHeader>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-muted/40 text-muted-foreground uppercase font-semibold border-b border-border">
              <tr>
                <th className="px-5 py-3">Tarih / Saat</th>
                <th className="px-5 py-3">Arayan / DID</th>
                <th className="px-5 py-3">Süre</th>
                <th className="px-5 py-3">Maliyet (Ro)</th>
                <th className="px-5 py-3">Sonuç Durumu</th>
                <th className="px-5 py-3">AI / MCP</th>
                <th className="px-5 py-3 text-right">Kayıt / Detay</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {filteredCdrs.map((cdr) => (
                <tr key={cdr.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-5 py-3.5 font-mono text-muted-foreground">
                    {cdr.startTime}
                  </td>
                  <td className="px-5 py-3.5">
                    <div className="font-mono font-bold text-foreground">{cdr.caller}</div>
                    <div className="text-[11px] text-muted-foreground font-mono">DID: {cdr.did}</div>
                  </td>
                  <td className="px-5 py-3.5 font-mono">
                    {Math.floor(cdr.durationSeconds / 60)}m {cdr.durationSeconds % 60}s
                  </td>
                  <td className="px-5 py-3.5 font-mono font-semibold text-foreground">
                    {cdr.cost.toFixed(2)} {cdr.currency}
                  </td>
                  <td className="px-5 py-3.5">
                    {getStatusBadge(cdr.resolutionStatus)}
                  </td>
                  <td className="px-5 py-3.5">
                    <div className="font-medium text-foreground">{cdr.agentPersona}</div>
                    <div className="text-[11px] text-amber-500 font-mono">
                      {cdr.mcpCallsCount} MCP çağrısı
                    </div>
                  </td>
                  <td className="px-5 py-3.5 text-right space-x-1.5">
                    <Button
                      variant={playingCdrId === cdr.id ? 'primary' : 'outline'}
                      size="sm"
                      onClick={() => togglePlayAudio(cdr.id)}
                      className="h-7 px-2.5 text-xs gap-1"
                    >
                      {playingCdrId === cdr.id ? <Pause className="w-3 h-3" /> : <Play className="w-3 h-3" />}
                      <span>Dinle</span>
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSelectedCdr(cdr)}
                      className="h-7 px-2 text-xs"
                    >
                      <ChevronRight className="w-4 h-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* CDR Details Modal */}
      <Modal
        isOpen={!!selectedCdr}
        onClose={() => setSelectedCdr(null)}
        title="Çağrı & AI Çözüm Analiz Özeti"
        description={`Call UUID: ${selectedCdr?.callId}`}
      >
        {selectedCdr && (
          <div className="space-y-4 text-xs">
            <div className="grid grid-cols-2 gap-3 p-3 bg-muted/40 rounded-xl">
              <div>
                <span className="text-muted-foreground">Arayan:</span>
                <p className="font-mono font-bold text-foreground text-sm">{selectedCdr.caller}</p>
              </div>
              <div>
                <span className="text-muted-foreground">Servis DID:</span>
                <p className="font-mono font-bold text-foreground text-sm">{selectedCdr.did}</p>
              </div>
              <div>
                <span className="text-muted-foreground">Süre:</span>
                <p className="font-bold text-foreground">{selectedCdr.durationSeconds} saniye</p>
              </div>
              <div>
                <span className="text-muted-foreground">Diameter Ro Ücreti:</span>
                <p className="font-mono font-bold text-emerald-500">{selectedCdr.cost.toFixed(2)} {selectedCdr.currency}</p>
              </div>
            </div>

            <div>
              <span className="font-semibold text-foreground">AI Görüşme Özeti (LLM Summary)</span>
              <p className="mt-1 p-3 bg-secondary/60 rounded-xl text-foreground leading-relaxed">
                {selectedCdr.summary}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="p-3 bg-secondary/40 rounded-xl">
                <span className="text-muted-foreground">MOS Ses Kalitesi Skoru:</span>
                <p className="font-bold text-foreground text-base mt-0.5">{selectedCdr.mosScore} / 5.0 (HD)</p>
              </div>
              <div className="p-3 bg-secondary/40 rounded-xl">
                <span className="text-muted-foreground">Çalıştırılan Dış Araçlar:</span>
                <p className="font-bold text-foreground text-base mt-0.5">{selectedCdr.mcpCallsCount} Araç</p>
              </div>
            </div>

            <div className="flex justify-end pt-3 border-t border-border">
              <Button onClick={() => setSelectedCdr(null)}>Kapat</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};
