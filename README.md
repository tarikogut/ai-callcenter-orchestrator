# AI Call Center Orchestrator & Visual Workflow Suite
### Carrier-Grade Multi-Tenant Visual Drag & Drop AI Call Center Platform
**Author:** Tarık Öğüt ([tarik@icell.cloud](mailto:tarik@icell.cloud)) | **Website:** [https://icell.cloud](https://icell.cloud)

---

## 1. Master Vision & Product Concept
Bu sistem, klasik telekom santrallerini yapay zeka ile birleştiren **n8n / Node-RED benzeri görsel sürükle-bırak (Drag & Drop) iş akışı motoruna** sahip, çok kiracılı (Multi-Tenant) bir **Turnkey AI Contact Center & CPaaS** platformudur.

Her müşteri (Eczane, Kargo, Klinik vb.) kendi web arayüzünde:
1. **Dahili (Extension) & DID Yönetimi:** Kendi dahili numaralarını (101, 102 vb.) ve dış santral numaralarını (0850...) ekler/düzenler/siler.
2. **Görsel İş Akışı Tasarımcısı (n8n Style Flow Builder):** Kod yazmadan blokları birbirine bağlar:
   - `[Gelen Arama]` ➔ `[Çalışma Saatleri Kontrolü]` ➔ `[AI Agent Karşılasın]` ➔ `[Eczane Stok MCP Sorgusu]` ➔ `[Sonuç Bulunamazsa 101 Dahiliye Aktar]`.
3. **Gerçek Zamanlı Diameter Ro Ücretlendirme:** Çağrının her saniyesi ve çalıştırılan her AI/MCP bloğu için Diameter Ro (CCR/CCA) üzerinden gerçek zamanlı kredi düşümü yapılır. Kredi biterse çağrı sonlandırılır veya uyarı anonsu basılır.
4. **Temsilci WebRTC Softphone:** Temsilciler hiçbir ek program kurmadan tarayıcıdan çağrı karşılar. AI'dan gelen çağrının özetini ekranda canlı görür.

---

## 2. Sistem Mimarisi & Katmanlar
```
[ Müşteri Telefonu / WebRTC / GSM ]
                 │
                 ▼ SIP INVITE (DID: 08501110001)
┌─────────────────────────────────────────────────────────────┐
│          KAMAILIO SBC + RTPENGINE (Ön Zırh & Registrar)     │
│  - Multi-Tenant Dahili Kaydı (101@eczane, 101@kargo)        │
│  - Güvenlik, DDoS, NAT Traversal, WebRTC-to-SIP Şifre Çözücü│
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼ SIP / Internal RTP
┌─────────────────────────────────────────────────────────────┐
│          FREESWITCH (Medya & Kuyruk Yönlendiricisi)         │
│  - mod_audio_fork ile Canlı PCM WebSocket Ses Akışı         │
│  - mod_callcenter ile Bekleme Kuyrukları                    │
│  - uuid_transfer ile Dinamik Canlı Temsilciye Aktarma       │
│  - Ses Kayıt (Call Recording)                               │
└──────────────────────────────┬──────────────────────────────┘
                               │ WebSocket (Audio + Control)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│          GO VISUAL WORKFLOW & AI ORCHESTRATOR               │
│                                                             │
│  1. Node Execution Engine (n8n Tarzı Blok Çalıştırıcı):     │
│     • TriggerNode (Inbound Call, DID Match)                 │
│     • AiAgentNode (Persona, VAD, Filler, Gemini Live)       │
│     • McpToolNode (Eczane/Kargo API Çağrıları)              │
│     • ConditionNode (Saat kontrolü, bakiye, niyet)          │
│     • TransferNode (Dahiliye veya Kuyruğa Aktarma)          │
│     • HangupNode (Çağrıyı Sonlandırma)                      │
│                                                             │
│  2. Diameter Ro / Prepaid Rating Engine:                    │
│     • Segment & Dakika başına CCR-Update / CCA kredilendirme│
│     • Kredi tükenince otomatik aksiyon                      │
│                                                             │
│  3. Multi-Tenant Extension & DID Manager                    │
└──────────────────────────────┬──────────────────────────────┘
                               │ REST / WebSocket
                               ▼
┌─────────────────────────────────────────────────────────────┐
│          WEB DASHBOARD & VISUAL FLOW BUILDER                │
│  - n8n Tarzı React-Flow / Vue-Flow Blok Bağlama Arayüzü     │
│  - Extension / DID CRUD Ekranı                              │
│  - Canlı Temsilci WebRTC Softphone (SIP.js)                 │
│  - Canlı Konuşma Transkripti & Raporlama (CDR & Audio)      │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. n8n Tarzı Görsel İş Akışı Blokları (Node Types)

| Blok Adı | Tip | Görevi / Açıklama |
| :--- | :--- | :--- |
| **`InboundCallTrigger`** | Giriş | Çağrı geldiğinde tetiklenir. Arayan numara (`$caller`), aranan DID (`$did`) parametrelerini üretir. |
| **`DiameterRatingNode`** | Faturalandırma | Çağrı başında ve her dakikada Diameter Ro CCR gönderir. Bakiye kontrolü yapar. |
| **`AiVoiceAgentNode`** | Zeka / Ses | Yapay zeka ajanını hatta sokar. Karakter promptu, ses tonu ve filler kelimeler bu blokta ayarlanır. |
| **`McpToolNode`** | Entegrasyon | Dış dünyaya bağlanır (Eczane ilaç stok, kargo takip, CRM, SQL sorgusu). |
| **`ConditionNode`** | Mantık | Karar düğümü: Mesai saatinde miyiz? Müşteri kızgın mı? İlaç bulundu mu? |
| **`TransferNode`** | Çağrı Aktarma | Çağrıyı bir dahiliye (örn: `101`) veya temsilci kuyruğuna (`queue_support`) bağlar. |
| **`PlayAudioNode`** | Anons | Hatta özel bir ses dosyası (WAV/MP3) veya bekleme müziği çalar. |
| **`HangupNode`** | Kapanış | Çağrıyı düzgün şekilde sonlandırır. |

---

## 4. Multi-Tenant İzolasyonu & Özelleştirme Mimarisi (Her Müşteriye Özel Alan)

Her bir kiracı (Tenant / İşletme) sistemde tamamen izole bir profile sahiptir:

1. **Bağımsız API Anahtarları (Bring Your Own Key - BYOK veya Platform Anahtarı):**
   - Eczane kendi OpenAI / Gemini / Claude API anahtarını kullanabilir veya platformun havuzundan tüketir.
   - Her müşterinin tüketim kotası ve maliyeti birbirinden bağımsız ayrışır.
2. **Kişiselleştirilebilir Yapay Zeka Ajanı (Name, Voice & Persona):**
   - **Ajan Adı:** Örn: "Eczacı Asistanı Ayşe", "Jet Kargo Asistanı Can", "Klinik Sekreteri Zeynep".
   - **Karakter & Ton:** "Sıcak ve yardımsever", "Resmi ve net", "Esprili ve enerjik".
   - **Özel Hitap & Filler Cümleleri:** *"Bir saniye stoklarımıza bakıyorum..."*, *"Kargonuzu hemen sorguluyorum..."*.
3. **Müşteriye Özel Bilgi Bankası (FAQ / Knowledge Base / RAG):**
   - Müşteri kendi web panelinden sık sorulan soruları (FAQ), PDF dökümanlarını, çalışma saatlerini, fiyat listesini yükler.
   - Ajan önce müşterinin kendi FAQ hafızasından cevap arar.
4. **Müşteriye Özel MCP (Model Context Protocol) Araçları:**
   - Eczane kendi stok programının MCP sunucusunu bağlar (`https://eczane.com/mcp`).
   - Kargo kendi ERP/kurye takip MCP sunucusunu bağlar (`https://kargo.com/mcp`).
   - Farklı müşterilerin verileri asla birbirine sızamaz.

### Akış Örneği (JSON Workflow Formatı)
```json
{
  "workflow_id": "eczane_hayat_flow_01",
  "tenant_id": "eczane_hayat",
  "name": "Eczane Gündüz Karşılama ve Nöbet Akışı",
  "tenant_settings": {
    "api_key": "enc_gemini_key_sec_991823",
    "faq_dataset_id": "faq_eczane_hayat_v1"
  },
  "nodes": [
    {
      "id": "node_1",
      "type": "InboundCallTrigger",
      "config": { "did": "08501110001" }
    },
    {
      "id": "node_2",
      "type": "DiameterRatingNode",
      "config": { "account_id": "eczane_hayat", "rate_per_min": 0.50 }
    },
    {
      "id": "node_3",
      "type": "AiVoiceAgentNode",
      "config": {
        "agent_name": "Ayşe (Eczacı Asistanı)",
        "persona": "Hayat Eczanesi asistanısın. İlaç sorulursa önce stok MCP'sine bak, genel sağlık sorularında doktora yönlendir.",
        "voice": "turkish_female_warm_01",
        "filler_phrases": ["Hemen stoklarıma bakıyorum...", "Bir saniye sistemden kontrol ediyorum..."]
      }
    },
    {
      "id": "node_4",
      "type": "McpToolNode",
      "config": {
        "mcp_server": "https://api.hayateczanesi.com/mcp",
        "tool_name": "check_stock"
      }
    },
    {
      "id": "node_5",
      "type": "TransferNode",
      "config": {
        "target_extension": "101",
        "condition": "user_wants_human"
      }
    }
  ],
  "connections": [
    { "from": "node_1", "to": "node_2" },
    { "from": "node_2", "to": "node_3" },
    { "from": "node_3", "to": "node_4" },
    { "from": "node_3", "to": "node_5" }
  ]
}
```

---

## 5. Teknoloji Yığını (Technology Stack)

| Katman | Teknoloji / Framework | Rol ve Açıklama |
| :--- | :--- | :--- |
| **Backend Framework** | **Go-Fiber (v2/v3)** | Yüksek hızlı, ultra düşük bellek tüketimli HTTP & WebSocket framework'ü. Express tarzı hızlı routing ve middleware mimarisi. |
| **Veritabanı (Database)** | **PostgreSQL (pgx / GORM)** | Multi-tenant veri ayrımı, iş akışları (JSONB), Tenant profilleri, API anahtarları, dahili/DID kayıtları ve CDR dökümleri. |
| **Frontend Framework** | **React (Vite / TypeScript)** | Hızlı, modern ve modüler kullanıcı arayüzü. |
| **UI & Tasarım Bileşenleri**| **shadcn/ui + Tailwind CSS** | Kurumsal düzeyde, modern, erişilebilir ve temiz UI bileşenleri (Dialog, Sheet, Table, Form vb.). |
| **Görsel Akış Motoru (UI)**| **ReactFlow (@xyflow/react)** | n8n tarzı sürükle-bırak düğümler, bağlantı kabloları ve zoom/pan tuvali. |
| **WebRTC Softphone** | **SIP.js / jssip** | Temsilcilerin tarayıcıdan tek tıkla mikrofon ve kulaklıkla çağrı yanıtlaması. |

---

## 6. Ayrık Portal Mimarisi: Admin Paneli vs Customer (Tenant) Paneli

Sistemde iki bağımsız rol ve kullanıcı deneyimi vardır:

### A. Süper Yönetici Paneli (Super Admin Portal)
- **Kullanıcı:** Platform Sahibi / Sistem Yöneticisi (`icell.cloud` ekibi).
- **Yetenekler:**
  - Tüm işletmeleri (Tenant'ları) listeleme, ekleme, dondurma, silme.
  - Global DID havuzu yönetimi (Hangi numara hangi müşteriye tahsisli?).
  - Global Diameter Ro / Bakiye ve Paket tanımları (Dakika başı ücretler, komisyonlar).
  - Sistem kaynak izleme (Aktif FreeSWITCH / Kamailio oturumları, toplam MPS/çağrı sayısı).
  - Global AI model kotaları ve sunucu durumları.

### B. Müşteri Paneli (Customer / Tenant Portal)
- **Kullanıcı:** Eczane Sahibi, Kargo Müdürü, Doktor / Klinik Sekreteri.
- **Yetenekler:**
  - **n8n Tarzı İş Akışı Tasarımcısı:** Sürükle-bırak ile çağrı karşılama mantığını çizme.
  - **AI Ajan Özelleştirme:** Ajanın adını ("Ayşe"), sesini, karakterini, araya girme cümlelerini belirleme.
  - **Kendi API Key / BYOK:** Kendi OpenAI/Gemini anahtarını girme veya platform bakiyesini görme.
  - **FAQ & Bilgi Bankası:** Soru-cevap metinlerini, çalışma saatlerini, fiyat listesini yükleme.
  - **MCP Entegrasyonu:** Kendi stok/ERP MCP sunucusunun URL ve yetki anahtarını tanımlama.
  - **Dahili Hatlar (Extensions):** `101`, `102` gibi dahili numaraları ve WebRTC temsilcilerini yönetme.
  - **Canlı İzleme & Softphone:** Temsilcilerin tarayıcıdan çağrı açması, konuşma dökümleri (transcripts) ve ses kayıtlarını dinleme.

---

## 7. Proje Modül Ağacı (`ai-callcenter-orchestrator`)
```
ai-callcenter-orchestrator/
├── cmd/
│   ├── orchestrator/       # Go-Fiber Core Server (WebSocket, FreeSWITCH Handler)
│   └── api/                # Go-Fiber REST API (Workflows, Extensions, DIDs, CDR)
├── pkg/
│   ├── db/                 # PostgreSQL bağlantısı, migrasyonlar, repository katmanı
│   ├── workflow/           # n8n benzeri akış motoru (Node Parser, Engine, DAG)
│   ├── tenant/             # Multi-tenant context, DID & Extension CRUD
│   ├── diameter/           # 3GPP TS 32.299 Ro Diameter Rating Client (CCR/CCA)
│   ├── audiofork/          # FreeSWITCH mod_audio_fork WebSocket protocol
│   ├── ai/                 # Real-time Voice (Gemini Live / Deepgram / TTS)
│   ├── mcp/                # Dynamic Tenant MCP Client
│   └── cdr/                # Call Detail Record, Transcripts, Recording Links
├── ui/
│   ├── admin-portal/       # React + shadcn/ui (Süper Admin Yönetim Paneli)
│   └── customer-portal/    # React + shadcn/ui + ReactFlow (Müşteri İş Akışı & PBX Paneli)
└── config/
```

---

## 6. Geliştirme Yol Haritası (Master Roadmap)
- [x] **Adım 1:** n8n tarzı görsel iş akışı, Diameter ücretlendirme, Extension/DID ve WebRTC mimari planının hazırlanması.
- [ ] **Adım 2:** Go `workflow` motoru (Düğümler, akış yürütücüsü ve durum makinesi).
- [ ] **Adım 3:** Diameter Ro gerçek zamanlı şarj istemcisi (CCR-Initial, CCR-Update, CCR-Terminate).
- [ ] **Adım 4:** Multi-tenant Extension & DID yönetim servisi.
- [ ] **Adım 5:** FreeSWITCH ses köprüsü & Canlı AI/MCP çalıştırma döngüsü.
- [ ] **Adım 6:** Web arayüzü (ReactFlow tabanlı sürükle-bırak workflow editörü ve WebRTC softphone).
