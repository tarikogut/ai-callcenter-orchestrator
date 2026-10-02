import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { PhoneOff } from 'lucide-react';

export const HangupNode: React.FC<NodeProps> = ({ data, selected }) => {
  const config = (data?.config as any) || {};

  return (
    <div
      className={`relative min-w-[220px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-rose-500/80'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-rose-500 !w-3 !h-3 !-top-1.5"
      />

      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-rose-500/10 text-rose-500 flex items-center justify-center font-bold">
          <PhoneOff className="w-4 h-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-rose-500">
            Hangup / BYE
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {(data?.label as string) || 'Terminate Call'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between items-center">
          <span>Reason:</span>
          <span className="text-foreground font-medium">
            {config.reason || 'Normal Clearing (200 OK)'}
          </span>
        </div>
      </div>
      {/* End node: no source handle */}
    </div>
  );
};
