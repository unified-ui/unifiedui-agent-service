# Trace Algorithm Review — Foundry & N8N Transformers

## Inhaltsverzeichnis

1. [Zusammenfassung](#1-zusammenfassung)
2. [Aktueller Algorithmus — Foundry](#2-aktueller-algorithmus--foundry)
3. [Aktueller Algorithmus — N8N](#3-aktueller-algorithmus--n8n)
4. [Datenstruktur-Analyse — Foundry API](#4-datenstruktur-analyse--foundry-api)
5. [Datenstruktur-Analyse — N8N API](#5-datenstruktur-analyse--n8n-api)
6. [Identifizierte Probleme — Foundry](#6-identifizierte-probleme--foundry)
7. [Identifizierte Probleme — N8N](#7-identifizierte-probleme--n8n)
8. [Domain-Model Review (TraceNode)](#8-domain-model-review-tracenode)
9. [Konzept — Neuer Foundry Algorithmus](#9-konzept--neuer-foundry-algorithmus)
10. [Konzept — Neuer N8N Algorithmus](#10-konzept--neuer-n8n-algorithmus)
11. [Frontend-Auswirkungen](#11-frontend-auswirkungen)
12. [Empfehlungen & Priorisierung](#12-empfehlungen--priorisierung)

---

## 1. Zusammenfassung

Die aktuellen Trace-Transformer für **Foundry** (`internal/services/traceimport/foundry/transformer.go`, 772 Zeilen) und **N8N** (`internal/services/traceimport/n8n/transformer.go`, 427 Zeilen) haben signifikante algorithmische Schwächen, die zu semantisch falschen Trace-Hierarchien führen.

**Foundry-Kernproblem**: Die Hierarchie-Bildung basiert auf `response_id`-Gruppierung. Dabei werden semantisch unabhängige Workflow-Aktionen (z.B. `EndConversation`) fälschlicherweise als Kinder von `SendActivity` gruppiert, weil sie die gleiche `response_id` teilen. Die verfügbare `action_id`/`previous_action_id`-DAG-Struktur wird ignoriert.

**N8N-Kernproblem**: Die Ausgabe ist **komplett flach** — alle Nodes auf gleicher Ebene. N8N liefert einen vollständigen Verbindungsgraphen (`WorkflowData.Connections`) mit expliziten Connection-Typen (`main`, `ai_languageModel`, `ai_tool`, etc.) sowie `Source`-Felder je `NodeExecution`, aber beides wird nicht für die Hierarchie-Bildung verwendet.

---

## 2. Aktueller Algorithmus — Foundry

### 2.1 Ablauf

```
Transform(items, config) → []TraceNode
├── 1. Items reversieren (API liefert newest-first)
├── 2. Gruppierung nach response_id → responseGroups map
├── 3. Gruppierung nach approval_request_id → approvalGroups map
├── 4. SendActivity-Container finden → sendActivityContainers map
└── 5. buildTraceNodesWithHierarchy() → Endgültige TraceNode-Liste
```

### 2.2 Kern-Logik: `buildTraceNodesWithHierarchy()`

Iteriert über alle Items mit einer `processedIDs`-Map:

| Bedingung | Aktion |
|---|---|
| Item hat `response_id` UND ein `SendActivity`-Container existiert für diese `response_id` | **Skip** (wird als Kind des SendActivity verarbeitet) |
| Type = `message` | → `transformMessage()` → `NodeTypeLLM` |
| Type = `workflow_action` mit Kind = `SendActivity` | → `transformSendActivityWithChildren()` → Container mit Kindern der gleichen `response_id` |
| Type = `workflow_action` (andere Kinds) | → `transformWorkflowAction()` → Standalone `NodeTypeWorkflow` |
| Type = `mcp_approval_request` | → `transformMCPGroup()` → Gruppiert approval_request + response + call |
| Type = `mcp_call` (ohne Approval) | → `transformMCPCall()` → Standalone `NodeTypeTool` |
| Type = `mcp_list_tools` | → `transformMCPListTools()` |
| Default | → `transformUnknown()` |

### 2.3 SendActivity-Container-Logik

`transformSendActivityWithChildren()` erstellt einen Container-Node vom `SendActivity`-Item und iteriert dann über **alle Items mit der gleichen response_id**, die zu Kinder-Nodes werden.

---

## 3. Aktueller Algorithmus — N8N

### 3.1 Ablauf

```
TransformExecution(execution, config) → Trace
├── 1. Iteriere über RunData map (nodeName → []NodeExecution)
├── 2. Für jede NodeExecution:
│   ├── mapNodeType() → NodeType (string-basierte Heuristik)
│   ├── buildNodeData() → Input/Output Extraktion
│   └── buildNodeMetadata() → n8n_node_type, token_usage, source, etc.
├── 3. Flat list aller TraceNodes
└── 4. SortNodesByTime() → Sortierung nach startTime
```

### 3.2 Kern-Logik

- **Komplett flat**: Keine Nutzung von `WorkflowData.Connections` oder `Source`-Feldern für Hierarchie
- **mapNodeType()**: String-Matching-Heuristik auf N8N-Typ-Namen:
  - `trigger` → `workflow`, `agent` → `agent`, `lmChat` → `llm`, `httpRequest` → `http`, etc.
- **buildNodeData()**: Extrahiert Input aus `inputOverride`/Chat-Trigger-Daten, Output aus `Main`-Branches
- **buildNodeMetadata()**: Speichert `n8n_node_type`, `run_index`, `token_usage`, `sub_execution`, `error`, `source` (als JSON-String)

---

## 4. Datenstruktur-Analyse — Foundry API

### 4.1 ConversationItem-Typen

| Type | Kind-Varianten | Beschreibung |
|---|---|---|
| `message` | — | Benutzer- oder Assistenten-Nachrichten. Rolle via `role`-Feld |
| `workflow_action` | `SendActivity`, `Question`, `InvokeAzureAgent`, `EndConversation` | Workflow-Orchestrierungsschritte |
| `mcp_call` | — | MCP Tool-Aufrufe |
| `mcp_approval_request` | — | MCP Human-in-the-Loop Anfrage |
| `mcp_approval_response` | — | MCP Genehmigungsantwort |
| `mcp_list_tools` | — | Verfügbare Tools je Server |

### 4.2 Verfügbare DAG-Felder (aktuell NICHT verwendet)

Jedes `workflow_action`-Item hat:

```json
{
  "action_id": "action-1767634451682",
  "parent_action_id": "trigger_wf",
  "previous_action_id": "action-1767634350800_Post"
}
```

- **`action_id`**: Eindeutige ID der Aktion
- **`parent_action_id`**: Immer `"trigger_wf"` bei Workflow-Agents (alle Top-Level unter dem Trigger)
- **`previous_action_id`**: Verkettet Aktionen sequentiell → **bildet eine verkettete Liste**

**Beobachtete Kette (aus POC-Daten, Conversation c671)**:
```
trigger_wf
  → SendActivity (action-1767653509477, Question)
    → InvokeAzureAgent (action-1767634350800)
      → SendActivity (action-1767634451682)
        → EndConversation (action-1767634464332)
```

### 4.3 Response-Gruppierung (aktuell verwendet)

```json
{
  "created_by": {
    "response_id": "wfresp_abc123"  // Alle Items eines API-Response-Zyklus teilen diese ID
  }
}
```

**Problem**: Die `response_id` ist zu grobgranular — sie gruppiert Items, die **zeitlich zusammen** zurückgeliefert werden, nicht Items, die **semantisch zusammengehören**.

### 4.4 Zwei Agent-Typen mit unterschiedlicher Trace-Struktur

| Typ | Beispiel | Trace-Charakteristik |
|---|---|---|
| **Simple Agent** (BasicAssistantAgent) | `resp_tracing5.json` | Nur `message`-Items, keine `workflow_action`s. Hat `partition_key`. |
| **Workflow Agent** (BasicWorkflow) | `resp_tracing_conv_c671*.json` | `workflow_action`s + `message`s + optional `mcp_*`. Messages von Sub-Agents haben **keine** workflow `response_id`. |

### 4.5 Sub-Agent Messages

Messages, die von aufgerufenen Sub-Agents stammen (z.B. `BasicAssistantAgent` via `InvokeAzureAgent`), haben ein `created_by`, das den Sub-Agent-Namen enthält, aber **keine `response_id`** des Workflows. Diese Messages werden aktuell nicht korrekt zugeordnet.

```json
// Sub-Agent Message (kein response_id vom Workflow)
{
  "type": "message",
  "role": "assistant",
  "created_by": {
    "agent": { "name": "BasicAssistantAgent" }
    // KEIN response_id hier
  }
}

// Workflow Message (hat response_id)
{
  "type": "message",
  "role": "assistant",
  "created_by": {
    "response_id": "wfresp_abc123",
    "agent": { "name": "..." }
  }
}
```

---

## 5. Datenstruktur-Analyse — N8N API

### 5.1 Execution-Response-Struktur

```
ExecutionResponse
├── WorkflowData
│   ├── Nodes []WorkflowNode          // Node-Definitionen (Type, Position, etc.)
│   └── Connections map[string]any     // ⭐ Vollständiger Verbindungsgraph
└── Data.ResultData
    └── RunData map[string][]NodeExecution  // Ausführungsdaten pro Node
        ├── StartTime, ExecutionTime
        ├── Source []NodeExecutionSource    // ⭐ previousNode-Referenz
        ├── Data (Main branches mit Output)
        └── Metadata (token_usage, sub_execution)
```

### 5.2 Connections-Graph (aktuell NICHT verwendet)

Das `Connections`-Feld enthält den **vollständigen Node-Graphen** mit **benannten Connection-Typen**:

```json
{
  "Chat Trigger": {
    "main": [[{"node": "Respond to Webhook", "type": "main", "index": 0}]]
  },
  "Respond to Webhook": {
    "main": [[{"node": "AI Agent", "type": "main", "index": 0}]]
  },
  "Azure OpenAI Chat Model": {
    "ai_languageModel": [[{"node": "AI Agent", "type": "ai_languageModel", "index": 0}]]
  }
}
```

**Zwei Connection-Typen**:

| Type | Bedeutung | Visualisierung |
|---|---|---|
| `main` | Sequentieller Kontrollfluss | Nodes auf gleicher Ebene, sequentiell |
| `ai_languageModel`, `ai_tool`, `ai_memory`, `ai_outputParser` | Sub-Node-Beziehung | **Kind-Node** des Ziel-Nodes |

### 5.3 Source-Feld (aktuell nur als Metadaten gespeichert)

```json
{
  "source": [
    { "previousNode": "Respond to Webhook", "previousNodeRun": 0, "previousNodeOutput": 0 }
  ]
}
```

### 5.4 Branching-Pattern (aus POC execution_1716)

```json
{
  "Switch": {
    "main": [
      [{"node": "HTTP Request", "type": "main", "index": 0}],   // Branch 0
      [{"node": "Merge1", "type": "main", "index": 1}]          // Branch 1
    ]
  }
}
```

**Switch** hat mehrere Output-Branches (Array-Index = Branch-Index). **Merge** kombiniert Branches.

### 5.5 N8N Cluster-Nodes (AI-Workflows)

N8N unterscheidet "Root Nodes" (AI Agent, Chain, etc.) und "Sub-Nodes" (Chat Model, Tools, Memory, etc.). Sub-Nodes verbinden sich über **nicht-main**-Connection-Typen:

- `ai_languageModel` → LLM-Model als Sub-Node
- `ai_tool` → Tool als Sub-Node
- `ai_memory` → Memory als Sub-Node
- `ai_outputParser` → Output Parser als Sub-Node

Diese Hierarchie ist **direkt aus den Connections ableitbar**.

---

## 6. Identifizierte Probleme — Foundry

### P-F1: EndConversation wird Kind von SendActivity (KRITISCH)

**Ist-Zustand**: `EndConversation` teilt die gleiche `response_id` wie `SendActivity` und wird daher als Kind-Node gruppiert.

**Soll-Zustand**: `EndConversation` ist semantisch ein **eigenständiger Workflow-Schritt** nach `SendActivity`.

**Beispiel (Conversation c671, Response 2)**:
```
Aktuell:                              Soll:
SendActivity (Container)              InvokeAzureAgent
├── InvokeAzureAgent                    └── Message (BasicAssistantAgent)
├── Message "Conversation..."         SendActivity
├── EndConversation                     └── Message "Conversation..."
└── ...                               EndConversation
```

### P-F2: response_id-Gruppierung zu grobgranular (KRITISCH)

Alle Workflow-Items eines API-Response-Zyklus teilen die gleiche `response_id`. Die Gruppierung danach erzeugt **falsche Parent-Child-Beziehungen**:

- `InvokeAzureAgent`, `SendActivity`, `EndConversation` landen alle in einer Gruppe
- Semantisch sind dies aber **sequentielle Schritte** im Workflow

### P-F3: action_id/previous_action_id-DAG wird ignoriert (KRITISCH)

Die Foundry API liefert eine explizite **sequentielle Verkettung** über `previous_action_id`. Diese wird aktuell nicht für die Hierarchie-Bildung genutzt. Stattdessen wird die grobgranulare `response_id` verwendet.

### P-F4: Sub-Agent-Messages werden nicht zugeordnet (MITTEL)

Messages von aufgerufenen Sub-Agents (z.B. `BasicAssistantAgent`) haben keine `response_id` des aufrufenden Workflows. Sie werden als Standalone-Nodes behandelt statt als Kinder des `InvokeAzureAgent`-Nodes.

### P-F5: Simple vs. Workflow Agents nicht unterschieden (NIEDRIG)

Einfache Agents (`BasicAssistantAgent`) erzeugen nur Messages (kein Workflow). Workflow-Agents erzeugen `workflow_action`s. Der Transformer unterscheidet nicht zwischen diesen Fällen, was bei einfachen Agents zu unerwarteten Ergebnissen führen kann.

---

## 7. Identifizierte Probleme — N8N

### P-N1: Komplett flache Ausgabe (KRITISCH)

**Ist-Zustand**: Alle Nodes werden als flache Liste ausgegeben, sortiert nach `startTime`.

**Soll-Zustand**: Hierarchische Struktur basierend auf dem Connections-Graphen.

**Beispiel (execution_119)**:
```
Aktuell:                              Soll:
Chat Trigger                          Chat Trigger
Respond to Webhook                    └── Respond to Webhook
AI Agent                                  └── AI Agent
Azure OpenAI Chat Model                       └── Azure OpenAI Chat Model (LLM)
```

### P-N2: Connections-Graph wird ignoriert (KRITISCH)

`WorkflowData.Connections` enthält den **vollständigen DAG** mit Connection-Typen. Dieser wird weder gelesen noch verarbeitet. Dies ist die primäre Datenquelle für die Hierarchie.

### P-N3: Sub-Node-Beziehungen nicht modelliert (KRITISCH)

N8N Sub-Nodes (`ai_languageModel`, `ai_tool`, `ai_memory`) verbinden sich über nicht-main-Connection-Typen. Diese Parent-Child-Beziehung ist eindeutig aus dem Graphen ablesbar, wird aber nicht genutzt.

### P-N4: Branching/Merging nicht dargestellt (MITTEL)

Switch-Nodes erzeugen Branches, Merge-Nodes vereinen sie. Die DAG-Struktur (Branching mit Index-basierten Outputs) geht in der flachen Darstellung komplett verloren.

### P-N5: Source-Feld nur als Metadaten (NIEDRIG)

Das `Source`-Feld (previousNode, previousNodeRun, previousNodeOutput) wird als JSON-String in die Metadaten geschrieben, aber nicht zur Graphen-Rekonstruktion genutzt.

---

## 8. Domain-Model Review (TraceNode)

### 8.1 Aktuelles Modell

```go
type TraceNode struct {
    ID          string            // UUID
    Name        string
    Type        NodeType          // agent|tool|llm|chain|retriever|workflow|function|http|code|conditional|loop|custom
    ReferenceID string
    StartAt     *time.Time
    EndAt       *time.Time
    Duration    *int64
    Status      NodeStatus        // pending|running|completed|failed|skipped|cancelled
    Logs        []string
    Data        *NodeData         // Input/Output (Text, ExtraData, Metadata)
    Nodes       []TraceNode       // ⭐ Rekursive Sub-Nodes
    Metadata    map[string]string
}
```

### 8.2 Bewertung

**Stärken**:
- Rekursive `Nodes`-Struktur ermöglicht beliebig tiefe Hierarchien
- `NodeType`-Enum deckt die meisten Use Cases ab
- `NodeData` mit Input/Output ist flexibel

**Schwächen / Mögliche Verbesserungen**:

| Bereich | Problem | Empfehlung |
|---|---|---|
| Keine Edge-Informationen | TraceNode speichert keine Beziehungs-Metadaten (z.B. "ist ai_languageModel von X") | `ConnectionType`-Feld in Metadaten oder dediziertes Feld |
| Kein Branch-Index | Bei Switch/If-Nodes ist der Output-Branch-Index nicht gespeichert | In Metadaten aufnehmen (`branch_index`) |
| Keine Unterscheidung Sequenz vs. Parallel | Kinder-Nodes könnten sowohl sequentiell als auch parallel sein | `execution_order: sequential|parallel` in Metadaten |

**Fazit**: Das aktuelle Domain-Modell ist **ausreichend flexibel**. Die rekursive `Nodes`-Struktur kann sowohl sequentielle Ketten als auch Parent-Child-Beziehungen abbilden. Erweiterte Informationen (Connection-Type, Branch-Index) können über `Metadata` transportiert werden, ohne das Modell zu ändern.

---

## 9. Konzept — Neuer Foundry Algorithmus

### 9.1 Grundidee

Ersetze die `response_id`-basierte Gruppierung durch eine **Zwei-Phasen-Strategie**:

1. **Phase 1**: Baue die sequentielle Aktionskette via `previous_action_id`
2. **Phase 2**: Ordne Messages den jeweiligen Aktionen zu (über Zeitstempel + Kontext)

### 9.2 Algorithmus

```
Transform(items, config) → []TraceNode
├── 1. Items reversieren (API newest-first → chronologisch)
├── 2. Items nach Typ klassifizieren:
│   ├── workflowActions []ConversationItem
│   ├── messages []ConversationItem  
│   ├── mcpItems []ConversationItem
│   └── otherItems []ConversationItem
├── 3. Action-DAG aufbauen:
│   ├── actionMap: action_id → ConversationItem
│   └── actionChain: previous_action_id → []action_id (sequentielle Verkettung)
├── 4. Für jede Workflow-Action:
│   ├── TraceNode erstellen (NodeTypeWorkflow)
│   ├── Zugehörige Messages zuordnen:
│   │   ├── Messages mit gleicher response_id → als Kind-Nodes
│   │   └── Sub-Agent-Messages (ohne response_id) → via Zeitfenster + InvokeAzureAgent-Zuordnung
│   └── MCP-Items zuordnen:
│       └── MCP-Gruppen (approval_request/response/call) → als Kind-Nodes des auslösenden Actions
├── 5. Standalone-Messages (ohne zugehörige Action) als Top-Level-Nodes
└── 6. Sortierung nach Zeitstempel
```

### 9.3 Action-Zuordnung von Messages

**Neue Logik für Message-Zuordnung:**

```
Für jede Message M:
    1. Falls M.response_id existiert UND eine Action A hat die gleiche response_id:
       → M wird Kind von A
       → Unterscheide nach A.Kind:
         - SendActivity: M ist die gesendete Nachricht (Output)
         - Question: M ist die gestellte Frage (Output)
    2. Falls M.created_by.agent existiert UND kein response_id:
       → Suche zeitlich nächsten InvokeAzureAgent vor M
       → M wird Kind des InvokeAzureAgent-Nodes
    3. Falls M.role == "user":
       → M ist eine User-Input-Message → Top-Level oder als Input des ersten Actions
```

### 9.4 Ergebnis-Beispiel (Conversation c671)

```
Response 1:
├── User Message "hi"
├── SendActivity (Question) [workflow]
│   └── Message "What is your name?" [llm]
├── User Message "Hey du!"

Response 2:
├── InvokeAzureAgent [workflow]
│   └── Message from BasicAssistantAgent [llm]
├── SendActivity [workflow]
│   └── Message "Conversation is ending here." [llm]
└── EndConversation [workflow]
```

### 9.5 Vorteile

- **Semantisch korrekte Hierarchie**: Jede Action ist ein eigenständiger Schritt
- **Messages korrekt zugeordnet**: Auch Sub-Agent-Messages
- **Zukunftssicher**: `previous_action_id`-Kette skaliert mit neuen Action-Kinds
- **Einfachere Logik**: Keine komplexe response_id-Gruppierung mit processedIDs-Map

---

## 10. Konzept — Neuer N8N Algorithmus

### 10.1 Grundidee

Nutze den `Connections`-Graphen, um eine **echte Hierarchie** aufzubauen. Unterscheide zwischen:
- **Main-Connections**: Sequentieller Flow → Nodes auf gleicher Ebene oder als sequentielle Kette
- **Non-Main-Connections**: Sub-Node-Beziehungen → Kind-Nodes

### 10.2 Algorithmus

```
TransformExecution(execution, config) → Trace
├── 1. Connections-Graph parsen:
│   ├── Parse WorkflowData.Connections zu typisierter Struktur
│   ├── mainEdges: map[sourceNode][]targetNode (nur type="main")
│   └── subEdges: map[sourceNode][]targetNode (type=ai_* etc.)
├── 2. Inverse Sub-Edge-Map:
│   └── parentOf: map[subNode]parentNode
│       (z.B. "Azure OpenAI Chat Model" → parent: "AI Agent")
├── 3. Root-Nodes identifizieren:
│   └── Nodes die NICHT Ziel einer sub-Edge sind UND Trigger sind oder keinen main-Vorgänger haben
├── 4. Für jeden Root-Node:
│   ├── TraceNode erstellen aus RunData[nodeName]
│   ├── Sub-Nodes (Kinder via subEdges) als Nodes[] rekursiv hinzufügen
│   └── Main-Nachfolger: Entweder als Geschwister-Nodes (flach) oder als Kinder (je nach Entscheidung)
├── 5. Branching-Handling:
│   ├── Switch/If-Nodes: Jeder Output-Branch-Index als separater Pfad
│   └── Branch-Index in Metadata speichern
└── 6. Execution-Daten (RunData) mit Nodes matchen:
    └── Für jeden Node: StartTime, Duration, Status, Input/Output aus RunData
```

### 10.3 Connection-Graph-Parsing

Neue Hilfsstruktur:

```go
type ParsedConnection struct {
    SourceNode string
    TargetNode string
    Type       string  // "main", "ai_languageModel", "ai_tool", etc.
    Index      int     // Output-Branch-Index (für Switch/If)
}

type ConnectionGraph struct {
    MainFlow    map[string][]ParsedConnection  // sourceNode → main connections
    SubNodes    map[string][]ParsedConnection  // sourceNode → sub-node connections
    ParentOf    map[string]string              // childNode → parentNode (inverse sub-edges)
    RootNodes   []string                       // Nodes ohne main-Vorgänger und nicht sub-node
}
```

### 10.4 Hierarchie-Strategie

**Option A — Flach mit Sub-Nodes** (Empfohlen):
```
Chat Trigger [workflow]
Respond to Webhook [workflow]
AI Agent [agent]
├── Azure OpenAI Chat Model [llm]     ← sub-node (ai_languageModel)
├── Memory Buffer [custom]             ← sub-node (ai_memory)
└── Search Tool [tool]                 ← sub-node (ai_tool)
```

Main-Flow-Nodes bleiben auf gleicher Ebene (sequentiell). Nur Sub-Nodes werden als Kinder dargestellt.

**Begründung**: Dies spiegelt die N8N-UI-Darstellung wider, wo der Main-Flow horizontal und Sub-Nodes vertikal angehängt sind.

**Option B — Vollständig hierarchisch**:
```
Chat Trigger [workflow]
└── Respond to Webhook [workflow]
    └── AI Agent [agent]
        ├── Azure OpenAI Chat Model [llm]
        └── Search Tool [tool]
```

Gesamter Flow als verschachtelte Hierarchie. Nachteil: Sehr tiefe Verschachtelung bei langen Workflows.

### 10.5 Branching-Beispiel (execution_1716)

```
When chat message received [workflow]
Edit Fields [function]
Switch [conditional]
├── Branch 0:
│   ├── HTTP Request [http]
│   ├── HTML [function]
│   └── Merge [function]               ← Merge-Punkt
│       └── Code [code]
│           └── Merge1 [function]       ← Finaler Merge
└── Branch 1:
    └── Merge1 [function]              ← Gleicher Merge-Punkt
```

Branch-Index wird in `Metadata["branch_index"]` gespeichert.

### 10.6 Ergebnis-Beispiel (execution_119)

```
Aktuell (flach):                      Neu (Option A):
├── Chat Trigger                      ├── Chat Trigger [workflow]
├── Respond to Webhook                ├── Respond to Webhook [workflow]
├── AI Agent                          └── AI Agent [agent]
└── Azure OpenAI Chat Model               └── Azure OpenAI Chat Model [llm]
```

---

## 11. Frontend-Auswirkungen

### 11.1 TracingVisualDialog / Canvas

Das Frontend nutzt die rekursive `TraceNode.Nodes`-Struktur für die Baum-Visualisierung. Die vorgeschlagenen Änderungen liefern **semantisch korrektere** Bäume, was die Darstellung verbessert ohne Frontend-Änderungen zu erfordern.

### 11.2 Potentielle Verbesserungen

| Bereich | Verbesserung |
|---|---|
| Branch-Visualisierung | `branch_index` in Metadata kann für farbcodierte Branches genutzt werden |
| Connection-Type-Anzeige | `connection_type` in Metadata kann als Label an Kanten angezeigt werden |
| Sub-Node-Styling | Nodes mit `connection_type != "main"` können visuell anders dargestellt werden |

---

## 12. Empfehlungen & Priorisierung

### Phase 1 — Kritische Fixes (Empfohlen als erstes)

| # | Aufgabe | Aufwand | Impact |
|---|---|---|---|
| 1 | **N8N: Connection-Graph parsen** — `WorkflowData.Connections` zu typisierter Struktur parsen | Mittel | Hoch |
| 2 | **N8N: Sub-Node-Hierarchie aufbauen** — Non-main-Connections → Parent-Child-Beziehungen | Mittel | Hoch |
| 3 | **Foundry: Action-DAG nutzen** — `previous_action_id`-Kette für sequentielle Abfolge statt response_id-Gruppierung | Hoch | Hoch |
| 4 | **Foundry: Message-Zuordnung verbessern** — Messages korrekt den Actions zuordnen (inkl. Sub-Agent-Messages) | Mittel | Hoch |

### Phase 2 — Verbesserungen

| # | Aufgabe | Aufwand | Impact |
|---|---|---|---|
| 5 | **N8N: Branch-Handling** — Switch/If-Branching mit Index-Tracking | Mittel | Mittel |
| 6 | **N8N: Metadaten erweitern** — `connection_type`, `branch_index` in Metadata | Niedrig | Mittel |
| 7 | **Foundry: Simple vs. Workflow Agent** — Unterschiedliche Transformationslogik je Agent-Typ | Niedrig | Niedrig |

### Phase 3 — Optimierungen

| # | Aufgabe | Aufwand | Impact |
|---|---|---|---|
| 8 | **Tests für neue Hierarchien** — Unit-Tests mit echten POC-Daten als Fixtures | Mittel | Hoch |
| 9 | **Frontend Branch-Visualisierung** — Farbkodierung für Branches | Niedrig | Niedrig |

### Zusammenfassung der Empfehlung

1. **N8N zuerst** — Der Fix ist klarer definiert (Connections-Graph ist eindeutig) und hat sofortigen visuellen Impact
2. **Foundry danach** — Komplexer wegen der Message-Zuordnungs-Logik und fehlender offizieller Dokumentation der action_id-Semantik
3. **Domain-Modell beibehalten** — Das aktuelle `TraceNode`-Modell ist ausreichend flexibel. Erweiterte Informationen über `Metadata`-Map transportieren
4. **POC-Daten als Test-Fixtures** — Die vorhandenen POC-JSON-Dateien sollten als Grundlage für Unit-Test-Fixtures verwendet werden

---

## Anhang: Quelldateien

| Datei | Pfad | Zeilen |
|---|---|---|
| Foundry Transformer | `internal/services/traceimport/foundry/transformer.go` | 772 |
| Foundry Types | `internal/services/traceimport/foundry/types.go` | ~200 |
| N8N Transformer | `internal/services/traceimport/n8n/transformer.go` | 427 |
| N8N Types | `internal/services/traceimport/n8n/types.go` | 327 |
| Domain Models | `internal/domain/models/trace.go` | 351 |
| POC Foundry | `poc/foundry_new/resp_*.json` | 5 Dateien |
| POC N8N | `poc/n8n/tracing/tmp/execution_*.json` | 4 Dateien |
