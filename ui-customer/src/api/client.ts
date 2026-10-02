import {
  Tenant,
  DIDNumber,
  DiameterMetric,
  Extension,
  AiPersonaConfig,
  McpServerConfig,
  FaqItem,
  CdrRecord,
} from '../types';
import {
  initialTenants,
  initialDids,
  initialDiameterMetric,
  initialExtensions,
  initialAiPersonas,
  initialMcpServers,
  initialFaqs,
  initialCdrs,
} from './mockData';

// Storage keys
const KEY_TENANTS = 'cpaas_tenants';
const KEY_DIDS = 'cpaas_dids';
const KEY_METRICS = 'cpaas_metrics';
const KEY_EXTENSIONS = 'cpaas_extensions';
const KEY_PERSONAS = 'cpaas_personas';
const KEY_MCP = 'cpaas_mcp';
const KEY_FAQS = 'cpaas_faqs';
const KEY_CDRS = 'cpaas_cdrs';

function loadOrInit<T>(key: string, initial: T): T {
  try {
    const raw = localStorage.getItem(key);
    if (raw) return JSON.parse(raw);
  } catch (e) {
    console.error('Storage read error', e);
  }
  return initial;
}

function save<T>(key: string, data: T): void {
  try {
    localStorage.setItem(key, JSON.stringify(data));
  } catch (e) {
    console.error('Storage write error', e);
  }
}

export const api = {
  // --- Admin: Tenants ---
  async getTenants(): Promise<Tenant[]> {
    return loadOrInit(KEY_TENANTS, initialTenants);
  },

  async saveTenant(tenant: Tenant): Promise<Tenant[]> {
    const list = await this.getTenants();
    const idx = list.findIndex((t) => t.id === tenant.id);
    let updated: Tenant[];
    if (idx >= 0) {
      updated = [...list];
      updated[idx] = tenant;
    } else {
      updated = [tenant, ...list];
    }
    save(KEY_TENANTS, updated);
    return updated;
  },

  async toggleTenantStatus(tenantId: string): Promise<Tenant[]> {
    const list = await this.getTenants();
    const updated = list.map((t) =>
      t.id === tenantId
        ? { ...t, status: (t.status === 'active' ? 'frozen' : 'active') as 'active' | 'frozen' }
        : t
    );
    save(KEY_TENANTS, updated);
    return updated;
  },

  // --- Admin: DIDs ---
  async getDids(): Promise<DIDNumber[]> {
    return loadOrInit(KEY_DIDS, initialDids);
  },

  async assignDid(didId: string, tenantId: string | null): Promise<DIDNumber[]> {
    const list = await this.getDids();
    const tenants = await this.getTenants();
    const tenant = tenantId ? tenants.find((t) => t.id === tenantId) : null;

    const updated = list.map((d) => {
      if (d.id === didId) {
        return {
          ...d,
          tenantId: tenantId || null,
          tenantName: tenant ? tenant.name : null,
          status: (tenantId ? 'assigned' : 'available') as 'assigned' | 'available',
          assignedAt: tenantId ? new Date().toISOString().split('T')[0] : null,
        };
      }
      return d;
    });
    save(KEY_DIDS, updated);

    // Also update tenant assigned DIDs
    const updatedTenants = tenants.map((t) => {
      const tenantDidNumbers = updated
        .filter((d) => d.tenantId === t.id)
        .map((d) => d.number);
      return { ...t, assignedDids: tenantDidNumbers };
    });
    save(KEY_TENANTS, updatedTenants);

    return updated;
  },

  async addDid(did: DIDNumber): Promise<DIDNumber[]> {
    const list = await this.getDids();
    const updated = [did, ...list];
    save(KEY_DIDS, updated);
    return updated;
  },

  // --- Admin: Diameter Metrics ---
  async getDiameterMetrics(): Promise<DiameterMetric> {
    return loadOrInit(KEY_METRICS, initialDiameterMetric);
  },

  // --- Customer: Extensions ---
  async getExtensions(tenantId: string): Promise<Extension[]> {
    const dict = loadOrInit(KEY_EXTENSIONS, initialExtensions);
    return dict[tenantId] || [];
  },

  async saveExtension(tenantId: string, ext: Extension): Promise<Extension[]> {
    const dict = loadOrInit(KEY_EXTENSIONS, initialExtensions);
    const tenantList = dict[tenantId] || [];
    const idx = tenantList.findIndex((e) => e.id === ext.id);
    let updatedList: Extension[];
    if (idx >= 0) {
      updatedList = [...tenantList];
      updatedList[idx] = ext;
    } else {
      updatedList = [ext, ...tenantList];
    }
    dict[tenantId] = updatedList;
    save(KEY_EXTENSIONS, dict);
    return updatedList;
  },

  async deleteExtension(tenantId: string, extId: string): Promise<Extension[]> {
    const dict = loadOrInit(KEY_EXTENSIONS, initialExtensions);
    const tenantList = (dict[tenantId] || []).filter((e) => e.id !== extId);
    dict[tenantId] = tenantList;
    save(KEY_EXTENSIONS, dict);
    return tenantList;
  },

  // --- Customer: AI Persona ---
  async getAiPersona(tenantId: string): Promise<AiPersonaConfig> {
    const dict = loadOrInit(KEY_PERSONAS, initialAiPersonas);
    return (
      dict[tenantId] || {
        agentName: 'Asistan',
        voiceModel: 'gemini-live-aoede-warm',
        provider: 'gemini',
        tone: 'warm_empathic',
        systemPrompt: 'Siz yardımcı bir yapay zeka asistanısınız.',
        fillerPhrases: ['Bir saniye lütfen...', 'Kontrol ediyorum...'],
        byokKey: '',
        byokProvider: 'gemini',
        ragSimilarityThreshold: 0.8,
        silenceTimeoutSec: 2.0,
        vadSensitivity: 'medium',
      }
    );
  },

  async saveAiPersona(tenantId: string, config: AiPersonaConfig): Promise<AiPersonaConfig> {
    const dict = loadOrInit(KEY_PERSONAS, initialAiPersonas);
    dict[tenantId] = config;
    save(KEY_PERSONAS, dict);
    return config;
  },

  // --- Customer: MCP Servers ---
  async getMcpServers(tenantId: string): Promise<McpServerConfig[]> {
    const dict = loadOrInit(KEY_MCP, initialMcpServers);
    return dict[tenantId] || [];
  },

  async saveMcpServer(tenantId: string, server: McpServerConfig): Promise<McpServerConfig[]> {
    const dict = loadOrInit(KEY_MCP, initialMcpServers);
    const list = dict[tenantId] || [];
    const idx = list.findIndex((s) => s.id === server.id);
    let updated: McpServerConfig[];
    if (idx >= 0) {
      updated = [...list];
      updated[idx] = server;
    } else {
      updated = [server, ...list];
    }
    dict[tenantId] = updated;
    save(KEY_MCP, dict);
    return updated;
  },

  // --- Customer: FAQs ---
  async getFaqs(tenantId: string): Promise<FaqItem[]> {
    const dict = loadOrInit(KEY_FAQS, initialFaqs);
    return dict[tenantId] || [];
  },

  async saveFaq(tenantId: string, faq: FaqItem): Promise<FaqItem[]> {
    const dict = loadOrInit(KEY_FAQS, initialFaqs);
    const list = dict[tenantId] || [];
    const idx = list.findIndex((f) => f.id === faq.id);
    let updated: FaqItem[];
    if (idx >= 0) {
      updated = [...list];
      updated[idx] = faq;
    } else {
      updated = [faq, ...list];
    }
    dict[tenantId] = updated;
    save(KEY_FAQS, dict);
    return updated;
  },

  async deleteFaq(tenantId: string, faqId: string): Promise<FaqItem[]> {
    const dict = loadOrInit(KEY_FAQS, initialFaqs);
    const list = (dict[tenantId] || []).filter((f) => f.id !== faqId);
    dict[tenantId] = list;
    save(KEY_FAQS, dict);
    return list;
  },

  // --- Customer / Global: CDRs ---
  async getCdrs(tenantId?: string): Promise<CdrRecord[]> {
    const list = loadOrInit(KEY_CDRS, initialCdrs);
    if (!tenantId) return list;
    return list.filter((c) => c.tenantId === tenantId);
  },

  async addCdr(record: CdrRecord): Promise<CdrRecord[]> {
    const list = loadOrInit(KEY_CDRS, initialCdrs);
    const updated = [record, ...list];
    save(KEY_CDRS, updated);
    return updated;
  },
};
