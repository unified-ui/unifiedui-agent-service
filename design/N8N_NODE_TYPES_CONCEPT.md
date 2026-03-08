# N8N Node Types — AI Automation Context

## Overview

This document catalogs the **30+ most relevant N8N node types** for AI automation workflows and describes their execution data structures as returned by the N8N API endpoint `GET /executions/{id}?includeData=true`.

The goal is to provide a comprehensive reference for the trace import transformer, ensuring every node type that appears in real AI-driven N8N workflows is correctly understood, categorized, and its data extracted.

---

## Table of Contents

1. [API Response Structure](#1-api-response-structure)
2. [Node Execution Structure](#2-node-execution-structure)
3. [Connection Types & Hierarchy](#3-connection-types--hierarchy)
4. [Node Catalog](#4-node-catalog)
   - [4.1 Triggers](#41-triggers)
   - [4.2 Root Nodes (AI Cluster)](#42-root-nodes-ai-cluster)
   - [4.3 Sub-Nodes: LLM Models](#43-sub-nodes-llm-models)
   - [4.4 Sub-Nodes: Memory](#44-sub-nodes-memory)
   - [4.5 Sub-Nodes: Tools](#45-sub-nodes-tools)
   - [4.6 Sub-Nodes: Vector Stores](#46-sub-nodes-vector-stores)
   - [4.7 Sub-Nodes: Embeddings](#47-sub-nodes-embeddings)
   - [4.8 Sub-Nodes: Output Parsers](#48-sub-nodes-output-parsers)
   - [4.9 Sub-Nodes: Document Loaders](#49-sub-nodes-document-loaders)
   - [4.10 Sub-Nodes: Text Splitters](#410-sub-nodes-text-splitters)
   - [4.11 Sub-Nodes: Retrievers](#411-sub-nodes-retrievers)
   - [4.12 Core Nodes (Flow / Data)](#412-core-nodes-flow--data)
5. [Metadata Fields](#5-metadata-fields)
6. [Summary Table](#6-summary-table)
7. [Notes for Transformer Implementation](#7-notes-for-transformer-implementation)

---

## 1. API Response Structure

**Endpoint**: `GET /executions/{id}?includeData=true`

```json
{
  "id": "119",
  "finished": true,
  "mode": "webhook" | "manual" | "trigger",
  "status": "success" | "error" | "waiting" | "running" | "crashed" | "new",
  "createdAt": "2026-01-05T16:07:28.804Z",
  "startedAt": "2026-01-05T16:07:28.807Z",
  "stoppedAt": "2026-01-05T16:07:34.682Z",
  "workflowId": "01V4K8pjRhOVncdg",
  "waitTill": null,
  "data": {
    "resultData": {
      "runData": { "<NodeName>": [ NodeExecution, ... ] },
      "lastNodeExecuted": "AI Agent",
      "error": null
    }
  },
  "workflowData": {
    "id": "01V4K8pjRhOVncdg",
    "name": "My Workflow",
    "nodes": [ WorkflowNode, ... ],
    "connections": { "<SourceNode>": { "<connectionType>": [[ Link, ... ]] } }
  }
}
```

---

## 2. Node Execution Structure

Every node in `runData` is keyed by its **display name** and holds an array of `NodeExecution` objects (one per run; multiple for loops/retries).

```json
{
  "NodeName": [{
    "startTime": 1767629248807,
    "executionTime": 5874,
    "executionStatus": "success" | "error" | "waiting",
    "source": [
      { "previousNode": "Respond to Webhook", "previousNodeRun": 0, "previousNodeOutput": 0 }
    ],
    "hints": [],
    "data": {
      "main": [[ { "json": { ... }, "pairedItem": { "item": 0 } } ]]
    },
    "inputOverride": { ... },
    "metadata": { ... },
    "error": null
  }]
}
```

**Key distinctions:**

| Aspect | Root / Core Nodes | Sub-Nodes (LLM, Tool, Memory, ...) |
|--------|-------------------|-------------------------------------|
| Output data key | `data.main` | `data.<connectionType>` (e.g. `data.ai_languageModel`) |
| Input data | n/a | `inputOverride.<connectionType>` |
| Metadata | optional `tokenUsage`, `subExecution` | always has `subRun: [{ node, runIndex }]` |
| Multi-run | rare (loops) | common (LLM called multiple times in agent loop) |

---

## 3. Connection Types & Hierarchy

### 3.1 All Connection Types

| Connection Type | Meaning | Source → Target |
|---|---|---|
| `main` | Sequential control flow | any node → next node |
| `ai_languageModel` | LLM model sub-node | LLM → Agent/Chain |
| `ai_tool` | Tool sub-node | Tool → Agent |
| `ai_memory` | Memory sub-node | Memory → Agent |
| `ai_outputParser` | Output parser sub-node | Parser → Agent/Chain |
| `ai_retriever` | Retriever sub-node | Retriever → Q&A Chain |
| `ai_document` | Document loader sub-node | DocLoader → Chain/VectorStore |
| `ai_textSplitter` | Text splitter sub-node | Splitter → DocLoader |
| `ai_vectorStore` | Vector store sub-node | VectorStore → Retriever/Tool |
| `ai_embedding` | Embedding model sub-node | Embedding → VectorStore |

### 3.2 Connection Structure

Connections are defined from **source → target**. For non-main connections the source is the sub-node and the target is the parent node:

```json
"connections": {
  "Azure OpenAI Chat Model": {
    "ai_languageModel": [[{
      "node": "AI Agent",
      "type": "ai_languageModel",
      "index": 0
    }]]
  },
  "Memory Buffer": {
    "ai_memory": [[{
      "node": "AI Agent",
      "type": "ai_memory",
      "index": 0
    }]]
  },
  "Search Tool": {
    "ai_tool": [[{
      "node": "AI Agent",
      "type": "ai_tool",
      "index": 0
    }]]
  }
}
```

Structure: `connections[sourceName][connectionType][branchIndex][linkIndex]`

### 3.3 Hierarchy rule

**`main` connections** = sequential flow (no parent-child hierarchy).
**Non-main connections** (`ai_*`) = sub-node attached to parent → creates parent-child trace hierarchy.

---

## 4. Node Catalog

### 4.1 Triggers

#### Chat Trigger

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.chatTrigger` |
| **Category** | trigger |
| **Output key** | `data.main` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "chatInput": "User message text",
        "sessionId": "dc812e23-58c9-4cae-bf11-833925982810"
      }
    }]]
  }
}
```

- `chatInput` — the user's message (may include `<history>...</history>` tags)
- `sessionId` — unique session identifier for multi-turn chat
- `source` is always `[]` (trigger, no predecessor)

#### Manual Trigger

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.manualTrigger` |
| **Category** | trigger |
| **Output key** | `data.main` |

```json
{
  "data": { "main": [[{ "json": {} }]] }
}
```

- Empty JSON output
- Used for development/testing workflow executions

#### Webhook

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.webhook` |
| **Category** | trigger |
| **Output key** | `data.main` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "headers": { ... },
        "params": { ... },
        "query": { ... },
        "body": { ... }
      }
    }]]
  }
}
```

#### Schedule Trigger

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.scheduleTrigger` |
| **Category** | trigger |
| **Output key** | `data.main` |

```json
{
  "data": { "main": [[{ "json": {} }]] }
}
```

#### Form Trigger

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.formTrigger` |
| **Category** | trigger |
| **Output key** | `data.main` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "fieldName": "field value",
        "submittedAt": "2026-01-07T16:08:01.751Z",
        "formMode": "production"
      }
    }]]
  }
}
```

---

### 4.2 Root Nodes (AI Cluster)

#### AI Agent

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.agent` |
| **Category** | agent |
| **Output key** | `data.main` |
| **Sub-connections** | `ai_languageModel`, `ai_tool`, `ai_memory`, `ai_outputParser` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "output": "The AI agent's final response text"
      }
    }]]
  },
  "metadata": {
    "tokenUsage": {
      "completionTokens": 297,
      "promptTokens": 333,
      "totalTokens": 630
    }
  }
}
```

- `output` — final agent response text
- `metadata.tokenUsage` — aggregated token counts from all LLM calls
- The AI Agent is the primary root node; all agent types (Tools Agent, ReAct, Conversational, etc.) share the same node type

#### Basic LLM Chain

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.chainLlm` |
| **Category** | agent |
| **Output key** | `data.main` |
| **Sub-connections** | `ai_languageModel`, `ai_outputParser` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "response": { "text": "Generated text response..." }
      }
    }]]
  }
}
```

- `response.text` — the LLM's response text

#### Question and Answer Chain

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.chainRetrievalQa` |
| **Category** | agent |
| **Output key** | `data.main` |
| **Sub-connections** | `ai_languageModel`, `ai_retriever` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "response": { "text": "Answer based on retrieved documents..." },
        "sourceDocuments": [
          { "pageContent": "...", "metadata": { "source": "..." } }
        ]
      }
    }]]
  }
}
```

- `response.text` — the answer
- `sourceDocuments` — array of retrieved document chunks used

#### Summarization Chain

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.chainSummarization` |
| **Category** | agent |
| **Output key** | `data.main` |
| **Sub-connections** | `ai_languageModel`, `ai_document` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "response": { "text": "Summary of the document..." }
      }
    }]]
  }
}
```

#### Information Extractor

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.information-extractor` |
| **Category** | agent |
| **Output key** | `data.main` |
| **Sub-connections** | `ai_languageModel`, `ai_outputParser` |

```json
{
  "data": {
    "main": [[{
      "json": {
        "output": { "field1": "value1", "field2": "value2" }
      }
    }]]
  }
}
```

- `output` — structured data matching a defined schema

#### Text Classifier

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.text-classifier` |
| **Category** | agent |
| **Output key** | `data.main` (multiple branches) |
| **Sub-connections** | `ai_languageModel` |

```json
{
  "data": {
    "main": [
      [{ "json": { "classification": "category_1" } }],
      [],
      []
    ]
  }
}
```

- Each output branch corresponds to a classification category
- Only the matching branch contains items

#### Sentiment Analysis

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.sentimentAnalysis` |
| **Category** | agent |
| **Output key** | `data.main` (multiple branches: positive/negative/neutral) |
| **Sub-connections** | `ai_languageModel` |

```json
{
  "data": {
    "main": [
      [{ "json": { "sentiment": "positive" } }],
      [],
      []
    ]
  }
}
```

#### LangChain Code

| Field | Value |
|-------|-------|
| **Type** | `@n8n/n8n-nodes-langchain.code` |
| **Category** | code |
| **Output key** | `data.main` |
| **Sub-connections** | `ai_languageModel`, `ai_tool`, `ai_memory`, `ai_outputParser` |

```json
{
  "data": {
    "main": [[{
      "json": { ... }
    }]]
  }
}
```

- Output depends on user-written LangChain code

---

### 4.3 Sub-Nodes: LLM Models

All LLM sub-nodes share the same RunData pattern. They output via `data.ai_languageModel` and receive input via `inputOverride.ai_languageModel`.

#### Common LLM RunData Structure

```json
{
  "startTime": 1767629248814,
  "executionTime": 5864,
  "executionStatus": "success",
  "source": [{ "previousNode": "AI Agent" }],
  "data": {
    "ai_languageModel": [[{
      "json": {
        "response": {
          "generations": [[{
            "text": "The LLM response text...",
            "generationInfo": {
              "prompt": 0,
              "completion": 0,
              "finish_reason": "stop",
              "system_fingerprint": null,
              "model_name": "gpt-5-mini-2025-08-07"
            }
          }]]
        },
        "tokenUsage": {
          "completionTokens": 297,
          "promptTokens": 333,
          "totalTokens": 630
        }
      }
    }]]
  },
  "inputOverride": {
    "ai_languageModel": [[{
      "json": {
        "messages": [ "Human: What is 1+1?" ],
        "estimatedTokens": 329,
        "options": {
          "model": "gpt-5-mini",
          "timeout": 60000,
          "max_retries": 2,
          "configuration": { "fetchOptions": {} }
        }
      }
    }]]
  },
  "metadata": {
    "subRun": [{ "node": "Azure OpenAI Chat Model", "runIndex": 0 }]
  }
}
```

**Key fields:**

| Path | Description |
|------|-------------|
| `data.ai_languageModel[0][0].json.response.generations[0][0].text` | Raw LLM output |
| `data.ai_languageModel[0][0].json.response.generations[0][0].generationInfo` | Model metadata (finish_reason, model_name) |
| `data.ai_languageModel[0][0].json.tokenUsage` | Token counts |
| `inputOverride.ai_languageModel[0][0].json.messages` | Prompt messages sent |
| `inputOverride.ai_languageModel[0][0].json.options.model` | Model name |
| `metadata.subRun` | Self-identification as sub-node |

#### LLM Node Types

| Name | Type Identifier | Provider-specific |
|------|-----------------|-------------------|
| OpenAI Chat Model | `@n8n/n8n-nodes-langchain.lmChatOpenAi` | `model_name`: `gpt-4o`, `gpt-4o-mini`, etc. |
| Azure OpenAI Chat Model | `@n8n/n8n-nodes-langchain.lmChatAzureOpenAi` | `model_name`: deployment name |
| Anthropic Chat Model | `@n8n/n8n-nodes-langchain.lmChatAnthropic` | `model_name`: `claude-sonnet-4-...`, may use `stop_reason` |
| Google Gemini Chat Model | `@n8n/n8n-nodes-langchain.lmChatGoogleGemini` | `model_name`: `gemini-2.0-flash`, different generationInfo |
| Groq Chat Model | `@n8n/n8n-nodes-langchain.lmChatGroq` | Similar to OpenAI format |
| DeepSeek Chat Model | `@n8n/n8n-nodes-langchain.lmChatDeepSeek` | Similar to OpenAI format |
| Mistral Cloud Chat Model | `@n8n/n8n-nodes-langchain.lmChatMistralCloud` | Similar to OpenAI format |
| Ollama Chat Model | `@n8n/n8n-nodes-langchain.lmChatOllama` | Local model name, no system_fingerprint |
| AWS Bedrock Chat Model | `@n8n/n8n-nodes-langchain.lmChatAwsBedrock` | AWS-specific model names |
| OpenRouter Chat Model | `@n8n/n8n-nodes-langchain.lmChatOpenRouter` | Routing to various models |
| xAI Grok Chat Model | `@n8n/n8n-nodes-langchain.lmChatXaiGrok` | `model_name`: `grok-*` |

**Important**: LLMs can run **multiple times** per agent execution (one per tool-calling loop iteration). Each run is a separate entry in the NodeExecution array.

---

### 4.4 Sub-Nodes: Memory

#### Common Memory RunData Structure

All memory sub-nodes output via `data.ai_memory` and run **twice** per agent execution: once to load history (before LLM) and once to save (after LLM).

```json
{
  "data": {
    "ai_memory": [[{
      "json": {
        "action": "load" | "save",
        "chatHistory": [
          { "type": "human", "data": { "content": "Hello" } },
          { "type": "ai", "data": { "content": "Hi there!" } }
        ]
      }
    }]]
  },
  "metadata": {
    "subRun": [{ "node": "Memory Buffer", "runIndex": 0 }]
  }
}
```

#### Memory Node Types

| Name | Type Identifier |
|------|-----------------|
| Simple Memory (Buffer Window) | `@n8n/n8n-nodes-langchain.memoryBufferWindow` |
| Postgres Chat Memory | `@n8n/n8n-nodes-langchain.memoryPostgresChat` |
| Redis Chat Memory | `@n8n/n8n-nodes-langchain.memoryRedisChat` |
| MongoDB Chat Memory | `@n8n/n8n-nodes-langchain.memoryMongoDbChat` |
| Motorhead Memory | `@n8n/n8n-nodes-langchain.memoryMotorhead` |
| Zep Memory | `@n8n/n8n-nodes-langchain.memoryZep` |
| Xata Memory | `@n8n/n8n-nodes-langchain.memoryXata` |
| Chat Memory Manager | `@n8n/n8n-nodes-langchain.memoryManager` |

---

### 4.5 Sub-Nodes: Tools

#### Common Tool RunData Structure

All tools output via `data.ai_tool`.

```json
{
  "data": {
    "ai_tool": [[{
      "json": {
        "response": "Tool execution result text..."
      }
    }]]
  },
  "metadata": {
    "subRun": [{ "node": "Search Tool", "runIndex": 0 }]
  }
}
```

#### Tool Node Types

| Name | Type Identifier | Notes |
|------|-----------------|-------|
| Call n8n Workflow Tool | `@n8n/n8n-nodes-langchain.toolWorkflow` | Has `metadata.subExecution` with `workflowId`/`executionId` |
| Custom Code Tool | `@n8n/n8n-nodes-langchain.toolCode` | Returns user-defined output |
| MCP Client Tool | `@n8n/n8n-nodes-langchain.toolMcp` | Model Context Protocol tool |
| SerpApi (Google Search) | `@n8n/n8n-nodes-langchain.toolSerpApi` | Search results as text |
| Wikipedia | `@n8n/n8n-nodes-langchain.toolWikipedia` | Wikipedia article content |
| Think Tool | `@n8n/n8n-nodes-langchain.toolThink` | Agent's internal reasoning text |
| Vector Store Q&A Tool | `@n8n/n8n-nodes-langchain.toolVectorStore` | Retrieved documents as answer |
| Calculator | `@n8n/n8n-nodes-langchain.toolCalculator` | Math computation result |
| AI Agent Tool | `@n8n/n8n-nodes-langchain.toolAiAgent` | Nested agent as tool |
| SearXNG Tool | `@n8n/n8n-nodes-langchain.toolSearxng` | Privacy-focused search |
| Wolfram Alpha | `@n8n/n8n-nodes-langchain.toolWolframAlpha` | Computational knowledge |

**Workflow Tool special metadata:**

```json
{
  "metadata": {
    "subRun": [...],
    "subExecution": {
      "workflowId": "abc123",
      "executionId": "789"
    }
  }
}
```

---

### 4.6 Sub-Nodes: Vector Stores

Output key depends on mode:
- **Retrieval mode** (via retriever): `data.ai_vectorStore`
- **Insert mode** (with document loaders): `data.main`

#### Retrieval Mode

```json
{
  "data": {
    "ai_vectorStore": [[{
      "json": {
        "documents": [
          { "pageContent": "...", "metadata": { "source": "..." } }
        ]
      }
    }]]
  }
}
```

#### Insert Mode

```json
{
  "data": {
    "main": [[{
      "json": {
        "success": true,
        "documentsInserted": 15
      }
    }]]
  }
}
```

#### Vector Store Node Types

| Name | Type Identifier |
|------|-----------------|
| Pinecone | `@n8n/n8n-nodes-langchain.vectorStorePinecone` |
| PGVector | `@n8n/n8n-nodes-langchain.vectorStorePgVector` |
| Qdrant | `@n8n/n8n-nodes-langchain.vectorStoreQdrant` |
| Simple (In-Memory) | `@n8n/n8n-nodes-langchain.vectorStoreInMemory` |
| Chroma | `@n8n/n8n-nodes-langchain.vectorStoreChroma` |
| Supabase | `@n8n/n8n-nodes-langchain.vectorStoreSupabase` |
| Redis | `@n8n/n8n-nodes-langchain.vectorStoreRedis` |
| Weaviate | `@n8n/n8n-nodes-langchain.vectorStoreWeaviate` |
| MongoDB Atlas | `@n8n/n8n-nodes-langchain.vectorStoreMongoDbAtlas` |
| Milvus | `@n8n/n8n-nodes-langchain.vectorStoreMilvus` |
| Azure AI Search | `@n8n/n8n-nodes-langchain.vectorStoreAzureAiSearch` |
| Zep | `@n8n/n8n-nodes-langchain.vectorStoreZep` |

---

### 4.7 Sub-Nodes: Embeddings

Output via `data.ai_embedding`. Often **not visible** in runData (consumed internally by vector stores).

```json
{
  "data": {
    "ai_embedding": [[{
      "json": {
        "embedding": [0.0123, -0.0456, ...],
        "tokenUsage": {
          "promptTokens": 15,
          "totalTokens": 15
        }
      }
    }]]
  }
}
```

#### Embedding Node Types

| Name | Type Identifier |
|------|-----------------|
| OpenAI Embeddings | `@n8n/n8n-nodes-langchain.embeddingsOpenAi` |
| Azure OpenAI Embeddings | `@n8n/n8n-nodes-langchain.embeddingsAzureOpenAi` |
| Google Gemini Embeddings | `@n8n/n8n-nodes-langchain.embeddingsGoogleGemini` |
| Google Vertex Embeddings | `@n8n/n8n-nodes-langchain.embeddingsGoogleVertex` |
| Cohere Embeddings | `@n8n/n8n-nodes-langchain.embeddingsCohere` |
| Ollama Embeddings | `@n8n/n8n-nodes-langchain.embeddingsOllama` |
| Mistral Cloud Embeddings | `@n8n/n8n-nodes-langchain.embeddingsMistralCloud` |
| HuggingFace Inference Embeddings | `@n8n/n8n-nodes-langchain.embeddingsHuggingFaceInference` |
| AWS Bedrock Embeddings | `@n8n/n8n-nodes-langchain.embeddingsAwsBedrock` |

---

### 4.8 Sub-Nodes: Output Parsers

Output via `data.ai_outputParser`. Often **not visible** in runData (consumed internally by chain/agent).

```json
{
  "data": {
    "ai_outputParser": [[{
      "json": {
        "parsed": { ... }
      }
    }]]
  }
}
```

#### Output Parser Node Types

| Name | Type Identifier |
|------|-----------------|
| Structured Output Parser | `@n8n/n8n-nodes-langchain.outputParserStructured` |
| Auto-fixing Output Parser | `@n8n/n8n-nodes-langchain.outputParserAutofixing` |
| Item List Output Parser | `@n8n/n8n-nodes-langchain.outputParserItemList` |

---

### 4.9 Sub-Nodes: Document Loaders

Output via `data.ai_document`.

```json
{
  "data": {
    "ai_document": [[{
      "json": {
        "pageContent": "Document text content...",
        "metadata": { "source": "input" }
      }
    }]]
  }
}
```

#### Document Loader Node Types

| Name | Type Identifier |
|------|-----------------|
| Default Data Loader | `@n8n/n8n-nodes-langchain.documentDefaultDataLoader` |
| GitHub Document Loader | `@n8n/n8n-nodes-langchain.documentGithubLoader` |

---

### 4.10 Sub-Nodes: Text Splitters

Output via `data.ai_textSplitter`. Typically **not visible** in runData (internal to document loaders).

#### Text Splitter Node Types

| Name | Type Identifier |
|------|-----------------|
| Recursive Character Text Splitter | `@n8n/n8n-nodes-langchain.textSplitterRecursiveCharacterTextSplitter` |
| Character Text Splitter | `@n8n/n8n-nodes-langchain.textSplitterCharacterTextSplitter` |
| Token Splitter | `@n8n/n8n-nodes-langchain.textSplitterTokenSplitter` |

---

### 4.11 Sub-Nodes: Retrievers

Output via `data.ai_retriever`.

```json
{
  "data": {
    "ai_retriever": [[{
      "json": {
        "documents": [
          { "pageContent": "...", "metadata": { "source": "..." } }
        ]
      }
    }]]
  }
}
```

#### Retriever Node Types

| Name | Type Identifier |
|------|-----------------|
| Vector Store Retriever | `@n8n/n8n-nodes-langchain.retrieverVectorStore` |
| Workflow Retriever | `@n8n/n8n-nodes-langchain.retrieverWorkflow` |
| MultiQuery Retriever | `@n8n/n8n-nodes-langchain.retrieverMultiQuery` |
| Contextual Compression Retriever | `@n8n/n8n-nodes-langchain.retrieverContextualCompression` |

---

### 4.12 Core Nodes (Flow / Data)

All core nodes output via `data.main`.

#### HTTP Request

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.httpRequest` |
| **Category** | tool |

```json
{
  "data": {
    "main": [[{
      "json": { ... }
    }]]
  }
}
```

- JSON responses parsed directly into `json`
- Binary responses may appear in `binary` field

#### Code (JavaScript / Python)

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.code` |
| **Category** | code |

```json
{
  "data": {
    "main": [[{
      "json": { ... },
      "pairedItem": { "item": 0 }
    }]]
  }
}
```

- Output depends on user-written code

#### Edit Fields (Set)

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.set` |
| **Category** | custom |

```json
{
  "data": {
    "main": [[{
      "json": { "field1": "value1", "field2": "value2" }
    }]]
  }
}
```

#### If

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.if` |
| **Category** | conditional |
| **Output** | Two branches |

```json
{
  "data": {
    "main": [
      [{ "json": { ... } }],
      []
    ]
  }
}
```

- Branch 0: true condition
- Branch 1: false condition

#### Switch

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.switch` |
| **Category** | conditional |
| **Output** | Multiple branches |

```json
{
  "data": {
    "main": [
      [{ "json": { ... } }],
      [],
      [],
      []
    ]
  }
}
```

- Only matching branch(es) contain items

#### Merge

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.merge` |
| **Category** | custom |

```json
{
  "data": {
    "main": [[
      { "json": { ... } },
      { "json": { ... } }
    ]]
  }
}
```

#### Execute Sub-workflow

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.executeWorkflow` |
| **Category** | tool |

```json
{
  "data": {
    "main": [[{ "json": { ... } }]]
  },
  "metadata": {
    "subExecution": {
      "workflowId": "subWorkflowId",
      "executionId": "subExecutionId"
    }
  }
}
```

#### Respond to Webhook

| Field | Value |
|-------|-------|
| **Type** | `n8n-nodes-base.respondToWebhook` |
| **Category** | custom |

- Passes through input data unchanged

#### Other Core Nodes

| Name | Type Identifier | Category |
|------|-----------------|----------|
| Execute Command | `n8n-nodes-base.executeCommand` | tool |
| Postgres | `n8n-nodes-base.postgres` | tool |
| MongoDB | `n8n-nodes-base.mongoDb` | tool |
| MySQL | `n8n-nodes-base.mySql` | tool |
| Redis | `n8n-nodes-base.redis` | tool |
| Function | `n8n-nodes-base.function` | code |
| Function Item | `n8n-nodes-base.functionItem` | code |
| NoOp | `n8n-nodes-base.noOp` | custom |
| HTML | `n8n-nodes-base.html` | custom |
| Form | `n8n-nodes-base.form` | form |
| Filter | `n8n-nodes-base.filter` | conditional |
| Loop Over Items | `n8n-nodes-base.splitInBatches` | custom |
| Read/Write Files | `n8n-nodes-base.readWriteFile` | tool |
| Aggregate | `n8n-nodes-base.aggregate` | custom |
| AI Transform | `n8n-nodes-base.aiTransform` | custom |
| Wait | `n8n-nodes-base.wait` | custom |

---

## 5. Metadata Fields

### 5.1 tokenUsage

Appears on **LLM sub-nodes** and may be **aggregated on Agent root nodes**.

```json
{
  "tokenUsage": {
    "completionTokens": 297,
    "promptTokens": 333,
    "totalTokens": 630
  }
}
```

**Location on LLM sub-node**: `data.ai_languageModel[0][0].json.tokenUsage`
**Location on Agent**: `metadata.tokenUsage` (aggregated from all LLM calls)

### 5.2 subRun

Identifies a sub-node execution. Present on every sub-node.

```json
{
  "metadata": {
    "subRun": [{ "node": "Azure OpenAI Chat Model", "runIndex": 0 }]
  }
}
```

### 5.3 subExecution

Links to a separate N8N execution (for Execute Sub-workflow and Workflow Tool nodes).

```json
{
  "metadata": {
    "subExecution": {
      "workflowId": "abc123",
      "executionId": "789"
    }
  }
}
```

### 5.4 source

Present on every non-trigger node, shows execution lineage:

```json
{
  "source": [{
    "previousNode": "Respond to Webhook",
    "previousNodeRun": 0,
    "previousNodeOutput": 0
  }]
}
```

- Trigger nodes have `source: []`
- `previousNodeOutput` indicates which branch triggered this node (relevant for Switch/If)

### 5.5 pairedItem

Links output items to corresponding input items:

```json
{
  "pairedItem": { "item": 0 }
}
```

- Can also be an array: `[{ "item": 0 }, { "item": 1 }]`

---

## 6. Summary Table

| # | Category | Node Type Identifier | Display Name | Output Key | Key JSON Fields |
|---|----------|---------------------|--------------|------------|-----------------|
| 1 | trigger | `@n8n/n8n-nodes-langchain.chatTrigger` | Chat Trigger | `data.main` | `chatInput`, `sessionId` |
| 2 | trigger | `n8n-nodes-base.manualTrigger` | Manual Trigger | `data.main` | `{}` |
| 3 | trigger | `n8n-nodes-base.webhook` | Webhook | `data.main` | `headers`, `params`, `query`, `body` |
| 4 | trigger | `n8n-nodes-base.scheduleTrigger` | Schedule Trigger | `data.main` | `{}` |
| 5 | trigger | `n8n-nodes-base.formTrigger` | Form Trigger | `data.main` | form fields, `submittedAt` |
| 6 | agent | `@n8n/n8n-nodes-langchain.agent` | AI Agent | `data.main` | `output` |
| 7 | agent | `@n8n/n8n-nodes-langchain.chainLlm` | Basic LLM Chain | `data.main` | `response.text` |
| 8 | agent | `@n8n/n8n-nodes-langchain.chainRetrievalQa` | Q&A Chain | `data.main` | `response.text`, `sourceDocuments` |
| 9 | agent | `@n8n/n8n-nodes-langchain.chainSummarization` | Summarization Chain | `data.main` | `response.text` |
| 10 | agent | `@n8n/n8n-nodes-langchain.information-extractor` | Information Extractor | `data.main` | `output` (structured) |
| 11 | agent | `@n8n/n8n-nodes-langchain.text-classifier` | Text Classifier | `data.main` (branches) | `classification` |
| 12 | agent | `@n8n/n8n-nodes-langchain.sentimentAnalysis` | Sentiment Analysis | `data.main` (branches) | `sentiment` |
| 13 | code | `@n8n/n8n-nodes-langchain.code` | LangChain Code | `data.main` | user-defined |
| 14 | llm | `@n8n/n8n-nodes-langchain.lmChatOpenAi` | OpenAI Chat Model | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 15 | llm | `@n8n/n8n-nodes-langchain.lmChatAzureOpenAi` | Azure OpenAI Chat | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 16 | llm | `@n8n/n8n-nodes-langchain.lmChatAnthropic` | Anthropic Chat Model | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 17 | llm | `@n8n/n8n-nodes-langchain.lmChatGoogleGemini` | Google Gemini Chat | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 18 | llm | `@n8n/n8n-nodes-langchain.lmChatGroq` | Groq Chat Model | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 19 | llm | `@n8n/n8n-nodes-langchain.lmChatDeepSeek` | DeepSeek Chat | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 20 | llm | `@n8n/n8n-nodes-langchain.lmChatMistralCloud` | Mistral Cloud Chat | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 21 | llm | `@n8n/n8n-nodes-langchain.lmChatOllama` | Ollama Chat Model | `data.ai_languageModel` | `response.generations`, `tokenUsage` |
| 22 | memory | `@n8n/n8n-nodes-langchain.memoryBufferWindow` | Simple Memory | `data.ai_memory` | `action`, `chatHistory` |
| 23 | tool | `@n8n/n8n-nodes-langchain.toolWorkflow` | Workflow Tool | `data.ai_tool` | `response` + `metadata.subExecution` |
| 24 | tool | `@n8n/n8n-nodes-langchain.toolCode` | Custom Code Tool | `data.ai_tool` | `response` |
| 25 | tool | `@n8n/n8n-nodes-langchain.toolMcp` | MCP Client Tool | `data.ai_tool` | `response` |
| 26 | tool | `@n8n/n8n-nodes-langchain.toolThink` | Think Tool | `data.ai_tool` | `response` (reasoning) |
| 27 | tool | `n8n-nodes-base.httpRequest` | HTTP Request | `data.main` | response body |
| 28 | code | `n8n-nodes-base.code` | Code | `data.main` | user-defined |
| 29 | conditional | `n8n-nodes-base.if` | If | `data.main` (2 branches) | pass-through |
| 30 | conditional | `n8n-nodes-base.switch` | Switch | `data.main` (N branches) | pass-through |
| 31 | custom | `n8n-nodes-base.merge` | Merge | `data.main` | merged items |
| 32 | tool | `n8n-nodes-base.executeWorkflow` | Execute Sub-workflow | `data.main` | sub-output + `metadata.subExecution` |
| 33 | custom | `n8n-nodes-base.set` | Edit Fields | `data.main` | set fields |
| 34 | custom | `n8n-nodes-base.respondToWebhook` | Respond to Webhook | `data.main` | pass-through |

---

## 7. Notes for Transformer Implementation

### 7.1 Sub-nodes not always in RunData

Embeddings, text splitters, and output parsers are frequently **not visible** in runData because they are consumed internally by their parent. The transformer must not fail on missing sub-nodes.

### 7.2 Memory runs twice

Memory nodes execute **twice** per agent call: once for `load` (before LLM) and once for `save` (after LLM). Both runs appear in the execution array.

### 7.3 LLM multi-run pattern

When an agent uses tool-calling, the LLM sub-node may produce **multiple runs** (one per iteration in the tool-calling loop). Each is a separate entry in the `NodeExecution` array.

### 7.4 Connection type → hierarchy

`main` connections = sequential flow, no parent-child.
`ai_*` connections = sub-node of the target → parent-child in trace hierarchy.

### 7.5 Binary data

Nodes like HTTP Request and Read/Write Files can produce binary data:

```json
{
  "binary": {
    "data": {
      "mimeType": "application/pdf",
      "data": "base64...",
      "fileName": "document.pdf",
      "fileSize": "12345"
    }
  }
}
```

### 7.6 pairedItem type variance

`pairedItem` can be either `{ "item": N }` (object) or `[{ "item": N }]` (array). The transformer should handle both.

### 7.7 Category mapping

The `GetNodeCategory()` function in `types.go` should be extended to cover all node types listed in this document. Recommended categories:

| Category | Matches |
|----------|---------|
| `trigger` | `*Trigger*`, `*trigger*` |
| `agent` | `agent`, `chain*`, `information-extractor`, `text-classifier`, `sentimentAnalysis` |
| `llm` | `lmChat*` |
| `memory` | `memory*`, `Memory*` |
| `tool` | `tool*`, `Tool*`, `httpRequest`, `executeWorkflow`, `postgres`, `mongo*`, `mysql`, `redis` |
| `code` | `code`, `function*` |
| `conditional` | `switch`, `if`, `filter`, `merge` |
| `vectorStore` | `vectorStore*` |
| `embedding` | `embeddings*` |
| `retriever` | `retriever*` |
| `outputParser` | `outputParser*` |
| `document` | `document*` |
| `textSplitter` | `textSplitter*` |
| `form` | `form*`, `Form*` |
| `custom` | everything else |
