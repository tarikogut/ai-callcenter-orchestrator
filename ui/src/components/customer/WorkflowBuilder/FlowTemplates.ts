import { Node, Edge } from '@xyflow/react';

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  nodes: Node[];
  edges: Edge[];
}

export const workflowTemplates: Record<string, WorkflowTemplate> = {
  pharmacy: {
    id: 'flow_pharmacy_duty',
    name: 'Eczane Gündüz Karşılama, Stok Sorgu & Nöbet Akışı',
    description: 'Inbound Call -> Working Hours -> AI Agent (Ayşe) -> Check Medicine MCP -> Transfer Ext 101 or Hangup',
    nodes: [
      {
        id: 'node_trigger',
        type: 'InboundCallTrigger',
        position: { x: 300, y: 50 },
        data: {
          label: 'Gelen Çağrı (0850 111 0001)',
          type: 'InboundCallTrigger',
          config: { did: '+90 850 111 0001', triggerType: 'call' },
        },
      },
      {
        id: 'node_condition_hours',
        type: 'ConditionNode',
        position: { x: 300, y: 190 },
        data: {
          label: 'Mesai / Nöbet Kontrolü',
          type: 'ConditionNode',
          config: { expression: 'is_open_hours == true || is_duty_tonight == true' },
        },
      },
      {
        id: 'node_ai_agent',
        type: 'AiVoiceAgentNode',
        position: { x: 180, y: 350 },
        data: {
          label: 'Ayşe (Eczacı Asistanı)',
          type: 'AiVoiceAgentNode',
          config: {
            agentName: 'Ayşe (Eczacı Asistanı)',
            voice: 'gemini-live-aoede-warm',
            persona: 'Hayat Eczanesi asistanısın. İlaç sorulursa stok MCP çalıştır. Reçeteli soru varsa Eczacı Mehmet Bey\'e (101) aktar.',
            fillerPhrases: ['Hemen stoklarıma bakıyorum...', 'Bir saniye sistemden kontrol ediyorum...'],
          },
        },
      },
      {
        id: 'node_mcp_stock',
        type: 'McpToolNode',
        position: { x: 180, y: 530 },
        data: {
          label: 'Eczane İlaç Stok MCP',
          type: 'McpToolNode',
          config: {
            mcpServer: 'https://api.hayateczanesi.com/mcp/v1',
            toolName: 'check_medicine_stock',
          },
        },
      },
      {
        id: 'node_transfer',
        type: 'TransferNode',
        position: { x: 480, y: 530 },
        data: {
          label: 'Eczacı Masasına Aktar (101)',
          type: 'TransferNode',
          config: { targetExtension: '101' },
        },
      },
      {
        id: 'node_hangup',
        type: 'HangupNode',
        position: { x: 300, y: 700 },
        data: {
          label: 'Görüşmeyi Sonlandır',
          type: 'HangupNode',
          config: { reason: 'AI Solution Provided (200 OK)' },
        },
      },
    ],
    edges: [
      { id: 'e1', source: 'node_trigger', target: 'node_condition_hours', animated: true },
      { id: 'e2', source: 'node_condition_hours', sourceHandle: 'true', target: 'node_ai_agent', animated: true },
      { id: 'e3', source: 'node_condition_hours', sourceHandle: 'false', target: 'node_transfer' },
      { id: 'e4', source: 'node_ai_agent', target: 'node_mcp_stock', animated: true },
      { id: 'e5', source: 'node_mcp_stock', target: 'node_hangup' },
      { id: 'e6', source: 'node_transfer', target: 'node_hangup' },
    ],
  },
  logistics: {
    id: 'flow_kargo_express',
    name: 'Jet Kargo Takip & Kurye Yönlendirme Akışı',
    description: 'Inbound Call -> Can AI Voice -> Track Shipment MCP -> Transfer Ext 201 or Hangup',
    nodes: [
      {
        id: 'node_kargo_trigger',
        type: 'InboundCallTrigger',
        position: { x: 300, y: 50 },
        data: {
          label: 'Gelen Çağrı (0850 222 3456)',
          type: 'InboundCallTrigger',
          config: { did: '+90 850 222 3456', triggerType: 'call' },
        },
      },
      {
        id: 'node_kargo_ai',
        type: 'AiVoiceAgentNode',
        position: { x: 300, y: 200 },
        data: {
          label: 'Can (Kargo Takip Asistanı)',
          type: 'AiVoiceAgentNode',
          config: {
            agentName: 'Can (Kargo Takip)',
            voice: 'gemini-live-fenrir-clear',
            persona: 'Müşteriden 12 haneli kargo takip kodunu alıp kurye konumunu söyle.',
            fillerPhrases: ['Kargonuzu merkez sistemden sorguluyorum...'],
          },
        },
      },
      {
        id: 'node_kargo_mcp',
        type: 'McpToolNode',
        position: { x: 300, y: 380 },
        data: {
          label: 'Kargo Barkod ERP MCP',
          type: 'McpToolNode',
          config: {
            mcpServer: 'https://erp.jetkargo.com.tr/mcp-gateway',
            toolName: 'track_shipment',
          },
        },
      },
      {
        id: 'node_kargo_hangup',
        type: 'HangupNode',
        position: { x: 300, y: 550 },
        data: {
          label: 'Başarılı Kapanış',
          type: 'HangupNode',
          config: { reason: 'Shipment info delivered' },
        },
      },
    ],
    edges: [
      { id: 'ek1', source: 'node_kargo_trigger', target: 'node_kargo_ai', animated: true },
      { id: 'ek2', source: 'node_kargo_ai', target: 'node_kargo_mcp', animated: true },
      { id: 'ek3', source: 'node_kargo_mcp', target: 'node_kargo_hangup', animated: true },
    ],
  },
};
