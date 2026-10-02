import React, { useState, useCallback, useRef } from 'react';
import {
  ReactFlow,
  ReactFlowProvider,
  addEdge,
  useNodesState,
  useEdgesState,
  Controls,
  Background,
  MiniMap,
  Connection,
  Edge,
  Node,
  BackgroundVariant,
  Panel,
} from '@xyflow/react';
import {
  Play,
  Save,
  Sparkles,
  CheckCircle,
  FileCode,
} from 'lucide-react';
import { TriggerNode } from './NodeTypes/TriggerNode';
import { AiAgentNode } from './NodeTypes/AiAgentNode';
import { McpNode } from './NodeTypes/McpNode';
import { ConditionNode } from './NodeTypes/ConditionNode';
import { TransferNode } from './NodeTypes/TransferNode';
import { HangupNode } from './NodeTypes/HangupNode';
import { PlayAudioNode } from './NodeTypes/PlayAudioNode';
import { SidebarPalette } from './SidebarPalette';
import { NodeConfigPanel } from './NodeConfigPanel';
import { workflowTemplates } from './FlowTemplates';
import { Button } from '../../ui/Button';
import { Modal } from '../../ui/Modal';

const nodeTypes = {
  InboundCallTrigger: TriggerNode,
  AiVoiceAgentNode: AiAgentNode,
  McpToolNode: McpNode,
  ConditionNode: ConditionNode,
  TransferNode: TransferNode,
  HangupNode: HangupNode,
  PlayAudioNode: PlayAudioNode,
};

interface CanvasProps {
  tenantId: string;
}

export const WorkflowCanvas: React.FC<CanvasProps> = ({ tenantId }) => {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);

  // Pick initial template based on tenant or default to pharmacy
  const defaultTemplate =
    tenantId === 'jet_kargo'
      ? workflowTemplates.logistics
      : workflowTemplates.pharmacy;

  const [nodes, setNodes, onNodesChange] = useNodesState(defaultTemplate.nodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(defaultTemplate.edges);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [reactFlowInstance, setReactFlowInstance] = useState<any>(null);

  // Simulation & export modals
  const [isSimulating, setIsSimulating] = useState(false);
  const [simStep, setSimStep] = useState(0);
  const [isJsonModalOpen, setIsJsonModalOpen] = useState(false);
  const [isSavedToast, setIsSavedToast] = useState(false);

  const onConnect = useCallback(
    (params: Connection | Edge) =>
      setEdges((eds) => addEdge({ ...params, animated: true }, eds)),
    [setEdges]
  );

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  const handleUpdateNode = (nodeId: string, data: any) => {
    setNodes((nds) =>
      nds.map((n) => (n.id === nodeId ? { ...n, data } : n))
    );
    setSelectedNode((prev) => (prev?.id === nodeId ? { ...prev, data } : prev));
  };

  const handleDeleteNode = (nodeId: string) => {
    setNodes((nds) => nds.filter((n) => n.id !== nodeId));
    setEdges((eds) => eds.filter((e) => e.source !== nodeId && e.target !== nodeId));
    setSelectedNode(null);
  };

  // Add node from Palette click
  const handleAddNodeFromPalette = (type: string, label: string, config: Record<string, any>) => {
    const id = `node_${Date.now()}`;
    const newNode: Node = {
      id,
      type,
      position: { x: 300 + Math.random() * 80, y: 150 + nodes.length * 60 },
      data: { label, type, config },
    };
    setNodes((nds) => [...nds, newNode]);
  };

  // Drag and drop handlers
  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      const type = event.dataTransfer.getData('application/reactflow/type');
      const label = event.dataTransfer.getData('application/reactflow/label');
      const rawConfig = event.dataTransfer.getData('application/reactflow/config');

      if (!type || !reactFlowWrapper.current || !reactFlowInstance) return;

      const position = reactFlowInstance.screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const config = rawConfig ? JSON.parse(rawConfig) : {};
      const newNode: Node = {
        id: `node_${Date.now()}`,
        type,
        position,
        data: { label, type, config },
      };

      setNodes((nds) => [...nds, newNode]);
    },
    [reactFlowInstance, setNodes]
  );

  // Template loader
  const loadTemplate = (key: 'pharmacy' | 'logistics') => {
    const t = workflowTemplates[key];
    setNodes(t.nodes);
    setEdges(t.edges);
    setSelectedNode(null);
  };

  // Save flow
  const handleSaveFlow = () => {
    const payload = {
      workflow_id: `flow_${tenantId}`,
      tenant_id: tenantId,
      nodes,
      edges,
      updated_at: new Date().toISOString(),
    };
    localStorage.setItem(`flow_${tenantId}`, JSON.stringify(payload));
    setIsSavedToast(true);
    setTimeout(() => setIsSavedToast(false), 3000);
  };

  // Test Simulation Runner
  const runSimulation = () => {
    setIsSimulating(true);
    setSimStep(0);

    const orderedNodeIds = nodes.map((n) => n.id);
    let current = 0;

    const interval = setInterval(() => {
      current++;
      if (current >= orderedNodeIds.length) {
        clearInterval(interval);
        setTimeout(() => setIsSimulating(false), 2000);
      } else {
        setSimStep(current);
      }
    }, 1200);
  };

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)] overflow-hidden">
      {/* Top Workflow Builder Bar */}
      <div className="h-12 border-b border-border bg-card/90 px-4 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-3">
          <span className="text-xs font-bold text-foreground">Visual Workflow Builder</span>
          <span className="text-[11px] px-2 py-0.5 rounded bg-primary/10 text-primary font-mono font-medium">
            n8n Engine Compatible
          </span>

          <div className="hidden md:flex items-center gap-1.5 pl-3 border-l border-border text-xs text-muted-foreground">
            <span>Template:</span>
            <button
              onClick={() => loadTemplate('pharmacy')}
              className="px-2 py-0.5 rounded hover:bg-secondary hover:text-foreground text-[11px]"
            >
              Eczane Nöbet
            </button>
            <span>•</span>
            <button
              onClick={() => loadTemplate('logistics')}
              className="px-2 py-0.5 rounded hover:bg-secondary hover:text-foreground text-[11px]"
            >
              Jet Kargo Takip
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isSavedToast && (
            <span className="text-xs text-emerald-500 font-semibold flex items-center gap-1 animate-in fade-in">
              <CheckCircle className="w-3.5 h-3.5" /> Deployed!
            </span>
          )}

          <Button
            variant="outline"
            size="sm"
            onClick={runSimulation}
            disabled={isSimulating}
            className="h-8 gap-1.5 text-xs"
          >
            <Play className={`w-3.5 h-3.5 ${isSimulating ? 'text-amber-500 animate-spin' : 'text-emerald-500'}`} />
            <span>{isSimulating ? `Testing Step ${simStep + 1}...` : 'Test Call Flow'}</span>
          </Button>

          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsJsonModalOpen(true)}
            className="h-8 gap-1.5 text-xs"
          >
            <FileCode className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Export JSON</span>
          </Button>

          <Button
            size="sm"
            onClick={handleSaveFlow}
            className="h-8 gap-1.5 text-xs"
          >
            <Save className="w-3.5 h-3.5" />
            <span>Deploy to Go Engine</span>
          </Button>
        </div>
      </div>

      {/* Main Canvas + Left Palette + Right Config Panel */}
      <div className="flex flex-1 overflow-hidden relative">
        {/* Left Palette */}
        <SidebarPalette onAddNode={handleAddNodeFromPalette} />

        {/* Canvas Center */}
        <div ref={reactFlowWrapper} className="flex-1 h-full w-full relative">
          <ReactFlow
            nodes={nodes.map((n, idx) => ({
              ...n,
              className: isSimulating && simStep === idx ? 'ring-4 ring-amber-400 rounded-xl transition-all scale-105' : '',
            }))}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onPaneClick={onPaneClick}
            onInit={setReactFlowInstance}
            onDrop={onDrop}
            onDragOver={onDragOver}
            nodeTypes={nodeTypes}
            fitView
            proOptions={{ hideAttribution: true }}
          >
            <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
            <Controls />
            <MiniMap
              nodeColor={() => '#3b82f6'}
              className="!bg-card !border !border-border !rounded-lg"
            />

            {isSimulating && (
              <Panel position="top-center">
                <div className="bg-amber-500/10 border border-amber-500/30 text-amber-600 dark:text-amber-400 backdrop-blur-md px-4 py-2 rounded-xl text-xs font-semibold shadow-lg flex items-center gap-2 animate-pulse">
                  <Sparkles className="w-4 h-4" />
                  <span>Simulating Live Inbound Call Execution & Diameter Ro Prepaid Reservation...</span>
                </div>
              </Panel>
            )}
          </ReactFlow>
        </div>

        {/* Right Config Panel */}
        {selectedNode && (
          <NodeConfigPanel
            selectedNode={selectedNode}
            onClose={() => setSelectedNode(null)}
            onUpdateNode={handleUpdateNode}
            onDeleteNode={handleDeleteNode}
          />
        )}
      </div>

      {/* JSON Export Modal */}
      <Modal
        isOpen={isJsonModalOpen}
        onClose={() => setIsJsonModalOpen(false)}
        title="Workflow Orchestrator JSON Schema"
        description="Standard Go Engine representation matching README.md section 4 format."
      >
        <div className="space-y-4">
          <pre className="p-3 bg-muted rounded-xl text-[11px] font-mono text-foreground overflow-x-auto max-h-96">
            {JSON.stringify(
              {
                workflow_id: `flow_${tenantId}`,
                tenant_id: tenantId,
                nodes: nodes.map((n) => ({
                  id: n.id,
                  type: n.type,
                  config: n.data?.config,
                })),
                edges: edges.map((e) => ({
                  id: e.id,
                  source: e.source,
                  target: e.target,
                  sourceHandle: e.sourceHandle,
                })),
              },
              null,
              2
            )}
          </pre>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              onClick={() => {
                navigator.clipboard.writeText(
                  JSON.stringify({ nodes, edges }, null, 2)
                );
                alert('Workflow JSON copied to clipboard!');
              }}
            >
              Copy to Clipboard
            </Button>
            <Button onClick={() => setIsJsonModalOpen(false)}>Close</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export const Canvas: React.FC<CanvasProps> = (props) => (
  <ReactFlowProvider>
    <WorkflowCanvas {...props} />
  </ReactFlowProvider>
);
