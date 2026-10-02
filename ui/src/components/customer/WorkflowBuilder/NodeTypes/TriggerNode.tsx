import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { PhoneIncoming, MessageSquare } from 'lucide-react';

export const TriggerNode: React.FC<NodeProps> = ({ data, selected }) => {
  const isSms = (data?.config as any)?.triggerType === 'sms';

  return (
    <div
      className={`relative min-w-[240px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-emerald-500/80'
      }`}
    >
      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-500 flex items-center justify-center font-bold">
          {isSms ? <MessageSquare className="w-4 h-4" /> : <PhoneIncoming className="w-4 h-4" />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-emerald-500">
            Trigger
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {(data?.label as string) || 'Inbound Call Trigger'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between font-mono">
          <span>DID Match:</span>
          <span className="text-foreground font-semibold">
            {(data?.config as any)?.did || 'Any Assigned DID'}
          </span>
        </div>
        <div className="flex justify-between">
          <span>Parameters:</span>
          <span className="text-foreground font-mono">$caller, $did</span>
        </div>
      </div>

      {/* Output handle only */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-emerald-500 !w-3 !h-3 !-bottom-1.5"
      />
    </div>
  );
};
