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
3. Fetch Messages inkl. paginierung (infinitive scroll) und order->desc implementieren
4. Platform-Service
    - Application Config für N8N anpassen
    - GET secret endpoint hinzufügen
    - GET config endpoint für agent-service hinzufügen
    - routes zu -> /api/v1/platform-service/* umbenennen
    - autonomous agents
        - hier API Key generieren lassen, inkl. rotate
            - werden in VAULT gespeichert und referenz uri in db auf autonomous-agent
            - PUT /api/v1/platform-service/tenants/{id}/autonomous-agents/{id}/keys/1|2/rotate
                - werden 
5. Agent-Service
    - Platform-Service abfragen
    - Config inkl encryption in redis speichern (3min)
    - traces implementieren
        - beim messages senden -> in jobQueue nach ende die traces fetchen und speichern (N8N -> traces collection)
            - traces mit message id ODER autonomous-agent-id speichern 
        - POST endpoint mit selben service implementieren
            - hier 
        - GET endpoint auf message implementieren
6. Frontend
    - api-client
        - platform-service jetzt auch -> /api/v1/platform-service/*
        - messages und traces endpoints hinzufügen
    - application config für N8N bei CREATE und EDIT anpassen
    - conversations page bauen
7. Foundry anbinden
    - hier direkt checken, wie man mit "Respond to Chat" arbeitet
8. Langchain + Langgraph API
    - state kann als traces an API gesendet werden (nutzt API für traces und gibt messageId an)


Der Ziel Flow wäre:
1. Request arrives
2. Get Config from cache OR if empty: Send request to Platform service
3. parse response from platform service (or cache) and use factory pattern to create needed clients (in this case: N8NAPIClientV1 and N8NChatWorkflowClientV1)
4. Invoke Agent system (in this case: n8n chat url with streaming response and send streaming text to client)
5. store new messages in db (also if error is here (try catch finally -> store))
6. encrypt secrets in config and store request in cache
7. send end message (in stream) to client

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

## Zukunft

- N8N
    - wie tool-calls, reasoning, long runnfing -> status dem ui zeigen?
