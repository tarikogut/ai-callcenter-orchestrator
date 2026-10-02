import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { Wrench } from 'lucide-react';

export const McpNode: React.FC<NodeProps> = ({ data, selected }) => {
  const config = (data?.config as any) || {};

  return (
    <div
      className={`relative min-w-[250px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-amber-500/80'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-amber-500 !w-3 !h-3 !-top-1.5"
      />

      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-amber-500/10 text-amber-500 flex items-center justify-center font-bold">
          <Wrench className="w-4 h-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-amber-500">
            MCP Tool Call
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {config.toolName || (data?.label as string) || 'External Tool'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between items-center">
          <span>Server:</span>
          <span className="font-mono text-foreground text-[10px] truncate max-w-[140px]">
            {config.mcpServer || 'https://api.domain.com/mcp'}
          </span>
        </div>
        <div className="flex justify-between items-center">
          <span>Tool:</span>
          <span className="font-mono text-primary font-semibold">
            {config.toolName || 'execute'}
          </span>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-amber-500 !w-3 !h-3 !-bottom-1.5"
      />
    </div>
  );
};
