# TODOs

Plan:
1. ✅ AgentWorkflow bauen (Beispiel)
2. ✅ Human-in-the-Loop Workflow bauen (Beispiel)

3. GO+GIN REST API Projektstruktur aufbauen
4. N8N Handler implementieren
    - JSON lesen mit Config
    - 

## N8N Integration

**N8N ApplicationConfig:**
- Workflow Type
    - AgentChat (Chat Trigger)
        - Workflow, in dem kontinuierlich Text zurückgegeben wird (kein Respond to Chat)
    - Human-in-the-Loop
        - hier N8N Chat embedden (https://www.npmjs.com/package/@n8n/chat)
- use unified-chat-history (chat history wird in chatInput mitgegeben als Markdown)


- Platform Service response:
```json

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