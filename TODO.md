# TODOs

**DONE**

**Plan:**

7. Frontend
    - Hover Chat Agents -> OnClick -> gehe zu conversations`chat-agent={id}
    - ConversationPage
        - Design
            - Header
                - ohne border
            - Content:
                - Nachrihcten korrekt sortieren
                    - kommen aus Backend bereits richtig sortiert
                - wenn conversationId = null
                    - dann ChatInput in mitte der Page, damit Schick
                - bei file-hover -> drop icon in der mitte deutlicher bzw den hintergrund blur
            - ChatInput
                - Backegroundcolor soll wie die der page sein (--app-bg?)
                - dafür soll border und shadow gegeben sein, damit der ChatInput etwas hochsticht
            - Sidebar
                - Expand + collaps button für sidebar -> andere icons
                - + New Chat Button mehr padding -> zu geringes padding
                - Search conversations bar raus -> dafür gibts den Button "Search chats"
                - Chat History
                    - Label "Today / Agent-Name"
                        - mit mehr abstand nach links (ist direkt am rand) und dick (font-weight)
                    - conversation item > 3-dots icon
                        - rename über Pin hinzufügen -> und direkt links in der sidebar den namen/title anpassen können
        - Logik
            - Header
                - Share Dialog erstellen
                    - manage access tabelle für conversation
                        - es dürfen nur principals ausgewählt werden, die zugriff auf die Application haben
                        - 

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

11. ZWEI Vaults fixen:
    - app_vault + secrets_vault
        - App Vault für application keys wie zB `X-Service-Key`
        - Secrets Vault -> ist vault für credentials aus der app etc...
    - *aktuell in auth.py > _validate_service_key soll app_vault nutzen
        - app_vault kann auch dotenv sein...

12. Frontend fix:
    - beim fetchen der Credentials im Create- und EditApplicationDialog wird noch credentials?limit=999 gefetcht -> hier eher paginierung, aber man kann ruhig 100 fetchen (nur name und id -> + orderBy=name order_direction=asc)

13. models.py refactoren
    - überall wo uuid von uns -> char(36) nutzen
        - zb bei Conversation.application_id string(100) -> char(36)

14. Bei delete conversation -> auch messages und traces löschen



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
