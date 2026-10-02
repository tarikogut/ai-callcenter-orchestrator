import React from 'react';
import { Handle, Position, NodeProps } from '@xyflow/react';
import { Volume2 } from 'lucide-react';

export const PlayAudioNode: React.FC<NodeProps> = ({ data, selected }) => {
  const config = (data?.config as any) || {};

  return (
    <div
      className={`relative min-w-[240px] rounded-xl border-2 bg-card p-3 shadow-lg transition-all ${
        selected ? 'border-primary ring-2 ring-primary/30 shadow-primary/20' : 'border-teal-500/80'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-teal-500 !w-3 !h-3 !-top-1.5"
      />

      <div className="flex items-center gap-2.5 pb-2 border-b border-border/60">
        <div className="w-8 h-8 rounded-lg bg-teal-500/10 text-teal-500 flex items-center justify-center font-bold">
          <Volume2 className="w-4 h-4" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-wider text-teal-500">
            Audio Announcement
          </div>
          <div className="text-xs font-bold text-foreground truncate">
            {(data?.label as string) || 'Play Announcement'}
          </div>
        </div>
      </div>

      <div className="mt-2 text-[11px] text-muted-foreground space-y-1">
        <div className="flex justify-between items-center font-mono">
          <span>Audio File:</span>
          <span className="text-foreground text-[10px] truncate max-w-[140px]">
            {config.audioFile || 'welcome_greeting.wav'}
          </span>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-teal-500 !w-3 !h-3 !-bottom-1.5"
      />
    </div>
  );
};
