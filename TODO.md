# TODOs

Plan:

1. N8N Handler implementieren
    - Config, inkl. Credentials etc als .json entwickeln
    - Platform-Service Handler implementieren
        - erstmal JSON Config lesen
    - stream response from n8n -> AgentNode
2. Messages Collection entwickeln
    - Fields entwickeln
    - Factory Colleczion und MongoDB Client implementieren
    - in endpoint -> Messages fetchen + Messages speichern + messages in chatInput geben

## N8N Integration

**N8N ApplicationConfig:**
- Workflow Type
    - AgentChat (Chat Trigger)
        - Workflow, in dem kontinuierlich Text zurückgegeben wird (kein Respond to Chat)
    - Human-in-the-Loop
        - hier N8N Chat embedden (https://www.npmjs.com/package/@n8n/chat)
- use unified-chat-history (chat history wird in chatInput mitgegeben als Markdown)

- /messages request
```json
{
    "conversationId": "uuid|None",
    "applicationId": "uuid|None",
    "message": {
        "content": "msg",
        "attachements": [
            
        ]
    },
    "invokeConfig": {
        "chatHistoryMessageCount": 15
    }
}
```


- Platform Service response:
```json
siehe ./poc/n8n/config.json
```


- Mit SessionID und Human-in-the-Loop Workflow testen
    - /execution abfrage und aktuellen Step erfahren?
    - ...

- REST API Struktur bauen, inkl. Factory pattern
- externen Service abfragen für Header und invoke URL etc (config)
    - erstmal JSON File lesen, später Platform-Service für Config abfragen
- Invoke implementieren
    1. Nur Chat-Trigger + Agent (Streaming) + executionId zurück
        - config fetchen
        - Agent triggern
        - antwort streamen
        - tracing fetchen
            - entweder über executionId oder sessionId
        - State speichern:
            - collections:
                - messages
                - traces
                - sessions
    2. Human-in-the-Loop
        - same
        - wie arbeitet man mit dem Human-in-the-Loop?
            - /execution fetchen?