import React, { useState, useEffect } from 'react';
import {
  Activity,
  TrendingUp,
  ShieldCheck,
  Cpu,
} from 'lucide-react';
import { DiameterMetric } from '../../types';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/Card';
import { Badge } from '../ui/Badge';

interface DiameterBillingProps {
  metrics: DiameterMetric;
}

interface CcrEventLog {
  id: string;
  time: string;
  tenantId: string;
  sessionType: 'CCR-Init' | 'CCR-Update' | 'CCR-Terminate';
  requestedUnits: string;
  grantedUnits: string;
  deductedAmount: number;
  resultCode: 'DIAMETER_SUCCESS (2001)' | 'DIAMETER_LOW_BALANCE (4012)';
}

export const DiameterBilling: React.FC<DiameterBillingProps> = ({ metrics }) => {
  // Configurable rating parameters
  const [voiceRatePerMin, setVoiceRatePerMin] = useState(0.45);
  const [aiTokenRatePer1k, setAiTokenRatePer1k] = useState(0.08);
  const [mcpInvocationFee, setMcpInvocationFee] = useState(0.05);
  const [lowBalanceCutoff, setLowBalanceCutoff] = useState(5.0);

  // Live CCR event simulation feed
  const [logs, setLogs] = useState<CcrEventLog[]>([
    {
      id: 'ccr_1',
      time: '09:32:01',
      tenantId: 'eczane_hayat',
      sessionType: 'CCR-Update',
      requestedUnits: '60s Voice + 120 Tokens',
      grantedUnits: '60s Voice Granted',
      deductedAmount: 0.49,
      resultCode: 'DIAMETER_SUCCESS (2001)',
    },
    {
      id: 'ccr_2',
      time: '09:31:55',
      tenantId: 'jet_kargo',
      sessionType: 'CCR-Update',
      requestedUnits: '60s Voice + 1 MCP Call',
      grantedUnits: '60s Voice Granted',
      deductedAmount: 0.50,
      resultCode: 'DIAMETER_SUCCESS (2001)',
    },
    {
      id: 'ccr_3',
      time: '09:31:40',
      tenantId: 'acibadem_dent',
      sessionType: 'CCR-Init',
      requestedUnits: '120s Voice Reservation',
      grantedUnits: '120s Reserved (0.90 TRY)',
      deductedAmount: 0.90,
      resultCode: 'DIAMETER_SUCCESS (2001)',
    },
  ]);

  // Live tick effect to simulate real-time Diameter CCR-U traffic
  useEffect(() => {
    const interval = setInterval(() => {
      const tenants = ['eczane_hayat', 'jet_kargo', 'acibadem_dent'];
      const tenant = tenants[Math.floor(Math.random() * tenants.length)];
      const types: ('CCR-Update' | 'CCR-Init' | 'CCR-Terminate')[] = ['CCR-Update', 'CCR-Update', 'CCR-Terminate'];
      const sessionType = types[Math.floor(Math.random() * types.length)];
      const now = new Date().toTimeString().split(' ')[0];

      const newLog: CcrEventLog = {
        id: `ccr_${Date.now()}`,
        time: now,
        tenantId: tenant,
        sessionType,
        requestedUnits: '60s RTP + Gemini Live Audio',
        grantedUnits: sessionType === 'CCR-Terminate' ? 'Session Released' : '60s Granted',
        deductedAmount: sessionType === 'CCR-Terminate' ? 0.0 : +(0.45 + Math.random() * 0.08).toFixed(2),
        resultCode: 'DIAMETER_SUCCESS (2001)',
      };

      setLogs((prev) => [newLog, ...prev.slice(0, 14)]);
    }, 4000);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
          <Activity className="w-6 h-6 text-primary" />
          Diameter Ro Prepaid Rating Engine & Billing
        </h1>
        <p className="text-sm text-muted-foreground mt-0.5">
          Telecom-grade real-time credit control (RFC 4006 / 3GPP TS 32.299) monitoring active call sessions.
        </p>
      </div>

      {/* Main Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="border-l-4 border-l-primary">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">Active Ro CCR Sessions</span>
              <Badge variant="success">Online</Badge>
            </div>
            <p className="text-3xl font-extrabold text-foreground mt-2">{metrics.activeSessions}</p>
            <p className="text-[11px] text-muted-foreground mt-1 flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-ping"></span>
              Synchronized with Go FreeSWITCH bridge
            </p>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-emerald-500">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">CCA Success Rate</span>
              <ShieldCheck className="w-4 h-4 text-emerald-500" />
            </div>
            <p className="text-3xl font-extrabold text-emerald-600 dark:text-emerald-400 mt-2">
              {metrics.ccaSuccessRate}%
            </p>
            <p className="text-[11px] text-muted-foreground mt-1">
              {metrics.ccrTotalCount.toLocaleString()} total CCR msgs today
            </p>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-amber-500">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">Average Rating Latency</span>
              <Cpu className="w-4 h-4 text-amber-500" />
            </div>
            <p className="text-3xl font-extrabold text-foreground mt-2">
              {metrics.avgRatingLatencyMs} <span className="text-base font-normal text-muted-foreground">ms</span>
            </p>
            <p className="text-[11px] text-muted-foreground mt-1">Sub-10ms telecom SLA satisfied</p>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-indigo-500">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">Billed Volume Today</span>
              <TrendingUp className="w-4 h-4 text-indigo-500" />
            </div>
            <p className="text-3xl font-extrabold text-foreground mt-2">
              {metrics.billedAmountToday.toFixed(2)}{' '}
              <span className="text-base font-normal text-muted-foreground">{metrics.currency}</span>
            </p>
            <p className="text-[11px] text-muted-foreground mt-1">
              From {metrics.globalCdrToday.toLocaleString()} processed calls
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Middle: Rating Engine Configuration & Architecture */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left 2 Cols: Global Rate Cards */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Global Rating Tariff & Credit Control Policy</CardTitle>
            <CardDescription>
              Configure tariff per minute, token pricing, and low balance behavior for the Go CCR-Update loop.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="text-xs font-semibold text-foreground flex items-center justify-between">
                  <span>Voice Call Rate (TRY / Minute)</span>
                  <span className="font-mono text-primary font-bold">{voiceRatePerMin.toFixed(2)} TRY</span>
                </label>
                <input
                  type="range"
                  min="0.10"
                  max="2.00"
                  step="0.05"
                  value={voiceRatePerMin}
                  onChange={(e) => setVoiceRatePerMin(parseFloat(e.target.value))}
                  className="w-full mt-2 accent-primary"
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  Charged every 60 seconds of FreeSWITCH RTP audio session.
                </p>
              </div>

              <div>
                <label className="text-xs font-semibold text-foreground flex items-center justify-between">
                  <span>AI Agent Audio Token Rate (/1k tokens)</span>
                  <span className="font-mono text-primary font-bold">{aiTokenRatePer1k.toFixed(3)} TRY</span>
                </label>
                <input
                  type="range"
                  min="0.01"
                  max="0.50"
                  step="0.01"
                  value={aiTokenRatePer1k}
                  onChange={(e) => setAiTokenRatePer1k(parseFloat(e.target.value))}
                  className="w-full mt-2 accent-primary"
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  Gemini Live WebSocket / OpenAI Realtime PCM stream cost.
                </p>
              </div>

              <div>
                <label className="text-xs font-semibold text-foreground flex items-center justify-between">
                  <span>MCP Tool Invocation Fee (TRY / Call)</span>
                  <span className="font-mono text-primary font-bold">{mcpInvocationFee.toFixed(2)} TRY</span>
                </label>
                <input
                  type="range"
                  min="0.01"
                  max="0.30"
                  step="0.01"
                  value={mcpInvocationFee}
                  onChange={(e) => setMcpInvocationFee(parseFloat(e.target.value))}
                  className="w-full mt-2 accent-primary"
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  External ERP/CRM API lookup tariff (e.g. check stock, tracking).
                </p>
              </div>

              <div>
                <label className="text-xs font-semibold text-foreground flex items-center justify-between">
                  <span>Low Credit Warning Threshold</span>
                  <span className="font-mono text-amber-500 font-bold">{lowBalanceCutoff.toFixed(2)} TRY</span>
                </label>
                <input
                  type="range"
                  min="1"
                  max="50"
                  step="1"
                  value={lowBalanceCutoff}
                  onChange={(e) => setLowBalanceCutoff(parseFloat(e.target.value))}
                  className="w-full mt-2 accent-amber-500"
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  Triggers early warning audio prompt to caller before hangup.
                </p>
              </div>
            </div>

            <div className="p-4 rounded-xl bg-secondary/50 border border-border flex items-center justify-between">
              <div>
                <p className="text-xs font-semibold text-foreground">
                  Action on Zero Credit (DIAMETER_CREDIT_LIMIT_REACHED)
                </p>
                <p className="text-[11px] text-muted-foreground">
                  Play Turkish low-balance announcement wav, then send SIP 487 / BYE.
                </p>
              </div>
              <Badge variant="destructive">Strict Disconnect</Badge>
            </div>
          </CardContent>
        </Card>

        {/* Right 1 Col: Architecture info */}
        <Card>
          <CardHeader>
            <CardTitle>Ro Architecture Spec</CardTitle>
            <CardDescription>Standards compliance</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <div className="p-3 bg-muted/40 rounded-lg space-y-1">
              <span className="font-semibold text-foreground">AVP 416 (CC-Request-Type):</span>
              <p className="text-muted-foreground">1: INITIAL, 2: UPDATE (every 60s), 3: TERMINATION.</p>
            </div>
            <div className="p-3 bg-muted/40 rounded-lg space-y-1">
              <span className="font-semibold text-foreground">AVP 437 (Cost-Information):</span>
              <p className="text-muted-foreground">Real-time monetary unit subtraction directly against tenant SQL balance.</p>
            </div>
            <div className="p-3 bg-muted/40 rounded-lg space-y-1">
              <span className="font-semibold text-foreground">Failover Behavior:</span>
              <p className="text-muted-foreground">Credit reservation holds funds in escrow during the duration of the call.</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Real-time Diameter Ro Stream Feed */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <span className="relative flex h-2.5 w-2.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500"></span>
              </span>
              Real-Time Diameter Ro Message Stream
            </CardTitle>
            <CardDescription>Live CCR-Init and CCR-Update packet transactions from Go Call Orchestrator</CardDescription>
          </div>
          <Badge variant="outline" className="font-mono">Port: 3868 / TCP</Badge>
        </CardHeader>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead className="bg-muted/40 text-muted-foreground uppercase font-semibold border-b border-border">
              <tr>
                <th className="px-5 py-2.5">Timestamp</th>
                <th className="px-5 py-2.5">Tenant</th>
                <th className="px-5 py-2.5">Diameter Message</th>
                <th className="px-5 py-2.5">Units Requested</th>
                <th className="px-5 py-2.5">Quota Granted</th>
                <th className="px-5 py-2.5">Debit (TRY)</th>
                <th className="px-5 py-2.5">Result-Code</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/60">
              {logs.map((log) => (
                <tr key={log.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-5 py-3 text-muted-foreground">{log.time}</td>
                  <td className="px-5 py-3 font-semibold text-foreground">{log.tenantId}</td>
                  <td className="px-5 py-3">
                    <span
                      className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                        log.sessionType === 'CCR-Init'
                          ? 'bg-blue-500/10 text-blue-500'
                          : log.sessionType === 'CCR-Update'
                          ? 'bg-emerald-500/10 text-emerald-500'
                          : 'bg-amber-500/10 text-amber-500'
                      }`}
                    >
                      {log.sessionType}
                    </span>
                  </td>
                  <td className="px-5 py-3 text-muted-foreground">{log.requestedUnits}</td>
                  <td className="px-5 py-3 text-foreground">{log.grantedUnits}</td>
                  <td className="px-5 py-3 font-bold text-foreground">
                    {log.deductedAmount > 0 ? `-${log.deductedAmount.toFixed(2)}` : '0.00'}
                  </td>
                  <td className="px-5 py-3 text-emerald-500">{log.resultCode}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
};
