# N8N Workflow Stream PoC

Go-basierter Chat Service für Tests mit n8n Workflows.

## Features

- **Chat API**: Synchrone und Streaming-Antworten
- **SSE Streaming**: Server-Sent Events für Echtzeit-Datenübertragung
- **Human-in-the-Loop**: Workflow-Unterbrechung für menschliche Eingaben
- **Mock Endpoints**: Testen ohne n8n-Verbindung

## Voraussetzungen

- Go 1.21+
- n8n (optional, für echte Workflow-Tests)

## Installation

```bash
# Dependencies installieren
go mod tidy

# App starten
go run main.go
```

## Konfiguration

Umgebungsvariablen:

| Variable | Default | Beschreibung |
|----------|---------|--------------|
| `SERVER_PORT` | `8080` | Server Port |
| `N8N_WEBHOOK_URL` | `http://localhost:5678/webhook/chat` | n8n Webhook für sync Chat |
| `N8N_STREAM_WEBHOOK_URL` | `http://localhost:5678/webhook/chat-stream` | n8n Webhook für Streaming |

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Chat (Synchron)
```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hallo!", "session_id": "test-123"}'
```

### Chat (Streaming)
```bash
curl -N -X POST http://localhost:8080/api/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "Erzähl mir eine Geschichte", "session_id": "test-123"}'
```

### Mock Stream (ohne n8n)
```bash
curl -N -X POST http://localhost:8080/api/test/mock-stream \
  -H "Content-Type: application/json" \
  -d '{"message": "Test Nachricht"}'
```

## Human-in-the-Loop

### 1. HITL Request erstellen (blockiert bis Response)
```bash
# Terminal 1: Request erstellen (wartet auf Antwort)
curl -X POST http://localhost:8080/api/hitl/request \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session-123",
    "request_id": "approval-1",
    "question": "Soll ich den Workflow fortsetzen?",
    "options": ["Ja", "Nein", "Abbrechen"],
    "timeout_secs": 300
  }'
```

### 2. Pending Requests anzeigen
```bash
# Terminal 2: Pending Requests abrufen
curl http://localhost:8080/api/hitl/pending
```

### 3. Response senden
```bash
# Terminal 2: Antwort senden
curl -X POST http://localhost:8080/api/hitl/respond \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": "approval-1",
    "response": "Ja",
    "approved": true
  }'
```

## n8n Workflow Setup

### Einfacher Chat Workflow

1. **Webhook Trigger Node**
   - Method: POST
   - Path: `/chat`
   - Response Mode: "When Last Node Finishes"

2. **Code/AI Node**
   - Verarbeite die eingehende Nachricht
   - Generiere Antwort

3. **Respond to Webhook Node**
   - Response Body: JSON mit `message` und `status`

### Streaming Workflow

1. **Webhook Trigger Node**
   - Method: POST
   - Path: `/chat-stream`
   - Response Mode: "Immediately"

2. **HTTP Request Node** (für jeden Token)
   - Method: POST
   - URL: Dein Client SSE Endpoint
   - Body: `{"type": "token", "content": "..."}`

### Human-in-the-Loop Workflow

1. **Webhook Trigger Node**

2. **HTTP Request Node** (HITL Request)
   - Method: POST
   - URL: `http://host.docker.internal:8080/api/hitl/request`
   - Body: `{"question": "Genehmigung erforderlich", ...}`
   - Timeout: Entsprechend konfigurieren

3. **Switch Node**
   - Basierend auf `approved` True/False

4. **Weitere Verarbeitung...**

## Architektur

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  Go Server  │────▶│    n8n      │
│  (Browser)  │◀────│   (SSE)     │◀────│  Workflows  │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Human-in-  │
                    │  the-Loop   │
                    │   Manager   │
                    └─────────────┘
```

## Entwicklung

```bash
# Mit Hot Reload (air installieren: go install github.com/air-verse/air@latest)
air

# Tests
go test ./...

# Build
go build -o n8n-stream main.go
```
