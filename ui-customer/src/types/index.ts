export type PortalMode = 'admin' | 'customer';

export interface Tenant {
  id: string;
  name: string;
  domain: string;
  industry: 'pharmacy' | 'logistics' | 'clinic' | 'ecommerce' | 'other';
  status: 'active' | 'frozen';
  balance: number; // in USD or TRY
  currency: string;
  maxConcurrentCalls: number;
  monthlySpendLimit: number;
  currentConcurrentCalls: number;
  assignedDids: string[];
  createdAt: string;
  contactEmail: string;
}

export interface DIDNumber {
  id: string;
  number: string;
  countryCode: string;
  carrier: string;
  tenantId: string | null;
  tenantName: string | null;
  assignedAt: string | null;
  status: 'assigned' | 'available' | 'reserved';
  monthlyCost: number;
}

export interface DiameterMetric {
  activeSessions: number;
  ccrTotalCount: number;
  ccaSuccessRate: number;
  avgRatingLatencyMs: number;
  globalCdrToday: number;
  billedAmountToday: number;
  currency: string;
}

export interface Extension {
  id: string;
  extensionNumber: string; // e.g. "101"
  name: string;
  tenantId: string;
  sipUsername: string;
  sipSecret: string;
  webrtcEnabled: boolean;
  status: 'online' | 'busy' | 'offline';
  assignedAgent?: string;
}

export interface McpTool {
  name: string;
  description: string;
  inputSchema: Record<string, any>;
  enabled: boolean;
}

export interface McpServerConfig {
  id: string;
  name: string;
  url: string;
  authToken: string;
  status: 'connected' | 'error' | 'disconnected';
  lastPingMs: number;
  tools: McpTool[];
}

export interface FaqItem {
  id: string;
  question: string;
  answer: string;
  category: string;
  lastUpdated: string;
}

export interface AiPersonaConfig {
  agentName: string;
  voiceModel: string;
  provider: 'gemini' | 'openai' | 'anthropic' | 'elevenlabs';
  tone: 'warm_empathic' | 'professional_strict' | 'cheerful' | 'fast_efficient';
  systemPrompt: string;
  fillerPhrases: string[];
  byokKey: string;
  byokProvider: 'gemini' | 'openai' | 'anthropic';
  ragSimilarityThreshold: number;
  silenceTimeoutSec: number;
  vadSensitivity: 'low' | 'medium' | 'high';
}

export interface WorkflowNodeData {
  label: string;
  type: 'InboundCallTrigger' | 'AiVoiceAgentNode' | 'McpToolNode' | 'ConditionNode' | 'TransferNode' | 'HangupNode' | 'PlayAudioNode';
  config: Record<string, any>;
  description?: string;
}

export interface CdrRecord {
  id: string;
  callId: string;
  tenantId: string;
  caller: string;
  did: string;
  startTime: string;
  durationSeconds: number;
  cost: number;
  currency: string;
  resolutionStatus: 'ai_resolved' | 'transferred_agent' | 'user_hangup' | 'credit_exhausted';
  agentPersona: string;
  mcpCallsCount: number;
  summary: string;
  audioRecordingUrl?: string;
  mosScore: number;
}

export interface LiveTranscriptEntry {
  id: string;
  timestamp: string;
  sender: 'caller' | 'ai' | 'agent' | 'system';
  text: string;
  mcpAction?: {
    tool: string;
    input: string;
    output: string;
  };
  sentiment?: 'positive' | 'neutral' | 'frustrated';
}
