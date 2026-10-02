import React, { useState, useEffect } from 'react';
import {
  Phone,
  PhoneOff,
  PhoneForwarded,
  Mic,
  MicOff,
  Radio,
  User,
  Bot,
  Sparkles,
  Clock,
  Wrench,
} from 'lucide-react';
import { LiveTranscriptEntry, Extension } from '../../types';
import { sampleLiveTranscript } from '../../api/mockData';
import { Button } from '../ui/Button';
import { Badge } from '../ui/Badge';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/Card';

interface WebRtcSoftphoneProps {
  activeExtension?: Extension;
  tenantName: string;
}

export const WebRtcSoftphone: React.FC<WebRtcSoftphoneProps> = ({
  activeExtension,
  tenantName,
}) => {
  // Softphone Call State
  const [callState, setCallState] = useState<'idle' | 'ringing' | 'connected'>('connected');
  const [dialNumber, setDialNumber] = useState('+90 532 987 6543');
  const [callDuration, setCallDuration] = useState(42);
  const [isMuted, setIsMuted] = useState(false);
  const [isAiControlling, setIsAiControlling] = useState(true);

  // Live Transcript Stream
  const [transcript, setTranscript] = useState<LiveTranscriptEntry[]>(sampleLiveTranscript);

  // Live timer for connected call
  useEffect(() => {
    let interval: any;
    if (callState === 'connected') {
      interval = setInterval(() => {
        setCallDuration((prev) => prev + 1);
      }, 1000);
    }
    return () => clearInterval(interval);
  }, [callState]);

  const formatDuration = (sec: number) => {
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  };

  const handleDialPress = (num: string) => {
    setDialNumber((prev) => prev + num);
  };

  const handleStartCall = () => {
    setCallState('ringing');
    setTimeout(() => {
      setCallState('connected');
      setCallDuration(0);
      setIsAiControlling(true);
    }, 1500);
  };

  const handleHangup = () => {
    setCallState('idle');
    setCallDuration(0);
  };

  const handleTakeOver = () => {
    setIsAiControlling(false);
    const now = new Date().toTimeString().split(' ')[0];
    setTranscript((prev) => [
      ...prev,
      {
        id: `tr_${Date.now()}`,
        timestamp: now,
        sender: 'system',
        text: `Temsilci (${activeExtension?.name || 'Dahili 101'}) görüşmeyi AI'dan devraldı. FreeSWITCH mod_audio_fork durduruldu.`,
      },
      {
        id: `tr_${Date.now() + 1}`,
        timestamp: now,
        sender: 'agent',
        text: 'İyi günler, ben yetkili temsilci. Size yardımcı olmaya devam ediyorum efendim.',
      },
    ]);
  };

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
          <Radio className="w-6 h-6 text-primary" />
          WebRTC Softphone & Live AI Call Monitor
        </h1>
        <p className="text-sm text-muted-foreground mt-0.5">
          In-browser SIP dialer with real-time speech transcription, MCP execution logs, and human takeover bridge.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column (5 Cols): Softphone Dialer */}
        <div className="lg:col-span-5 space-y-4">
          <Card className="border-2 border-primary/20 shadow-xl overflow-hidden">
            {/* Softphone Top Header */}
            <div className="p-4 bg-gradient-to-r from-slate-900 to-indigo-950 text-white flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse"></span>
                <div>
                  <div className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                    WebRTC SIP Register
                  </div>
                  <div className="text-sm font-bold">
                    Ext {activeExtension?.extensionNumber || '101'} • {tenantName}
                  </div>
                </div>
              </div>
              <Badge variant="outline" className="text-emerald-400 border-emerald-400/40 text-[10px]">
                WSS: 200 OK
              </Badge>
            </div>

            <CardContent className="p-5 space-y-5">
              {/* Call Status Display */}
              <div className="p-4 rounded-2xl bg-secondary/70 border border-border text-center space-y-1">
                {callState === 'connected' ? (
                  <>
                    <div className="flex items-center justify-center gap-2 text-emerald-500 font-semibold text-xs">
                      <span className="w-2 h-2 rounded-full bg-emerald-500 animate-ping"></span>
                      <span>{isAiControlling ? 'AI Agent Görüşmede (Live)' : 'Canlı Temsilci devraldı'}</span>
                    </div>
                    <div className="text-lg font-mono font-bold text-foreground">{dialNumber}</div>
                    <div className="text-xs font-mono text-muted-foreground flex items-center justify-center gap-1">
                      <Clock className="w-3 h-3" />
                      <span>{formatDuration(callDuration)}</span>
                      <span>• Opus 48kHz HD Audio</span>
                    </div>
                  </>
                ) : callState === 'ringing' ? (
                  <div className="py-2">
                    <p className="text-xs text-amber-500 font-semibold animate-pulse">Çalıyor (SIP INVITE Gönderildi)...</p>
                    <p className="text-sm font-mono font-bold text-foreground mt-1">{dialNumber}</p>
                  </div>
                ) : (
                  <div className="py-2">
                    <p className="text-xs text-muted-foreground">Hazır (Boşta)</p>
                    <input
                      type="text"
                      value={dialNumber}
                      onChange={(e) => setDialNumber(e.target.value)}
                      placeholder="Numara veya Dahili girin..."
                      className="w-full text-center font-mono font-bold text-lg bg-transparent text-foreground focus:outline-none mt-1"
                    />
                  </div>
                )}
              </div>

              {/* Dialpad Numbers */}
              <div className="grid grid-cols-3 gap-2.5 select-none">
                {['1', '2', '3', '4', '5', '6', '7', '8', '9', '*', '0', '#'].map((digit) => (
                  <button
                    key={digit}
                    onClick={() => handleDialPress(digit)}
                    disabled={callState === 'connected' && false}
                    className="h-12 rounded-xl bg-card border border-border/80 hover:bg-secondary hover:border-primary/50 text-foreground font-semibold text-base flex flex-col items-center justify-center active:scale-95 transition-all shadow-xs"
                  >
                    <span>{digit}</span>
                  </button>
                ))}
              </div>

              {/* Call Control Actions */}
              <div className="pt-2 flex items-center justify-center gap-3">
                {callState === 'connected' ? (
                  <>
                    <Button
                      variant={isMuted ? 'destructive' : 'outline'}
                      size="icon"
                      onClick={() => setIsMuted(!isMuted)}
                      className="w-12 h-12 rounded-full"
                    >
                      {isMuted ? <MicOff className="w-5 h-5" /> : <Mic className="w-5 h-5" />}
                    </Button>

                    <Button
                      variant="destructive"
                      size="icon"
                      onClick={handleHangup}
                      className="w-14 h-14 rounded-full shadow-lg shadow-rose-500/25 animate-in zoom-in"
                    >
                      <PhoneOff className="w-6 h-6" />
                    </Button>

                    <Button
                      variant="outline"
                      size="icon"
                      onClick={() => alert('Dahiliye Aktar: Çağrı FreeSWITCH üzerinden dahili 102ye transfer edildi.')}
                      className="w-12 h-12 rounded-full"
                      title="Transfer"
                    >
                      <PhoneForwarded className="w-5 h-5" />
                    </Button>
                  </>
                ) : (
                  <>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setDialNumber('')}
                      className="text-xs h-10 px-3"
                    >
                      Temizle
                    </Button>
                    <Button
                      variant="success"
                      onClick={handleStartCall}
                      className="w-14 h-14 rounded-full shadow-lg shadow-emerald-500/25 flex items-center justify-center"
                    >
                      <Phone className="w-6 h-6" />
                    </Button>
                  </>
                )}
              </div>

              {/* Human Takeover Call Button */}
              {callState === 'connected' && isAiControlling && (
                <div className="pt-2">
                  <Button
                    onClick={handleTakeOver}
                    className="w-full bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-600 hover:to-orange-600 text-white font-bold py-2.5 shadow-md flex items-center justify-center gap-2"
                  >
                    <User className="w-4 h-4" />
                    <span>Çağrıyı AI'dan Devral (Take Over)</span>
                  </Button>
                  <p className="text-[10px] text-center text-muted-foreground mt-1">
                    AI konuşmayı keser, FreeSWITCH ses köprüsünü doğrudan bu softphone kulaklığına bağlar.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Right Column (7 Cols): Live AI Transcription & Telemetry */}
        <div className="lg:col-span-7 space-y-4">
          <Card className="h-full flex flex-col">
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Sparkles className="w-4 h-4 text-primary animate-pulse" />
                  Live AI Audio Transcription & Telemetry
                </CardTitle>
                <CardDescription>
                  Gelen çağrıda arayan ile yapay zeka arasındaki canlı konuşma akışı (Gemini Live / mod_audio_fork)
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="font-mono text-[10px]">
                  RTT Latency: 180ms
                </Badge>
                <Badge variant="success" className="text-[10px]">
                  VAD Aktif
                </Badge>
              </div>
            </CardHeader>

            {/* Conversation Log Stream */}
            <CardContent className="p-4 flex-1 flex flex-col overflow-hidden">
              <div className="flex-1 overflow-y-auto space-y-3 pr-1 max-h-[460px]">
                {transcript.map((item) => (
                  <div
                    key={item.id}
                    className={`flex flex-col ${
                      item.sender === 'caller'
                        ? 'items-start'
                        : item.sender === 'ai'
                        ? 'items-end'
                        : item.sender === 'agent'
                        ? 'items-end'
                        : 'items-center'
                    }`}
                  >
                    {item.sender === 'system' ? (
                      <div className="my-1 px-3 py-1 rounded-full bg-muted text-[10px] text-muted-foreground font-mono text-center border border-border">
                        {item.timestamp} • {item.text}
                      </div>
                    ) : (
                      <div
                        className={`max-w-[85%] rounded-2xl p-3 shadow-xs space-y-1.5 ${
                          item.sender === 'caller'
                            ? 'bg-secondary text-foreground rounded-tl-xs'
                            : item.sender === 'ai'
                            ? 'bg-primary text-primary-foreground rounded-tr-xs'
                            : 'bg-emerald-600 text-white rounded-tr-xs'
                        }`}
                      >
                        <div className="flex items-center justify-between gap-3 text-[10px] opacity-80 font-medium">
                          <span className="flex items-center gap-1">
                            {item.sender === 'caller' ? (
                              <>
                                <User className="w-3 h-3" /> Arayan Hasta / Müşteri
                              </>
                            ) : item.sender === 'ai' ? (
                              <>
                                <Bot className="w-3 h-3" /> Ayşe AI Agent
                              </>
                            ) : (
                              <>
                                <User className="w-3 h-3" /> Temsilci (Mehmet Bey)
                              </>
                            )}
                          </span>
                          <span className="font-mono">{item.timestamp}</span>
                        </div>

                        <p className="text-xs leading-relaxed">{item.text}</p>

                        {/* MCP Action Card if executed during turn */}
                        {item.mcpAction && (
                          <div className="mt-2 p-2 rounded-lg bg-black/20 border border-white/10 text-[11px] font-mono space-y-1">
                            <div className="flex items-center gap-1.5 text-amber-300 font-semibold text-[10px]">
                              <Wrench className="w-3 h-3" />
                              <span>MCP Tool: {item.mcpAction.tool}</span>
                            </div>
                            <div className="text-[10px] text-slate-200">
                              Param: {item.mcpAction.input}
                            </div>
                            <div className="text-[10px] text-emerald-300">
                              Output: {item.mcpAction.output}
                            </div>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>

              {/* Live Status Indicator Footer */}
              <div className="mt-4 pt-3 border-t border-border flex items-center justify-between text-xs text-muted-foreground">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                  <span>FreeSWITCH WebSocket mod_audio_fork Stream Aktif</span>
                </div>
                <div className="font-mono text-[11px]">
                  Diameter Ro: 0.78 TRY debit
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};
