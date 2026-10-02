import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { GitBranch, Check, X } from 'lucide-react';

export const ConditionNode: React.FC<NodeProps> = ({ data, selected }) => {
  const config = (data?.config as any) || {};

  return (
    <div
      className={`relative min-w-[250px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-purple-500/80'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-purple-500 !w-3 !h-3 !-top-1.5"
      />

      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-purple-500/10 text-purple-500 flex items-center justify-center font-bold">
          <GitBranch className="w-4 h-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-purple-500">
            Decision / Condition
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {(data?.label as string) || 'Check Condition'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between items-center font-mono">
          <span>Expression:</span>
          <span className="text-foreground font-semibold truncate max-w-[150px]">
            {config.expression || 'is_open_hours == true'}
          </span>
        </div>
      </div>

      {/* Two outputs: Left = True, Right = False */}
      <div className="mt-3 flex justify-between text-[10px] font-bold px-2 pt-1 border-t border-border/40">
        <div className="flex items-center gap-1 text-emerald-500">
          <Check className="w-3 h-3" />
          <span>TRUE</span>
        </div>
        <div className="flex items-center gap-1 text-rose-500">
          <span>FALSE</span>
          <X className="w-3 h-3" />
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="true"
        className="!bg-emerald-500 !w-3 !h-3 !-bottom-1.5 !left-[25%]"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="false"
        className="!bg-rose-500 !w-3 !h-3 !-bottom-1.5 !left-[75%]"
      />
    </div>
  );
};
