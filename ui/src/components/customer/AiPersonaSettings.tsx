import React, { useState, useEffect } from 'react';
import {
  Bot,
  Key,
  Plus,
  Trash2,
  CheckCircle,
  Eye,
  EyeOff,
  Search,
  Upload,
  Server,
  Save,
} from 'lucide-react';
import { AiPersonaConfig, McpServerConfig, FaqItem } from '../../types';
import { Button } from '../ui/Button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/Card';
import { Badge } from '../ui/Badge';
import { Modal } from '../ui/Modal';

interface AiPersonaSettingsProps {
  tenantId: string;
  persona: AiPersonaConfig;
  mcpServers: McpServerConfig[];
  faqs: FaqItem[];
  onSavePersona: (config: AiPersonaConfig) => void;
  onSaveMcpServer: (server: McpServerConfig) => void;
  onSaveFaq: (faq: FaqItem) => void;
  onDeleteFaq: (faqId: string) => void;
}

export const AiPersonaSettings: React.FC<AiPersonaSettingsProps> = ({
  tenantId: _tenantId,
  persona,
  mcpServers,
  faqs,
  onSavePersona,
  onSaveMcpServer,
  onSaveFaq,
  onDeleteFaq,
}) => {
  // Local form state for Persona
  const [formData, setFormData] = useState<AiPersonaConfig>({ ...persona });
  const [showApiKey, setShowApiKey] = useState(false);
  const [newFiller, setNewFiller] = useState('');
  const [isSavedAlert, setIsSavedAlert] = useState(false);

  // FAQ Modal states
  const [isFaqModalOpen, setIsFaqModalOpen] = useState(false);
  const [faqQuestion, setFaqQuestion] = useState('');
  const [faqAnswer, setFaqAnswer] = useState('');
  const [faqCategory, setFaqCategory] = useState('Genel Bilgi');
  const [faqSearch, setFaqSearch] = useState('');

  // MCP Server Modal states
  const [isMcpModalOpen, setIsMcpModalOpen] = useState(false);
  const [mcpName, setMcpName] = useState('');
  const [mcpUrl, setMcpUrl] = useState('');
  const [mcpToken, setMcpToken] = useState('');

  useEffect(() => {
    setFormData({ ...persona });
  }, [persona]);

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    onSavePersona(formData);
    setIsSavedAlert(true);
    setTimeout(() => setIsSavedAlert(false), 3000);
  };

  const addFiller = () => {
    if (!newFiller.trim()) return;
    setFormData({
      ...formData,
      fillerPhrases: [...formData.fillerPhrases, newFiller.trim()],
    });
    setNewFiller('');
  };

  const removeFiller = (index: number) => {
    const arr = [...formData.fillerPhrases];
    arr.splice(index, 1);
    setFormData({ ...formData, fillerPhrases: arr });
  };

  const handleCreateFaq = (e: React.FormEvent) => {
    e.preventDefault();
    if (!faqQuestion || !faqAnswer) return;

    const newFaq: FaqItem = {
      id: `faq_${Date.now()}`,
      question: faqQuestion,
      answer: faqAnswer,
      category: faqCategory,
      lastUpdated: new Date().toISOString().split('T')[0],
    };

    onSaveFaq(newFaq);
    setIsFaqModalOpen(false);
    setFaqQuestion('');
    setFaqAnswer('');
  };

  const handleAddMcpServer = (e: React.FormEvent) => {
    e.preventDefault();
    if (!mcpName || !mcpUrl) return;

    const newServer: McpServerConfig = {
      id: `mcp_${Date.now()}`,
      name: mcpName,
      url: mcpUrl,
      authToken: mcpToken,
      status: 'connected',
      lastPingMs: 22,
      tools: [
        {
          name: 'query_api',
          description: 'Auto-discovered tool from MCP manifest',
          inputSchema: { query: 'string' },
          enabled: true,
        },
      ],
    };

    onSaveMcpServer(newServer);
    setIsMcpModalOpen(false);
    setMcpName('');
    setMcpUrl('');
    setMcpToken('');
  };

  const filteredFaqs = faqs.filter(
    (f) =>
      f.question.toLowerCase().includes(faqSearch.toLowerCase()) ||
      f.answer.toLowerCase().includes(faqSearch.toLowerCase()) ||
      f.category.toLowerCase().includes(faqSearch.toLowerCase())
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Bot className="w-6 h-6 text-primary" />
            AI Persona & Agent Settings
          </h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Configure agent identity, voice parameters, BYOK provider API keys, and knowledge base.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {isSavedAlert && (
            <span className="text-xs text-emerald-500 font-semibold flex items-center gap-1 animate-in fade-in">
              <CheckCircle className="w-4 h-4" /> Persona Saved!
            </span>
          )}
          <Button onClick={handleSave} className="gap-2">
            <Save className="w-4 h-4" />
            <span>Save Settings</span>
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left 2 Cols: Main Persona Config */}
        <div className="lg:col-span-2 space-y-6">
          {/* Identity & Voice Card */}
          <Card>
            <CardHeader>
              <CardTitle>Agent Identity & Voice Model</CardTitle>
              <CardDescription>
                Customize name, voice synthesis, emotional tone, and speech detection.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="text-xs font-semibold text-foreground">
                    Agent Name (Arayanlara Tanıtılacak İsim)
                  </label>
                  <input
                    type="text"
                    value={formData.agentName}
                    onChange={(e) => setFormData({ ...formData, agentName: e.target.value })}
                    className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
                    placeholder="e.g. Ayşe (Eczacı Asistanı)"
                  />
                </div>

                <div>
                  <label className="text-xs font-semibold text-foreground">Voice Engine Model</label>
                  <select
                    value={formData.voiceModel}
                    onChange={(e) => setFormData({ ...formData, voiceModel: e.target.value })}
                    className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background font-mono"
                  >
                    <option value="gemini-live-aoede-warm">gemini-live-aoede-warm (Kadın - Samimi & Sıcak)</option>
                    <option value="gemini-live-fenrir-clear">gemini-live-fenrir-clear (Erkek - Net & Hızlı)</option>
                    <option value="gemini-live-kore-calm">gemini-live-kore-calm (Kadın - Sakin & Güven Veren)</option>
                    <option value="gemini-live-puck-energetic">gemini-live-puck-energetic (Erkek - Enerjik)</option>
                    <option value="elevenlabs-turkish-ayse">elevenlabs-turkish-ayse (Ultra Gerçekçi Nöral)</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div>
                  <label className="text-xs font-semibold text-foreground">Conversation Tone</label>
                  <select
                    value={formData.tone}
                    onChange={(e) => setFormData({ ...formData, tone: e.target.value as any })}
                    className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
                  >
                    <option value="warm_empathic">Sıcak & Empatik</option>
                    <option value="professional_strict">Resmi & Kurumsal</option>
                    <option value="cheerful">Esprili & Enerjik</option>
                    <option value="fast_efficient">Hızlı & Çözüm Odaklı</option>
                  </select>
                </div>

                <div>
                  <label className="text-xs font-semibold text-foreground">
                    VAD Sensitivity (Konuşma Algılama)
                  </label>
                  <select
                    value={formData.vadSensitivity}
                    onChange={(e) => setFormData({ ...formData, vadSensitivity: e.target.value as any })}
                    className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
                  >
                    <option value="high">Yüksek (Araya girmeye duyarlı)</option>
                    <option value="medium">Orta (Dengeli)</option>
                    <option value="low">Düşük (Cümle bitene kadar bekle)</option>
                  </select>
                </div>

                <div>
                  <label className="text-xs font-semibold text-foreground">
                    Sessizlik Zaman Aşımı (sn)
                  </label>
                  <input
                    type="number"
                    step="0.1"
                    min="1.0"
                    max="5.0"
                    value={formData.silenceTimeoutSec}
                    onChange={(e) => setFormData({ ...formData, silenceTimeoutSec: parseFloat(e.target.value) })}
                    className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
                  />
                </div>
              </div>

              <div>
                <label className="text-xs font-semibold text-foreground">
                  System Instructions & Persona Prompt (Karakter & Görev Tanımı)
                </label>
                <textarea
                  rows={5}
                  value={formData.systemPrompt}
                  onChange={(e) => setFormData({ ...formData, systemPrompt: e.target.value })}
                  className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background font-mono leading-relaxed"
                  placeholder="Sen işletmemizin sanal santral asistanısın..."
                />
              </div>
            </CardContent>
          </Card>

          {/* Filler Phrases Card */}
          <Card>
            <CardHeader>
              <CardTitle>Dynamic Filler Phrases (Dolgu & Bekletme Cümleleri)</CardTitle>
              <CardDescription>
                AI dış MCP servisini sorgularken hatta sessizlik oluşmaması için hemen telaffuz edilecek cümleler.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="e.g. Hemen sistemimizden kontrol ediyorum..."
                  value={newFiller}
                  onChange={(e) => setNewFiller(e.target.value)}
                  className="flex-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
                />
                <Button size="sm" onClick={addFiller} className="gap-1.5">
                  <Plus className="w-3.5 h-3.5" />
                  <span>Ekle</span>
                </Button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {formData.fillerPhrases.map((phrase, idx) => (
                  <div
                    key={idx}
                    className="flex items-center justify-between p-2.5 rounded-lg border border-border bg-secondary/40 text-xs"
                  >
                    <span className="truncate pr-2 italic">"{phrase}"</span>
                    <button
                      onClick={() => removeFiller(idx)}
                      className="text-rose-500 hover:text-rose-700 shrink-0"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* FAQ & Knowledge Base Manager */}
          <Card>
            <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <CardTitle>FAQ & Knowledge Base (RAG Hafızası)</CardTitle>
                <CardDescription>
                  Ajanın çağrıyı karşılarken ilk kontrol ettiği şirket içi sık sorulan sorular ve cevaplar.
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => alert('PDF/TXT Belge Yükleme Mock: Dosya vektör veri tabanına işlendi!')}
                  className="gap-1.5 text-xs"
                >
                  <Upload className="w-3.5 h-3.5" />
                  <span>Upload Doc</span>
                </Button>
                <Button
                  size="sm"
                  onClick={() => setIsFaqModalOpen(true)}
                  className="gap-1.5 text-xs"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add FAQ</span>
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="relative">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                <input
                  type="text"
                  placeholder="Bilgi bankasında ara..."
                  value={faqSearch}
                  onChange={(e) => setFaqSearch(e.target.value)}
                  className="w-full pl-9 pr-3 py-1.5 text-xs rounded-lg border border-border bg-background"
                />
              </div>

              <div className="space-y-2 max-h-72 overflow-y-auto">
                {filteredFaqs.map((faq) => (
                  <div
                    key={faq.id}
                    className="p-3 rounded-xl border border-border bg-card hover:border-primary/40 transition-colors space-y-1.5"
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-foreground text-xs">{faq.question}</span>
                      <div className="flex items-center gap-2">
                        <Badge variant="secondary" className="text-[10px]">{faq.category}</Badge>
                        <button
                          onClick={() => onDeleteFaq(faq.id)}
                          className="text-muted-foreground hover:text-rose-500"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                    <p className="text-xs text-muted-foreground leading-relaxed">{faq.answer}</p>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right 1 Col: BYOK & MCP Configurations */}
        <div className="space-y-6">
          {/* BYOK API Key Card */}
          <Card className="border-l-4 border-l-primary">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Key className="w-4 h-4 text-primary" />
                BYOK (Bring Your Own Key)
              </CardTitle>
              <CardDescription>
                Kendi model API anahtarınızı tanımlayarak platform kotalarından bağımsız çalışın.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-foreground">API Sağlayıcı</label>
                <select
                  value={formData.byokProvider}
                  onChange={(e) => setFormData({ ...formData, byokProvider: e.target.value as any })}
                  className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
                >
                  <option value="gemini">Google Gemini 2.0 (Multimodal Live WebSocket)</option>
                  <option value="openai">OpenAI Realtime API (GPT-4o Voice)</option>
                  <option value="anthropic">Anthropic Claude 3.5 Sonnet</option>
                </select>
              </div>

              <div>
                <label className="text-xs font-semibold text-foreground">API Key</label>
                <div className="relative mt-1">
                  <input
                    type={showApiKey ? 'text' : 'password'}
                    value={formData.byokKey}
                    onChange={(e) => setFormData({ ...formData, byokKey: e.target.value })}
                    placeholder="AIzaSy... or sk-proj-..."
                    className="w-full pl-3 pr-9 py-2 text-xs rounded-lg border border-border bg-background font-mono"
                  />
                  <button
                    type="button"
                    onClick={() => setShowApiKey(!showApiKey)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  >
                    {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </button>
                </div>
                <p className="text-[10px] text-muted-foreground mt-1">
                  Anahtar AES-256 ile şifrelenir ve yalnızca sizin aramalarınızda kullanılır.
                </p>
              </div>

              <div className="p-3 bg-secondary/40 rounded-xl flex items-center justify-between text-xs">
                <span>BYOK Durumu:</span>
                <span className="font-semibold text-emerald-500 flex items-center gap-1">
                  <CheckCircle className="w-3.5 h-3.5" /> Doğrulandı
                </span>
              </div>
            </CardContent>
          </Card>

          {/* MCP Server Integration */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Server className="w-4 h-4 text-amber-500" />
                  MCP Server Entegrasyonu
                </CardTitle>
                <CardDescription>Model Context Protocol sunucuları</CardDescription>
              </div>
              <Button size="sm" variant="outline" onClick={() => setIsMcpModalOpen(true)} className="h-7 px-2">
                <Plus className="w-3.5 h-3.5" />
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              {mcpServers.map((server) => (
                <div
                  key={server.id}
                  className="p-3 rounded-xl border border-border bg-secondary/30 space-y-2 text-xs"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-bold text-foreground truncate">{server.name}</span>
                    <Badge variant="success">{server.lastPingMs}ms</Badge>
                  </div>
                  <div className="text-[11px] font-mono text-muted-foreground truncate">
                    {server.url}
                  </div>
                  <div className="pt-1 border-t border-border/40">
                    <p className="text-[10px] font-semibold text-muted-foreground uppercase">
                      Bağlı Araçlar ({server.tools.length}):
                    </p>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {server.tools.map((t) => (
                        <span
                          key={t.name}
                          className="px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 dark:text-amber-400 font-mono text-[10px]"
                        >
                          {t.name}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Add FAQ Modal */}
      <Modal
        isOpen={isFaqModalOpen}
        onClose={() => setIsFaqModalOpen(false)}
        title="Yeni Soru-Cevap (FAQ) Ekle"
        description="Ajanın arayan müşterilere doğrudan vereceği bilgiyi girin."
      >
        <form onSubmit={handleCreateFaq} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-foreground">Soru / Müşteri Talebi</label>
            <input
              type="text"
              required
              placeholder="e.g. Cumartesi günleri açık mısınız?"
              value={faqQuestion}
              onChange={(e) => setFaqQuestion(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">Kategori</label>
            <input
              type="text"
              value={faqCategory}
              onChange={(e) => setFaqCategory(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">Ajanın Vereceği Cevap</label>
            <textarea
              rows={4}
              required
              placeholder="Evet, cumartesi günleri 09:00 - 18:00 arası açığız."
              value={faqAnswer}
              onChange={(e) => setFaqAnswer(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsFaqModalOpen(false)}>
              İptal
            </Button>
            <Button type="submit">Kaydet</Button>
          </div>
        </form>
      </Modal>

      {/* Add MCP Modal */}
      <Modal
        isOpen={isMcpModalOpen}
        onClose={() => setIsMcpModalOpen(false)}
        title="MCP Sunucusu Ekle"
        description="Harici ERP, CRM veya Veritabanı MCP sunucusu bağlayın."
      >
        <form onSubmit={handleAddMcpServer} className="space-y-4">
          <div>
            <label className="text-xs font-semibold text-foreground">Sunucu Adı</label>
            <input
              type="text"
              required
              placeholder="e.g. Eczane Stok MCP Sunucusu"
              value={mcpName}
              onChange={(e) => setMcpName(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">MCP Sunucu URL</label>
            <input
              type="text"
              required
              placeholder="https://api.domain.com/mcp/v1"
              value={mcpUrl}
              onChange={(e) => setMcpUrl(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background font-mono"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-foreground">Bearer Token (Opsiyonel)</label>
            <input
              type="text"
              placeholder="Bearer secret_token_xyz"
              value={mcpToken}
              onChange={(e) => setMcpToken(e.target.value)}
              className="w-full mt-1 px-3 py-2 text-xs rounded-lg border border-border bg-background font-mono"
            />
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t border-border">
            <Button variant="outline" type="button" onClick={() => setIsMcpModalOpen(false)}>
              İptal
            </Button>
            <Button type="submit">Sunucuyu Bağla</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};
