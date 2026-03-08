# Konzept: Agent Config Caching im Agent Service

## 1. Ist-Zustand

### 1.1 Session Cache (existiert bereits)

Der Agent Service nutzt **Redis** als Cache-Backend. Aktuell wird ausschließlich **Session-Daten** gecacht:

| Aspekt | Detail |
|--------|--------|
| **Was wird gecacht** | `session.Data` — enthält Agent Config + Chat History pro User/Conversation |
| **Cache Key** | `session:{tenantID}:{userID}:{conversationID}` |
| **TTL** | 180 Sekunden (3 Min), konfigurierbar via `CACHE_TTL_SECONDS` |
| **Verschlüsselung** | AES-verschlüsselt vor dem Speichern |
| **Invalidierung** | TTL-basiert; Stale Entries werden bei Decrypt-Fehler automatisch gelöscht |
| **Implementierung** | `internal/services/session/service.go` |

### 1.2 Config Fetching (aktuell KEIN Cache)

Bei jedem `SendMessage` Request:

```
1. sessionService.GetSession()         → Redis Lookup
2. IF session exists:
     → agentConfig + chatHistory aus Cache (KEIN Platform-Call)
3. IF session NOT exists:
     → platformClient.GetAgentConfig()  → HTTP GET an Platform Service
     → docDBClient.ListChatHistory()    → DocDB Query
4. NACH Streaming:
     → updateSessionCache()  oder updateSessionCacheConfigOnly()
       → Session mit Config + History in Redis schreiben
```

**Problem**: Wenn keine Session existiert (erster Request, Session expired, neues Gespräch), wird **immer** ein HTTP-Call an den Platform Service gemacht. Bei hoher Last kann das zu unnötigen Roundtrips führen, da sich die Agent-Config nur selten ändert.

### 1.3 `X-Use-Cache` Header (existiert im Platform Service)

| Aspekt | Detail |
|--------|--------|
| **Header Name** | `X-Use-Cache` |
| **Default** | `"true"` |
| **Bypass** | `X-Use-Cache: false` → skippt den Platform-internen Redis-Cache |
| **Genutzt in** | Frontend `client.ts` (`noCache?: boolean` Option), Platform Service (Auth Middleware, Dashboard, Tags, etc.) |
| **Aktuell im Agent Service** | **Nicht implementiert** — weder gelesen noch weitergeleitet |

---

## 2. Vorgeschlagenes Konzept: Dedicated Config Cache

### 2.1 Übersicht

Einführung eines **separaten Config-Caches** im Agent Service, der Agent-Konfigurationen **pro Tenant + User + ChatAgent** cached — unabhängig von der Session (Conversation).

```
┌─────────────────────────────────────────────────┐
│                  SendMessage                     │
│                                                  │
│  1. Check Session Cache                          │
│     → HIT: Config + History aus Session          │
│     → MISS: ↓                                    │
│                                                  │
│  2. Check Config Cache  ← NEU                    │
│     → HIT: Config aus Config-Cache               │
│            + History aus DocDB                   │
│     → MISS: ↓                                    │
│                                                  │
│  3. Platform Service Call                         │
│     → Config fetchen (inkl. Auth-Check)          │
│     → Config in Config-Cache schreiben  ← NEU    │
│     → History aus DocDB laden                    │
│                                                  │
│  4. Session Cache schreiben (wie bisher)          │
└─────────────────────────────────────────────────┘
```

### 2.2 Cache-Architektur

#### Cache Key

```
config:{tenantID}:{userID}:{chatAgentID}
```

**Warum MIT UserID?** Der Platform Service prüft beim `GetChatAgentConfig`-Call die **Berechtigung des Users** (via `authToken`). Ohne UserID im Cache Key könnte folgender Sicherheits-Bypass entstehen:

```
1. User A (berechtigt) chattet mit Agent X → Config wird in Cache geschrieben
2. User B (NICHT berechtigt) chattet mit Agent X → Config-Cache HIT
   → Platform Service wird NICHT aufgerufen → Keine Auth-Prüfung!
   → User B kann mit Agent X chatten, obwohl er keine Berechtigung hat
```

Mit UserID im Key ist jeder Cache-Entry an einen spezifischen User gebunden. Der Platform Service wird beim ersten Request eines Users **immer** aufgerufen (= Auth-Check), und nur Folgerequests desselben Users (innerhalb der TTL) sind Cache-Hits.

> **Effizienz-Impakt**: Die Multi-User-Cache-Effizienz (ein Entry für alle User) geht verloren. Der **Hauptgewinn bleibt** aber bestehen: Ein User mit mehreren Conversations zum selben Agent braucht nur 1x den Platform Service Call pro TTL-Fenster, statt 1x pro neue Conversation.

> **Session Cache vs. Config Cache**: Der Session Cache ist per `session:{t}:{u}:{conversationID}` — jede neue Conversation braucht einen neuen Platform Call. Der Config Cache ist per `config:{t}:{u}:{chatAgentID}` — überlebt über Conversations hinweg.

#### Datenstruktur

```go
type ConfigCacheEntry struct {
    Config    *platform.ChatAgentConfigResponse `json:"config"`
    CachedAt  time.Time                          `json:"cachedAt"`
}
```

#### TTL

- **Default**: `300 Sekunden (5 Minuten)` — konfigurierbar via `CONFIG_CACHE_TTL_SECONDS`
- Bewusst länger als der Session-Cache (180s), da Configs sich seltener ändern als Chat History

#### Verschlüsselung

Wie beim Session-Cache: **AES-verschlüsselt** in Redis, da die Config sensible Daten enthält (AI Model API Keys, Tool Credentials).

### 2.3 Cache-Regeln

#### Nur erfolgreiche Responses cachen

**Niemals** cachen bei:
- HTTP 4xx Responses (Bad Request, Forbidden, Not Found, etc.)
- HTTP 5xx Responses (Internal Server Error, etc.)
- Netzwerkfehler / Timeouts
- Leere oder invalide Config-Responses

**Nur** cachen bei:
- HTTP 200 mit valider `ChatAgentConfigResponse`

#### No-Cache: `X-Use-Cache` Header

```
Frontend  →  Agent Service  →  Platform Service
             (liest Header)     (liest Header)
```

| Header | Verhalten im Agent Service |
|--------|----------------------------|
| `X-Use-Cache: true` (default) | Config-Cache nutzen (Read + Write) |
| `X-Use-Cache: false` | Config-Cache **skippen**, direkt Platform Service anfragen, Header an Platform Service **weiterleiten** |

**Playground**: Das Frontend setzt `X-Use-Cache: false` im Playground, damit Entwickler immer die aktuellste Config sehen, z.B. direkt nach Änderungen an Prompts, Tools oder AI Models.

### 2.4 Cache-Invalidierung

#### Strategie: TTL-basiert (wie Session-Cache)

Es wird **keine** explizite Invalidierung implementiert. Gründe:

1. **Einfachheit**: Kein zusätzlicher Invalidierungskanal (Webhook, Event Bus, etc.) zwischen Platform Service und Agent Service nötig
2. **Konsistenz**: Gleiche Strategie wie beim existierenden Session-Cache
3. **Akzeptable Latenz**: Max 5 Minuten bis eine Config-Änderung wirksam wird — für Produktions-Agenten akzeptabel
4. **Playground-Bypass**: Entwickler nutzen `X-Use-Cache: false` im Playground → sofortige Änderungen

#### Optionale Erweiterung (Zukunft)

Falls kürzere Invalidierungszeiten nötig werden:
- **Event-basiert**: Platform Service published ein Redis Pub/Sub Event bei Config-Änderung → Agent Service invalidiert betroffenen Cache-Entry
- **Webhook**: Platform Service ruft Agent Service Webhook auf → `DELETE config:{tenantID}:{chatAgentID}`

### 2.5 Integration in den Request-Flow

#### `messages_send.go` — Angepasster Flow

```go
// 1. Session Cache (wie bisher)
sessionData, err := h.sessionService.GetSession(ctx, tenantID, userID, conversationID)

if sessionData != nil {
    agentConfig = sessionData.Config
    chatHistory = sessionData.ChatHistory
} else {
    // 2. NEU: Config Cache prüfen (wenn X-Use-Cache != false)
    useCache := c.GetHeader("X-Use-Cache") != "false"
    
    if useCache {
        cachedConfig, err := h.configCache.Get(ctx, tenantID, userID, chatAgentID)
        if err == nil && cachedConfig != nil {
            agentConfig = cachedConfig
            // History trotzdem aus DocDB laden
            chatHistory = loadChatHistory(...)
        }
    }
    
    // 3. Platform Service Call (nur wenn kein Cache-Hit)
    if agentConfig == nil {
        agentConfig, err = h.platformClient.GetAgentConfig(ctx, tenantID, chatAgentID, conversationID, authToken)
        // ...
        
        // Config in Cache schreiben (nur bei Erfolg)
        if useCache && err == nil {
            _ = h.configCache.Set(ctx, tenantID, userID, chatAgentID, agentConfig)
        }
        
        chatHistory = loadChatHistory(...)
    }
}
```

#### `X-Use-Cache` Header weiterleiten

Wenn der Agent Service den Platform Service aufruft, muss der `X-Use-Cache` Header weitergereicht werden:

```go
func (c *client) GetChatAgentConfig(ctx context.Context, tenantID, chatAgentID, authToken string, useCache bool) (*ChatAgentConfigResponse, error) {
    headers := map[string]string{
        "X-Service-Key": c.serviceKey,
        "Authorization": "Bearer " + authToken,
    }
    if !useCache {
        headers["X-Use-Cache"] = "false"
    }
    // ...
}
```

### 2.6 Config Cache Service Interface

```go
package configcache

type Service interface {
    Get(ctx context.Context, tenantID, userID, chatAgentID string) (*platform.AgentConfig, error)
    Set(ctx context.Context, tenantID, userID, chatAgentID string, config *platform.AgentConfig) error
    Delete(ctx context.Context, tenantID, userID, chatAgentID string) error
    DeleteByTenant(ctx context.Context, tenantID string) error
}
```

Die Implementierung nutzt den bestehenden `cache.Client` (Redis) und `encryption.Encryptor` — gleiche Infrastruktur wie der Session-Cache.

---

## 3. Performance-Impact

### Vorher (ohne Config-Cache)

| Szenario | Platform Service Calls |
|----------|----------------------|
| Erste Nachricht in Conversation | 1x GetAgentConfig |
| Folgende Nachricht (Session aktiv) | 0 |
| Nachricht nach Session-Expiry (3 Min) | 1x GetAgentConfig |
| 10 User, selber Agent, keine Session | 10x GetAgentConfig |
| 1 User, 5 Conversations, keine Session | 5x GetAgentConfig |

### Nachher (mit Config-Cache)

| Szenario | Platform Service Calls |
|----------|----------------------|
| Erste Nachricht in Conversation | 1x GetAgentConfig (+ Config-Cache Write) |
| Folgende Nachricht (Session aktiv) | 0 |
| Nachricht nach Session-Expiry (3 Min) | 0 (Config-Cache Hit, nur DocDB für History) |
| 10 User, selber Agent, keine Session | 10x GetAgentConfig (1x pro User wegen Auth) |
| 1 User, 5 Conversations, keine Session | 1x GetAgentConfig + 4x Config-Cache Hit |
| Playground | 1x GetAgentConfig (kein Cache) |

**Ersparnis**: Besonders bei Multi-Conversation-Szenarien signifikant — ein User der mehrere Conversations mit demselben Agent hat, braucht nur 1x den Platform Service Call pro TTL-Fenster.

---

## 4. Zusammenfassung

| Aspekt | Session Cache (existiert) | Config Cache (NEU) |
|--------|---------------------------|---------------------|
| **Scope** | Pro User + Conversation | Pro User + ChatAgent |
| **Cache Key** | `session:{t}:{u}:{c}` | `config:{t}:{u}:{a}` |
| **Inhalt** | Config + Chat History | Nur Config |
| **TTL** | 180s (3 Min) | 300s (5 Min) |
| **Verschlüsselung** | AES | AES |
| **Invalidierung** | TTL | TTL |
| **No-Cache Bypass** | Nein | Ja (`X-Use-Cache: false`) |
| **Playground** | - | Immer bypass (`X-Use-Cache: false`) |
| **Nur bei 2xx** | N/A | Ja — niemals Fehler cachen |
