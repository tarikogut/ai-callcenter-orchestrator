import React from 'react';
import {
  PhoneIncoming,
  Bot,
  Wrench,
  GitBranch,
  PhoneForwarded,
  PhoneOff,
  Volume2,
  Plus,
  HelpCircle,
} from 'lucide-react';

interface PaletteItem {
  type: string;
  label: string;
  category: string;
  icon: React.ReactNode;
  color: string;
  description: string;
  defaultConfig: Record<string, any>;
}

const paletteItems: PaletteItem[] = [
  {
    type: 'InboundCallTrigger',
    label: 'Inbound Call Trigger',
    category: 'Trigger',
    icon: <PhoneIncoming className="w-4 h-4" />,
    color: 'text-emerald-500 border-emerald-500/30 bg-emerald-500/10',
    description: 'Fires when SIP INVITE hits carrier DID',
    defaultConfig: { did: '', triggerType: 'call' },
  },
  {
    type: 'AiVoiceAgentNode',
    label: 'AI Voice Agent',
    category: 'Intelligence',
    icon: <Bot className="w-4 h-4" />,
    color: 'text-indigo-500 border-indigo-500/30 bg-indigo-500/10',
    description: 'Full-duplex Gemini Live / OpenAI real-time agent',
    defaultConfig: {
      agentName: 'Ayşe Asistan',
      voice: 'gemini-live-aoede-warm',
      persona: 'Yardımsever ve profesyonel asistan.',
      fillerPhrases: ['Bir saniye lütfen...', 'Hemen bakıyorum...'],
    },
  },
  {
    type: 'McpToolNode',
    label: 'MCP Tool Action',
    category: 'Integration',
    icon: <Wrench className="w-4 h-4" />,
    color: 'text-amber-500 border-amber-500/30 bg-amber-500/10',
    description: 'Execute external ERP / CRM / database MCP tools',
    defaultConfig: {
      mcpServer: 'https://api.hayateczanesi.com/mcp',
      toolName: 'check_stock',
    },
  },
  {
    type: 'ConditionNode',
    label: 'Condition / Logic',
    category: 'Routing',
    icon: <GitBranch className="w-4 h-4" />,
    color: 'text-purple-500 border-purple-500/30 bg-purple-500/10',
    description: 'Branch based on working hours, sentiment or balance',
    defaultConfig: {
      expression: 'is_open_hours == true',
    },
  },
  {
    type: 'TransferNode',
    label: 'Transfer Call',
    category: 'Routing',
    icon: <PhoneForwarded className="w-4 h-4" />,
    color: 'text-blue-500 border-blue-500/30 bg-blue-500/10',
    description: 'Bridge call to Dahili Extension (101) or Support Queue',
    defaultConfig: {
      targetExtension: '101',
    },
  },
  {
    type: 'PlayAudioNode',
    label: 'Play Audio WAV',
    category: 'Media',
    icon: <Volume2 className="w-4 h-4" />,
    color: 'text-teal-500 border-teal-500/30 bg-teal-500/10',
    description: 'Stream pre-recorded greeting or music on hold',
    defaultConfig: {
      audioFile: 'welcome_greeting.wav',
    },
  },
  {
    type: 'HangupNode',
    label: 'Hangup Call',
    category: 'Lifecycle',
    icon: <PhoneOff className="w-4 h-4" />,
    color: 'text-rose-500 border-rose-500/30 bg-rose-500/10',
    description: 'Terminate call gracefully and trigger Diameter Ro Terminate',
    defaultConfig: {
      reason: 'Call Completed (200 OK)',
    },
  },
];

interface SidebarPaletteProps {
  onAddNode: (type: string, label: string, config: Record<string, any>) => void;
}

export const SidebarPalette: React.FC<SidebarPaletteProps> = ({ onAddNode }) => {
  const onDragStart = (event: React.DragEvent, nodeType: string, label: string, config: Record<string, any>) => {
    event.dataTransfer.setData('application/reactflow/type', nodeType);
    event.dataTransfer.setData('application/reactflow/label', label);
    event.dataTransfer.setData('application/reactflow/config', JSON.stringify(config));
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <div className="w-72 border-r border-border bg-card/40 flex flex-col shrink-0 select-none">
      <div className="p-3 border-b border-border/50">
        <h3 className="text-xs font-bold uppercase tracking-wider text-foreground">
          Node Palette (Sürükle & Bırak)
        </h3>
        <p className="text-[11px] text-muted-foreground mt-0.5">
          Drag nodes into canvas or click + to append.
        </p>
      </div>

      <div className="p-3 overflow-y-auto space-y-2 flex-1">
        {paletteItems.map((item) => (
          <div
            key={item.type}
            draggable
            onDragStart={(e) => onDragStart(e, item.type, item.label, item.defaultConfig)}
            onClick={() => onAddNode(item.type, item.label, item.defaultConfig)}
            className="group relative p-2.5 rounded-xl border border-border bg-card hover:border-primary/50 hover:shadow-md cursor-grab active:cursor-grabbing transition-all flex items-start gap-2.5"
          >
            <div className={`p-2 rounded-lg border shrink-0 ${item.color}`}>
              {item.icon}
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-foreground group-hover:text-primary transition-colors">
                  {item.label}
                </span>
                <button
                  type="button"
                  className="opacity-0 group-hover:opacity-100 p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-opacity"
                  title="Add to canvas"
                >
                  <Plus className="w-3.5 h-3.5" />
                </button>
              </div>
              <p className="text-[10px] text-muted-foreground leading-snug mt-0.5">
                {item.description}
              </p>
            </div>
          </div>
        ))}
      </div>

      <div className="p-3 border-t border-border/50 bg-secondary/30 text-[11px] text-muted-foreground">
        <div className="flex items-center gap-1.5 font-medium text-foreground">
          <HelpCircle className="w-3.5 h-3.5 text-primary" />
          <span>Execution Order</span>
        </div>
        <p className="text-[10px] mt-1 leading-normal">
          Inbound calls flow from Top to Bottom. Connect true/false decision handles for dynamic routing.
        </p>
      </div>
    </div>
  );
};
