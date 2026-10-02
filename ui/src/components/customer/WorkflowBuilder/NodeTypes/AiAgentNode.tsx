import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { Bot, Sparkles } from 'lucide-react';

export const AiAgentNode: React.FC<NodeProps> = ({ data, selected }) => {
  const config = (data?.config as any) || {};

  return (
    <div
      className={`relative min-w-[260px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-indigo-500/80'
      }`}
    >
      {/* Top Handle (Input) */}
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-indigo-500 !w-3 !h-3 !-top-1.5"
      />

      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-500 flex items-center justify-center font-bold">
          <Bot className="w-4 h-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-wider text-indigo-500">
            <Sparkles className="w-3 h-3" />
            AI Voice Agent
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {config.agentName || (data?.label as string) || 'AI Agent'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between items-center">
          <span>Voice Model:</span>
          <span className="text-foreground font-mono font-medium truncate max-w-[130px]">
            {config.voice || 'gemini-live-aoede'}
          </span>
        </div>
        <div className="flex justify-between items-center">
          <span>Fillers:</span>
          <span className="text-emerald-500 font-semibold">
            {config.fillerPhrases ? `${config.fillerPhrases.length} configured` : 'Enabled'}
          </span>
        </div>
        {config.persona && (
          <p className="text-[10px] text-muted-foreground italic line-clamp-2 mt-1 bg-muted/40 p-1.5 rounded">
            "{config.persona}"
          </p>
        )}
      </div>

      {/* Bottom Handle (Output) */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-indigo-500 !w-3 !h-3 !-bottom-1.5"
      />
    </div>
  );
};
