import React, { useState, useEffect } from 'react';
import type { Node } from '@xyflow/react';
import {
  X,
  Trash2,
  Save,
  Plus,
  Minus,
} from 'lucide-react';
import { Button } from '../../ui/Button';

interface NodeConfigPanelProps {
  selectedNode: Node | null;
  onClose: () => void;
  onUpdateNode: (nodeId: string, data: any) => void;
  onDeleteNode: (nodeId: string) => void;
}

export const NodeConfigPanel: React.FC<NodeConfigPanelProps> = ({
  selectedNode,
  onClose,
  onUpdateNode,
  onDeleteNode,
}) => {
  const [label, setLabel] = useState('');
  const [config, setConfig] = useState<Record<string, any>>({});
  const [newFiller, setNewFiller] = useState('');

  // Update local state when selectedNode changes
  useEffect(() => {
    if (selectedNode) {
      setLabel((selectedNode.data?.label as string) || '');
      setConfig({ ...((selectedNode.data?.config as any) || {}) });
    }
  }, [selectedNode]);

  if (!selectedNode) return null;

  const handleSave = () => {
    onUpdateNode(selectedNode.id, {
      ...selectedNode.data,
      label,
      config,
    });
  };

  const addFillerPhrase = () => {
    if (!newFiller.trim()) return;
    const fillers = config.fillerPhrases ? [...config.fillerPhrases] : [];
    fillers.push(newFiller.trim());
    setConfig({ ...config, fillerPhrases: fillers });
    setNewFiller('');
  };

  const removeFillerPhrase = (index: number) => {
    const fillers = [...(config.fillerPhrases || [])];
    fillers.splice(index, 1);
    setConfig({ ...config, fillerPhrases: fillers });
  };

  const nodeType = selectedNode.type;

  return (
    <div className="w-84 border-l border-border bg-card flex flex-col shrink-0 select-none animate-in slide-in-from-right-4 duration-200">
      {/* Header */}
      <div className="p-4 border-b border-border/60 flex items-center justify-between">
        <div className="min-w-0">
          <div className="text-[11px] font-bold uppercase tracking-wider text-primary">
            Node Configuration
          </div>
          <h4 className="text-sm font-bold text-foreground truncate mt-0.5">
            {selectedNode.type}
          </h4>
        </div>
        <button
          onClick={onClose}
          className="p-1 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Body */}
      <div className="p-4 overflow-y-auto space-y-4 flex-1 text-xs">
        {/* Node Label */}
        <div>
          <label className="font-semibold text-foreground">Node Display Name</label>
          <input
            type="text"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Dynamic Fields based on Node Type */}
        {nodeType === 'InboundCallTrigger' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">Inbound DID Number</label>
              <input
                type="text"
                placeholder="+90 850 111 0001 or * for any"
                value={config.did || ''}
                onChange={(e) => setConfig({ ...config, did: e.target.value })}
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              />
              <p className="text-[10px] text-muted-foreground mt-1">
                Matches the SIP To URI from Kamailio registrar.
              </p>
            </div>
            <div>
              <label className="font-semibold text-foreground">Trigger Channel Type</label>
              <select
                value={config.triggerType || 'call'}
                onChange={(e) => setConfig({ ...config, triggerType: e.target.value })}
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background"
              >
                <option value="call">Voice Call (SIP Inbound RTP)</option>
                <option value="sms">SMS / Message Trigger</option>
              </select>
            </div>
          </div>
        )}

        {nodeType === 'AiVoiceAgentNode' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">Agent Persona Name</label>
              <input
                type="text"
                value={config.agentName || ''}
                onChange={(e) => setConfig({ ...config, agentName: e.target.value })}
                placeholder="e.g. Ayşe, Can"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background"
              />
            </div>

            <div>
              <label className="font-semibold text-foreground">Voice Model Preset</label>
              <select
                value={config.voice || 'gemini-live-aoede-warm'}
                onChange={(e) => setConfig({ ...config, voice: e.target.value })}
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              >
                <option value="gemini-live-aoede-warm">gemini-live-aoede-warm (Kadın, Samimi)</option>
                <option value="gemini-live-fenrir-clear">gemini-live-fenrir-clear (Erkek, Net)</option>
                <option value="gemini-live-puck-energetic">gemini-live-puck-energetic (Canlı)</option>
                <option value="openai-realtime-alloy">openai-realtime-alloy (Dengeli)</option>
                <option value="openai-realtime-shimmer">openai-realtime-shimmer (Yumuşak)</option>
              </select>
            </div>

            <div>
              <label className="font-semibold text-foreground">Prompt Instructions & Persona</label>
              <textarea
                rows={4}
                value={config.persona || ''}
                onChange={(e) => setConfig({ ...config, persona: e.target.value })}
                placeholder="Sen sıcak ve yardımsever bir asistansın..."
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background leading-relaxed"
              />
            </div>

            <div>
              <label className="font-semibold text-foreground">Filler Phrases (Dolgu Cümleleri)</label>
              <p className="text-[10px] text-muted-foreground mb-1.5">
                Spoken immediately when waiting for MCP or LLM processing to eliminate silence.
              </p>
              <div className="space-y-1.5 mb-2 max-h-36 overflow-y-auto">
                {(config.fillerPhrases || []).map((filler: string, idx: number) => (
                  <div key={idx} className="flex items-center gap-1.5 bg-secondary/60 p-1.5 rounded-md text-[11px]">
                    <span className="flex-1 truncate">{filler}</span>
                    <button
                      type="button"
                      onClick={() => removeFillerPhrase(idx)}
                      className="text-rose-500 hover:text-rose-700"
                    >
                      <Minus className="w-3 h-3" />
                    </button>
                  </div>
                ))}
              </div>
              <div className="flex gap-1.5">
                <input
                  type="text"
                  placeholder="e.g. Hemen bakıyorum efendim..."
                  value={newFiller}
                  onChange={(e) => setNewFiller(e.target.value)}
                  className="flex-1 px-2.5 py-1 text-xs rounded border border-border bg-background"
                />
                <Button size="sm" type="button" onClick={addFillerPhrase} className="h-7 px-2">
                  <Plus className="w-3 h-3" />
                </Button>
              </div>
            </div>
          </div>
        )}

        {nodeType === 'McpToolNode' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">MCP Server Endpoint URL</label>
              <input
                type="text"
                value={config.mcpServer || ''}
                onChange={(e) => setConfig({ ...config, mcpServer: e.target.value })}
                placeholder="https://api.hayateczanesi.com/mcp"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              />
            </div>
            <div>
              <label className="font-semibold text-foreground">Tool Identifier</label>
              <input
                type="text"
                value={config.toolName || ''}
                onChange={(e) => setConfig({ ...config, toolName: e.target.value })}
                placeholder="check_medicine_stock"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              />
              <p className="text-[10px] text-muted-foreground mt-1">
                The function invoked by the AI agent via JSON-RPC.
              </p>
            </div>
          </div>
        )}

        {nodeType === 'ConditionNode' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">Logic Expression / Rule</label>
              <input
                type="text"
                value={config.expression || ''}
                onChange={(e) => setConfig({ ...config, expression: e.target.value })}
                placeholder="is_open_hours == true"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              />
              <p className="text-[10px] text-muted-foreground mt-1">
                Examples: <code>is_open_hours == true</code>, <code>sentiment == 'angry'</code>, <code>balance &gt; 10</code>
              </p>
            </div>
          </div>
        )}

        {nodeType === 'TransferNode' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">Target Extension (Dahili)</label>
              <input
                type="text"
                value={config.targetExtension || ''}
                onChange={(e) => setConfig({ ...config, targetExtension: e.target.value })}
                placeholder="101"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              />
              <p className="text-[10px] text-muted-foreground mt-1">
                Transfers call using FreeSWITCH <code>uuid_transfer</code> to registered SIP client.
              </p>
            </div>
          </div>
        )}

        {nodeType === 'PlayAudioNode' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">Audio Prompt File</label>
              <input
                type="text"
                value={config.audioFile || ''}
                onChange={(e) => setConfig({ ...config, audioFile: e.target.value })}
                placeholder="welcome_message.wav"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background font-mono"
              />
            </div>
          </div>
        )}

        {nodeType === 'HangupNode' && (
          <div className="space-y-3">
            <div>
              <label className="font-semibold text-foreground">SIP Clearing Cause</label>
              <input
                type="text"
                value={config.reason || ''}
                onChange={(e) => setConfig({ ...config, reason: e.target.value })}
                placeholder="Normal Clearing"
                className="w-full mt-1 px-3 py-1.5 rounded-lg border border-border bg-background"
              />
            </div>
          </div>
        )}
      </div>

      {/* Footer Actions */}
      <div className="p-4 border-t border-border flex items-center justify-between gap-2">
        <Button
          variant="destructive"
          size="sm"
          onClick={() => onDeleteNode(selectedNode.id)}
          className="gap-1.5"
        >
          <Trash2 className="w-3.5 h-3.5" />
          <span>Delete Node</span>
        </Button>
        <Button size="sm" onClick={handleSave} className="gap-1.5">
          <Save className="w-3.5 h-3.5" />
          <span>Apply Changes</span>
        </Button>
      </div>
    </div>
  );
};
