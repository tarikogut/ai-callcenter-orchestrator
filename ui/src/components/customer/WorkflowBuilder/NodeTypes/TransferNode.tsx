import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { PhoneForwarded } from 'lucide-react';

export const TransferNode: React.FC<NodeProps> = ({ data, selected }) => {
  const config = (data?.config as any) || {};

  return (
    <div
      className={`relative min-w-[240px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-blue-500/80'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-blue-500 !w-3 !h-3 !-top-1.5"
      />

      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-blue-500/10 text-blue-500 flex items-center justify-center font-bold">
          <PhoneForwarded className="w-4 h-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-blue-500">
            Transfer Call
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {(data?.label as string) || 'Transfer to Human'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between items-center font-mono">
          <span>Target:</span>
          <span className="text-foreground font-semibold px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500">
            Ext {config.targetExtension || '101'}
          </span>
        </div>
        <div className="flex justify-between items-center text-[10px]">
          <span>Protocol:</span>
          <span className="text-foreground">SIP uuid_transfer</span>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-blue-500 !w-3 !h-3 !-bottom-1.5"
      />
    </div>
  );
};
