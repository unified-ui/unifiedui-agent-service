# ADR 006: Generalized Tool Call Detection for Foundry Agents

## Date: June 2025

## Status: Accepted

## Context

unified-ui supports Reasoning and Tool Call visualization only for ReACT agents (via unifiedui-sdk). Microsoft Foundry agents (Standard-Agents with OpenAPI tools, Workflow-Agents, and future MCP-enabled agents) also emit tool call and workflow events that the Agent Service previously **ignored**.

Goal: All agent types display unified reasoning/tool-call boxes in the frontend using the same SSE event protocol.

### Source of Truth: unifiedui-sdk Streaming Protocol

The SDK (`unifiedui_sdk.streaming`) defines the canonical streaming protocol with 22 event types. All implementations must align with the SDK's `StreamMessage(type, content, config)` format.

Key SDK config contracts:
- **tool_call_start** → `tool_call_id`, `tool_name`, `tool_arguments`
- **tool_call_end** → `tool_call_id`, `tool_name`, `tool_status`, `tool_result`, `tool_error`, `tool_duration_ms`
- **sub_agent_start** → `sub_agent_id`, `sub_agent_name`, `step_number`, `tools`
- **sub_agent_end** → `sub_agent_id`

### Naming Convention: snake_case → camelCase Translation

The Go SSE Writer is the translation layer between backend (Python SDK, snake_case) and frontend (JavaScript, camelCase). The Writer methods map SDK config keys to FE-expected keys:
- `tool_name` → `toolName`
- `tool_arguments` → `toolInput`
- `tool_result` → `toolResult`
- `agent_name` / `sub_agent_name` → `agentName`
- `sub_agent_id` → `agentId`

---

## Decision: Generalized `*_call` / `*_call_output` Pattern

Instead of hardcoding each Foundry tool type (`openapi_call`, `function_call`, `mcp_call`, ...), we detect tool calls generically using suffix matching:

```go
func isToolCall(itemType string) bool {
    return strings.HasSuffix(itemType, "_call") && itemType != "workflow_action"
}

func isToolCallOutput(itemType string) bool {
    return strings.HasSuffix(itemType, "_call_output")
}
```

This automatically supports:
- `openapi_call` / `openapi_call_output` (Standard-Agent OpenAPI tools)
- `function_call` / `function_call_output` (if Foundry uses these)
- `mcp_call` / `mcp_call_output` (MCP tools — future)
- Any future `*_call` types Foundry may add

The original `item.type` (e.g. `"openapi_call"`) is passed as `callType` in the config so the FE can display it as a label.

### Workflow Actions: Special Case

`workflow_action` items use `kind` to discriminate:
- `kind=InvokeAzureAgent` → `SUB_AGENT_START` / `SUB_AGENT_END`
- Other kinds (`SendActivity`, `Question`, etc.) → `TOOL_CALL_START` / `TOOL_CALL_END`
- `kind=EndConversation` → ignored (metadata only)

---

## Foundry Event Analysis (from POC Captures)

### Standard-Agent (StarWarsAgent) — OpenAPI Tool Calls

```
1. response.output_item.added  — type='openapi_call', status='in_progress'
                                   name='StarWarsAPI_listPeople', call_id='call_xxx'
2. response.output_item.done   — type='openapi_call', status='completed', arguments='{}'
3. response.output_item.added  — type='openapi_call_output', status='in_progress'
4. response.output_item.done   — type='openapi_call_output', status='completed'
                                   output='{"response":"..."}'
5. response.output_item.added  — type='message' (assistant text follows)
6. response.output_text.delta  — Text streaming (N events)
7. response.completed
```

Key fields in `openapi_call`:
```json
{
  "type": "openapi_call",
  "id": "fc_xxx",
  "call_id": "call_xxx",
  "name": "StarWarsAPI_listPeople",
  "arguments": "{}",
  "status": "in_progress|completed"
}
```

Key fields in `openapi_call_output`:
```json
{
  "type": "openapi_call_output",
  "id": "fco_xxx",
  "call_id": "call_xxx",
  "output": "{\"response\":\"...\"}",
  "status": "completed"
}
```

### Workflow-Agent (BasicWorkflow)

```
1. response.output_item.added — type='workflow_action', kind='InvokeAzureAgent'
2. [Sub-Agent text streaming]
3. response.output_item.done  — type='workflow_action', kind='InvokeAzureAgent', status='completed'
4. response.output_item.added — type='workflow_action', kind='SendActivity'
5. response.output_item.done  — kind='SendActivity', status='completed'
6. response.output_item.added — type='workflow_action', kind='EndConversation'
7. response.output_item.done  — kind='EndConversation', status='completed'
8. response.completed
```

---

## Mapping: Foundry → Agent Service → SSE → Frontend

### Generic `*_call` Items

| Foundry Event | Chunk Type | SSE Event | FE Config |
|---|---|---|---|
| `output_item.added` + `*_call` | `tool_call_start` | `TOOL_CALL_START` | `toolName`, `toolInput`, `callType` |
| `output_item.done` + `*_call` | `tool_call_stream` | `TOOL_CALL_STREAM` | arguments as content |
| `output_item.done` + `*_call_output` | `tool_call_end` | `TOOL_CALL_END` | `toolResult`, `toolName`, `callType` |

### Workflow Actions

| Foundry Event | Chunk Type | SSE Event | FE Config |
|---|---|---|---|
| `output_item.added` + `workflow_action` + `InvokeAzureAgent` | `sub_agent_start` | `SUB_AGENT_START` | `agentName` |
| `output_item.done` + `workflow_action` + `InvokeAzureAgent` | `sub_agent_end` | `SUB_AGENT_END` | — |
| `output_item.added` + `workflow_action` + other | `tool_call_start` | `TOOL_CALL_START` | `toolName` (=kind) |
| `output_item.done` + `workflow_action` + other | `tool_call_end` | `TOOL_CALL_END` | — |
| `workflow_action` + `EndConversation` | — | (ignored) | — |

---

## Implementation

### 1. Foundry Types (`internal/services/agents/foundry/types.go`)

Add missing fields to `OutputItem`:
- `CallID string` — maps to Foundry's `call_id`
- `Name string` — tool/function name
- `Arguments string` — serialized tool arguments
- `Output string` — tool call result

### 2. Foundry Client (`internal/services/agents/foundry/workflow_client.go`)

Add `isToolCall()` / `isToolCallOutput()` helpers using `strings.HasSuffix`.

Extend `processEvent()`:
- `EventOutputItemAdded` + `isToolCall()` → emit `agents.ChunkTypeToolCallStart` with config
- `EventOutputItemAdded` + `workflow_action` + `InvokeAzureAgent` → emit `agents.ChunkTypeSubAgentStart`
- `EventOutputItemAdded` + `workflow_action` + other → emit `agents.ChunkTypeToolCallStart`
- `EventOutputItemDone` + `isToolCall()` → emit `agents.ChunkTypeToolCallStream` (arguments)
- `EventOutputItemDone` + `isToolCallOutput()` → emit `agents.ChunkTypeToolCallEnd` (result)
- `EventOutputItemDone` + `workflow_action` → emit corresponding end event

### 3. SSE Writer (`internal/api/sse/writer.go`)

Add camelCase key mapping for FE compatibility:
- `WriteToolCallStart`: add `toolName`, `toolInput` keys
- `WriteToolCallEnd`: add `toolResult`, `toolName` keys
- `WriteSubAgentStart`: add `agentName`, `agentId` keys

### 4. Handler (`internal/api/handlers/messages_send.go`)

Add a shared `forwardReasoningChunk()` helper. Wire it into `handleFoundryStreaming()` for all `ChunkTypeToolCall*` and `ChunkTypeSubAgent*` chunk types.

### 5. Frontend

Add `toolResult` and `callType` to `SSEStreamMessage` config interface (`api/types.ts`). No logic changes needed — existing `useChat.ts` already handles all event types.
- [ ] `mcp_call` / `mcp_call_output` Items im Adapter parsen
- [ ] Events an Frontend weiterleiten

### Phase 4: Message Persistence (optional)

- [ ] Reasoning Steps neben Messages in MongoDB/CosmosDB speichern
- [ ] Beim Laden einer Conversation die gespeicherten Steps anzeigen
- [ ] Schema-Design für `reasoning_steps` Collection/Subdocument

### Phase 5: Tests

- [ ] Unit-Tests für neue Event-Typen im Foundry Adapter
- [ ] Unit-Tests für handleFoundryStreaming() mit Tool-Call-Events
- [ ] Integration-Test mit Mock-Foundry-Stream (openapi_call + workflow_action)

---

## Referenz-Dateien

| Datei | Beschreibung |
|-------|-------------|
| `poc/foundry_new/stream_output.txt` | Capture: StarWarsAgent mit OpenAPI Tool Calls |
| `poc/foundry_new/resp_agent_workflow_foundry_stream_1_output.txt` | Capture: BasicWorkflow (einfach) |
| `poc/foundry_new/resp_agent_workflow_foundry_stream_2_output.txt` | Capture: BasicWorkflow (mit InvokeAzureAgent + SendActivity) |
| `poc/foundry_new/capture_stream.py` | Capture-Script (OpenAI SDK + Foundry endpoint) |

## Entscheidung

Implementierung in Phasen 1-5 wie oben beschrieben. Frontend ist bereits vorbereitet (useChat + ReasoningSection). Hauptarbeit liegt im Go Agent-Service (Adapter + Handler).
