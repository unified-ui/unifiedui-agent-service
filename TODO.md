# TODOs

**Plan:**

4. Platform-Service
    - Application Config für N8N validieren
        - config (json field -> gibts schon)
            - api_version
            - workflow_type
            - use_unified_chat_history
            - chat_history_count
            - chat_url
            - api_api_key_credential_id (required)
                - type: N8N_API_KEY
            - chat_auth_credential_id -> {"username": "", "password": ""}.str() | None
                - type: N8N_BASIC_AUTH
    - GET config endpoint für agent-service hinzufügen
        - GET /api/v1/platform-service/applications/{id}/config
            - auth:
                1. request muss von selber origin kommen (localhost -> localhost etc? geht das? -> service-to-service auth...)
                2. user aus bearer token muss zugriff auf application haben (GLOBAL_ADMIN, APPLICATIONS_ADMIN, READ, WRITE, ADMIN)
    - credentials routes anpassen
        - hier richtige typen festlegen
            - API_KEY
            - N8N_API_KEY
            - N8N_BASIC_AUTH -> muss dict mit username und password sein -> wird in string umgewandelt beim speichern
5. Frontend
    - api-client
        - platform-service jetzt auch -> /api/v1/platform-service/*
        - messages endpoints hinzufügen
    - application config für N8N bei CREATE und EDIT anpassen

6. Agent-Service
    - Platform-Service abfragen

7. Frontend
    - conversations page bauen

8. Foundry anbinden
    - hier direkt checken, wie man mit "Respond to Chat" arbeitet
    - ...

9. Langchain + Langgraph API
    - state kann als traces an API gesendet werden (nutzt API für traces und gibt messageId an)

10. Agent-Service
    - GET secret endpoint hinzufügen
        - GET /api/v1/platform-service/tenants/{id}/credentials/{id}/secret
    - autonomous agents
        - hier API Key generieren lassen, inkl. rotate
            - beim erstellen: werden in VAULT gespeichert und referenz uri in db auf autonomous-agent
            - PUT /api/v1/platform-service/tenants/{id}/autonomous-agents/{id}/keys/1|2/rotate
                - werden 
    - traces implementieren
        - beim messages senden -> in jobQueue nach ende die traces fetchen und speichern (N8N -> traces collection)
            - traces mit message id ODER autonomous-agent-id speichern 
        - POST endpoint mit selben service implementieren


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
