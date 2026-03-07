# N8N Automation Nodes — Integration Concept

## Overview

This document catalogs **~78 frequently used N8N automation nodes** that are **not yet covered** by the existing trace import transformer. While the first concept ([N8N_NODE_TYPES_CONCEPT.md](./N8N_NODE_TYPES_CONCEPT.md)) focused on AI/LangChain cluster nodes and core flow nodes, this document covers **SaaS integrations, messaging, CRM, file storage, databases, message queues, e-commerce, developer tools, and additional core utility nodes**.

### Key Difference from AI Nodes

All automation nodes documented here use the standard `main` output connection type — unlike AI sub-nodes which use specialized connections (`ai_languageModel`, `ai_tool`, etc.). This means:

- **No new `NodeOutputData` fields needed** — the existing `Main [][]NodeOutputItem` handles all output
- **No new connection graph parsing needed** — these nodes appear as regular (non-sub) nodes
- **The existing `extractOutputData()` already handles them correctly** via `GetOutputItems()` cascade

The main implementation work involves:
1. Adding new node type constants to `types.go`
2. Updating `GetNodeCategory()` with new suffix patterns
3. Updating `mapNodeType()` with new suffix-to-NodeType mappings
4. Optionally adding new `NodeType` domain constants for better frontend visualization

---

## Table of Contents

1. [Currently Covered Nodes](#1-currently-covered-nodes)
2. [New Node Catalog](#2-new-node-catalog)
   - [2.1 Communication & Messaging](#21-communication--messaging)
   - [2.2 Productivity & Spreadsheets](#22-productivity--spreadsheets)
   - [2.3 Project Management](#23-project-management)
   - [2.4 CRM & Sales](#24-crm--sales)
   - [2.5 File Storage & Cloud](#25-file-storage--cloud)
   - [2.6 Developer Tools & DevOps](#26-developer-tools--devops)
   - [2.7 Databases (Additional)](#27-databases-additional)
   - [2.8 Message Queues & Event Streaming](#28-message-queues--event-streaming)
   - [2.9 E-Commerce & Payments](#29-e-commerce--payments)
   - [2.10 Customer Support & ITSM](#210-customer-support--itsm)
   - [2.11 Marketing & Email](#211-marketing--email)
   - [2.12 Data Transformation & Utility](#212-data-transformation--utility)
   - [2.13 File I/O & Binary Operations](#213-file-io--binary-operations)
   - [2.14 Workflow Data & Triggers](#214-workflow-data--triggers)
   - [2.15 Microsoft & Azure](#215-microsoft--azure)
3. [Output Data Structure](#3-output-data-structure)
4. [New NodeType Domain Constants](#4-new-nodetype-domain-constants)
5. [GetNodeCategory Updates](#5-getnodecategory-updates)
6. [mapNodeType Updates](#6-mapnodetype-updates)
7. [Summary Table](#7-summary-table)
8. [Implementation Recommendations](#8-implementation-recommendations)
   - [8.7 Default Fallback & Error Resilience](#87-default-fallback--error-resilience)

---

## 1. Currently Covered Nodes

The transformer currently defines **100 node type constants** in `types.go`:

| Category | Count | Examples |
|----------|-------|---------|
| LangChain AI (agents, chains, LLMs, memory, tools, vector stores, embeddings, parsers, loaders, splitters, retrievers) | 71 | `@n8n/n8n-nodes-langchain.agent`, `.lmChatOpenAi`, `.toolWorkflow` |
| Triggers | 5 | `chatTrigger`, `manualTrigger`, `webhook`, `formTrigger`, `scheduleTrigger` |
| Core (flow/data) | 19 | `httpRequest`, `code`, `set`, `switch`, `if`, `merge`, `aggregate` |
| Database | 4 | `postgres`, `mongoDb`, `mySql`, `redis` |
| Form | 1 | `form` |

**NOT covered**: SaaS app nodes, additional triggers, messaging, CRM, storage, queues, e-commerce, marketing, support, additional databases, and data transformation utilities.

---

## 2. New Node Catalog

### 2.1 Communication & Messaging

Messaging and email nodes are among the most frequently used in N8N workflows — especially in AI agent automation where results are delivered to users via chat or email.

#### Slack
- **Type**: `n8n-nodes-base.slack`
- **Trigger**: `n8n-nodes-base.slackTrigger`
- **Operations**: Channel (archive, create, get, invite, join, kick, rename), File (get, upload), Message (send, update, delete, search, getPermalink), Reaction (add/get/remove), Star (add/delete/list), User (get, list), UserGroup (create, update, enable, disable, list)
- **Output**: `{ "ok": true, "channel": "C0123456", "ts": "1234567890.123456", "message": { "text": "...", "type": "message" } }`
- **Relevance**: Top-1 chat integration for AI agent workflows (chatbot responses, notifications, approvals)

#### Discord
- **Type**: `n8n-nodes-base.discord`
- **Operations**: Channel (get, getAll, create, update, delete), Message (send, getAll, delete), Member (getAll, roleAdd, roleRemove)
- **Output**: `{ "id": "...", "content": "...", "channel_id": "...", "author": { ... } }`
- **Relevance**: Popular for community automation and AI bot interactions

#### Telegram
- **Type**: `n8n-nodes-base.telegram`
- **Trigger**: `n8n-nodes-base.telegramTrigger`
- **Operations**: Message (sendMessage, sendPhoto, sendDocument, sendSticker, sendVideo, editMessage, deleteMessage, pinMessage), Chat (get, getAdministrators, getMember, setDescription, setTitle, leaveChat), Callback (answerQuery), File (get)
- **Output**: `{ "ok": true, "result": { "message_id": 123, "chat": { "id": -100123 }, "text": "..." } }`
- **Relevance**: Very popular for AI chatbot workflows

#### Microsoft Teams
- **Type**: `n8n-nodes-base.microsoftTeams`
- **Trigger**: `n8n-nodes-base.microsoftTeamsTrigger`
- **Operations**: Channel (create, get, getAll, update), ChannelMessage (create, getAll), ChatMessage (create, get, getAll), Chat (create, get, getAll), Task (create, delete, get, getAll, update)
- **Output**: `{ "id": "...", "body": { "content": "...", "contentType": "html" }, "from": { ... } }`
- **Relevance**: Primary team messaging integration in enterprise AI workflows

#### Gmail
- **Type**: `n8n-nodes-base.gmail`
- **Trigger**: `n8n-nodes-base.gmailTrigger`
- **Operations**: Draft (create, delete, get, getAll), Label (create, delete, get, getAll), Message (delete, get, getAll, reply, send), MessageLabel (add, remove), Thread (addLabel, delete, get, getAll, removeLabel, reply, trash, untrash)
- **Output**: `{ "id": "...", "threadId": "...", "labelIds": [...], "snippet": "...", "payload": { "headers": [...] } }`
- **Relevance**: Email-based AI workflows, automated responses, email processing

#### Microsoft Outlook
- **Type**: `n8n-nodes-base.microsoftOutlook`
- **Trigger**: `n8n-nodes-base.microsoftOutlookTrigger`
- **Operations**: Calendar (create, delete, get, getAll, update), Contact (create, delete, get, getAll, update), Draft (create, delete, get, send, update), Event (create, delete, get, getAll, update), Folder (create, delete, get, getAll, update), Message (delete, get, getAll, move, reply, send, update)
- **Output**: `{ "id": "...", "subject": "...", "bodyPreview": "...", "from": { "emailAddress": { ... } } }`

#### Send Email (SMTP)
- **Type**: `n8n-nodes-base.sendEmail`
- **Operations**: Send email via SMTP with attachments
- **Output**: `{ "accepted": ["recipient@example.com"], "response": "250 OK", "messageId": "..." }`

#### Twilio
- **Type**: `n8n-nodes-base.twilio`
- **Trigger**: `n8n-nodes-base.twilioTrigger`
- **Operations**: SMS (send), MMS (send), Call (make)
- **Output**: `{ "sid": "...", "status": "queued", "from": "+1...", "to": "+1...", "body": "..." }`
- **Relevance**: SMS/voice integration in AI agent workflows

#### WhatsApp Business Cloud
- **Type**: `n8n-nodes-base.whatsApp`
- **Trigger**: `n8n-nodes-base.whatsAppTrigger`
- **Operations**: Message (send text, image, document, template, interactive)
- **Output**: `{ "messaging_product": "whatsapp", "contacts": [...], "messages": [{ "id": "wamid.." }] }`

---

### 2.2 Productivity & Spreadsheets

#### Google Sheets
- **Type**: `n8n-nodes-base.googleSheets`
- **Trigger**: `n8n-nodes-base.googleSheetsTrigger`
- **Operations**: Document (create, delete), Sheet (append, clear, create, delete, getAll, read, remove, update)
- **Output**: Row data as JSON with column headers as keys: `{ "row_number": 2, "Name": "John", "Email": "john@example.com" }`
- **Relevance**: Extremely common data source/destination in AI workflows (log results, read prompts, store evaluations)

#### Airtable
- **Type**: `n8n-nodes-base.airtable`
- **Trigger**: `n8n-nodes-base.airtableTrigger`
- **Operations**: Record (create, delete, get, getMany, search, update, upsert)
- **Output**: `{ "id": "rec...", "createdTime": "...", "fields": { "Name": "...", "Status": "..." } }`
- **Relevance**: Popular as structured data backend for AI agents

#### Notion
- **Type**: `n8n-nodes-base.notion`
- **Trigger**: `n8n-nodes-base.notionTrigger`
- **Operations**: Database (get, getAll, search), Page (archive, create, search), DatabasePage (create, get, getAll, update), Block (append, getAll)
- **Output**: `{ "id": "...", "object": "page", "properties": { ... }, "url": "https://notion.so/..." }`
- **Relevance**: Knowledge base integration for AI agent workflows

#### Microsoft Excel 365
- **Type**: `n8n-nodes-base.microsoftExcel`
- **Operations**: Table (addRow, getColumns, getRows, lookup), Workbook (getAll), Worksheet (getAll, getContent)
- **Output**: Row data as JSON with column headers as keys (similar to Google Sheets)

#### Google Docs
- **Type**: `n8n-nodes-base.googleDocs`
- **Operations**: Document (create, get, update)
- **Output**: `{ "documentId": "...", "title": "...", "body": { ... } }`

#### Google Calendar
- **Type**: `n8n-nodes-base.googleCalendar`
- **Trigger**: `n8n-nodes-base.googleCalendarTrigger`
- **Operations**: Calendar (get, getAll), Event (create, delete, get, getAll, update)
- **Output**: `{ "id": "...", "summary": "Meeting", "start": { "dateTime": "..." }, "end": { "dateTime": "..." } }`

---

### 2.3 Project Management

#### Jira Software
- **Type**: `n8n-nodes-base.jira`
- **Trigger**: `n8n-nodes-base.jiraTrigger`
- **Operations**: Issue (create, delete, get, getAll, update, changelog, notify, transition), IssueComment (add, get, getAll, remove, update), User (create, delete, get)
- **Output**: `{ "id": "10001", "key": "PROJ-123", "fields": { "summary": "...", "status": { "name": "In Progress" } } }`
- **Relevance**: AI agents creating/updating tickets, sprint automation

#### Trello
- **Type**: `n8n-nodes-base.trello`
- **Trigger**: `n8n-nodes-base.trelloTrigger`
- **Operations**: Attachment (create, delete, get, getAll), Board (create, delete, get, update), Card (create, delete, get, update), Checklist (create, delete, getAll, item operations), Label (addToCard, create, delete, getAll, removeFromCard, update), List (archive, create, get, getAll, update)
- **Output**: `{ "id": "...", "name": "...", "desc": "...", "idBoard": "...", "idList": "..." }`

#### Asana
- **Type**: `n8n-nodes-base.asana`
- **Trigger**: `n8n-nodes-base.asanaTrigger`
- **Operations**: Project (create, delete, get, getAll, update), Task (create, delete, get, getAll, move, search, update), Subtask (create, getAll), User (get, getAll)
- **Output**: `{ "gid": "...", "name": "...", "assignee": { ... }, "completed": false }`

#### Linear
- **Type**: `n8n-nodes-base.linear`
- **Trigger**: `n8n-nodes-base.linearTrigger`
- **Operations**: Issue (create, delete, get, update), Team (get, getAll)
- **Output**: `{ "id": "...", "title": "...", "state": { "name": "In Progress" }, "priority": 2 }`
- **Relevance**: Modern engineering teams using AI for issue triage

#### monday.com
- **Type**: `n8n-nodes-base.mondayCom`
- **Operations**: Board (archive, create, get, getAll), BoardGroup (create, delete, getAll), BoardItem (addUpdate, changeColumnValue, changeMultipleColumnValues, create, delete, get, getAll, getByColumnValue, moveToGroup), Column (create, getAll)
- **Output**: `{ "id": "...", "name": "...", "column_values": [{ "id": "status", "text": "Working on it" }] }`

---

### 2.4 CRM & Sales

#### HubSpot
- **Type**: `n8n-nodes-base.hubSpot`
- **Trigger**: `n8n-nodes-base.hubSpotTrigger`
- **Operations**: Contact (create, delete, get, getAll, getRecentlyCreatedUpdated, search, update), Company (create, delete, get, getAll, getRecentlyCreatedUpdated, search, update), Deal (create, delete, get, getAll, getRecentlyCreatedUpdated, search, update), Engagement (create, delete, get, getAll), Form (getAll, getFields, getSubmissions), Ticket (create, delete, get, getAll, update)
- **Output**: `{ "id": "123", "properties": { "firstname": "...", "email": "...", "company": "..." } }`
- **Relevance**: AI agents for lead qualification, automated CRM updates

#### Salesforce
- **Type**: `n8n-nodes-base.salesforce`
- **Operations**: Account (create, addNote, delete, get, getAll, getSummary, update), Attachment (create, delete, get, getAll, getSummary, update), Case (addComment, create, delete, get, getAll, getSummary, update), Contact (create, addNote, addToCase, delete, get, getAll, getSummary, update), CustomObject (create, delete, get, getAll, update), Document, Flow, Lead (addNote, create, delete, get, getAll, getSummary, update), Opportunity (addNote, create, delete, get, getAll, getSummary, update), Search, Task, User
- **Output**: `{ "id": "001...", "success": true, "Name": "Acme Corp", "Type": "Customer" }`

#### Pipedrive
- **Type**: `n8n-nodes-base.pipedrive`
- **Operations**: Activity (create, delete, get, getAll, update), Deal (create, delete, get, getAll, search, update), File, Lead, Note, Organization, Person, Product
- **Output**: `{ "success": true, "data": { "id": 1, "title": "...", "value": 5000, "currency": "USD" } }`

---

### 2.5 File Storage & Cloud

#### Google Drive
- **Type**: `n8n-nodes-base.googleDrive`
- **Trigger**: `n8n-nodes-base.googleDriveTrigger`
- **Operations**: Drive (create, delete, get, list, update), File (copy, create, createFromText, delete, download, get, list, move, share, update, upload), Folder (create, delete, share), SharedDrive (create, delete, get, getAll, update)
- **Output**: `{ "id": "1abc...", "name": "document.pdf", "mimeType": "application/pdf", "webViewLink": "..." }`
- **Relevance**: Common file storage for AI document processing workflows

#### Microsoft OneDrive
- **Type**: `n8n-nodes-base.microsoftOneDrive`
- **Trigger**: `n8n-nodes-base.microsoftOneDriveTrigger`
- **Operations**: File (copy, delete, download, get, search, upload), Folder (create, getChildren, rename, search)
- **Output**: `{ "id": "...", "name": "report.xlsx", "size": 12345, "webUrl": "..." }`

#### S3 (AWS / Compatible)
- **Type**: `n8n-nodes-base.s3`
- **Operations**: Bucket (create, delete, getAll, search), File (copy, delete, download, getAll, upload), Folder (create, delete, getAll)
- **Output**: `{ "Key": "path/to/file.pdf", "Bucket": "my-bucket", "ETag": "...", "Size": 67890 }`
- **Relevance**: Object storage for AI document pipelines, model artifacts

#### Dropbox
- **Type**: `n8n-nodes-base.dropbox`
- **Operations**: File (copy, delete, download, get, move, search, upload), Folder (create, delete, list, search)
- **Output**: `{ "name": "file.txt", "path_display": "/folder/file.txt", "size": 1234 }`

#### Box
- **Type**: `n8n-nodes-base.box`
- **Operations**: File (copy, delete, download, get, search, upload), Folder (create, delete, get, search, update)
- **Output**: `{ "type": "file", "id": "...", "name": "...", "sha1": "...", "size": 1234 }`

---

### 2.6 Developer Tools & DevOps

#### GitHub
- **Type**: `n8n-nodes-base.github`
- **Trigger**: `n8n-nodes-base.githubTrigger`
- **Operations**: File (create, delete, edit, get), Issue (create, createComment, edit, get, lock), Label (create, getAll), Release (create, delete, get, getAll, update), Repository (get, getIssues, getProfile, listPopularPaths, listTopReferrers), Review (create, get, getAll, update), User (getRepositories, invite)
- **Output**: `{ "id": 123, "number": 42, "title": "...", "state": "open", "user": { "login": "..." } }`
- **Relevance**: AI agent tools for code management, issue creation, PR review automation

#### GitLab
- **Type**: `n8n-nodes-base.gitlab`
- **Trigger**: `n8n-nodes-base.gitlabTrigger`
- **Operations**: Issue (create, createComment, edit, get, getAll), Release (create, delete, get, getAll, update), Repository (get, getIssues)
- **Output**: `{ "id": 123, "iid": 1, "title": "...", "state": "opened", "web_url": "..." }`

#### Git
- **Type**: `n8n-nodes-base.git`
- **Operations**: Add, AddConfig, Clone, Commit, Fetch, Log, Pull, Push, PushTags, Status, Tag
- **Output**: `{ "commitHash": "abc123...", "message": "...", "author": "...", "date": "..." }`

---

### 2.7 Databases (Additional)

Currently covered: `postgres`, `mongoDb`, `mySql`, `redis`. Adding:

#### Microsoft SQL Server
- **Type**: `n8n-nodes-base.microsoftSql`
- **Operations**: Execute query, Insert, Update, Delete
- **Output**: Row data as JSON array: `[{ "id": 1, "name": "...", "value": 42.0 }]`

#### Elasticsearch
- **Type**: `n8n-nodes-base.elasticsearch`
- **Operations**: Document (create, delete, get, getAll, update), Index (create, delete, get, getAll)
- **Output**: `{ "_index": "my-index", "_id": "1", "_source": { "title": "...", "body": "..." } }`
- **Relevance**: Search/analytics backend for AI knowledge retrieval

#### Supabase
- **Type**: `n8n-nodes-base.supabase`
- **Operations**: Row (create, delete, get, getAll, update)
- **Output**: Row data as JSON: `{ "id": 1, "name": "...", "created_at": "..." }`

#### Snowflake
- **Type**: `n8n-nodes-base.snowflake`
- **Operations**: Execute query, Insert, Update
- **Output**: Row data as JSON array

---

### 2.8 Message Queues & Event Streaming

#### Kafka
- **Type**: `n8n-nodes-base.kafka`
- **Trigger**: `n8n-nodes-base.kafkaTrigger`
- **Operations**: Send message to topic
- **Trigger Output**: `{ "message": "...", "topic": "my-topic", "partition": 0, "offset": 42, "timestamp": "..." }`
- **Relevance**: Event-driven AI agent architectures

#### RabbitMQ
- **Type**: `n8n-nodes-base.rabbitMq`
- **Trigger**: `n8n-nodes-base.rabbitMqTrigger`
- **Operations**: Send message to queue/exchange
- **Output**: `{ "success": true }`

#### AMQP
- **Type**: `n8n-nodes-base.amqp`
- **Trigger**: `n8n-nodes-base.amqpTrigger`
- **Operations**: Send message
- **Output**: `{ "success": true }`

#### MQTT
- **Type**: `n8n-nodes-base.mqtt`
- **Trigger**: `n8n-nodes-base.mqttTrigger`
- **Operations**: Publish message to topic
- **Output**: `{ "topic": "...", "message": "..." }`
- **Relevance**: IoT + AI automation scenarios

---

### 2.9 E-Commerce & Payments

#### Stripe
- **Type**: `n8n-nodes-base.stripe`
- **Trigger**: `n8n-nodes-base.stripeTrigger`
- **Operations**: Balance (get), Charge (create, get, getAll, update), Coupon (create, getAll), Customer (create, delete, get, getAll, update), Source (create, delete, get), Token (create)
- **Output**: `{ "id": "ch_...", "amount": 2000, "currency": "usd", "status": "succeeded" }`

#### Shopify
- **Type**: `n8n-nodes-base.shopify`
- **Trigger**: `n8n-nodes-base.shopifyTrigger`
- **Operations**: Order (create, delete, get, getAll, update), Product (create, delete, get, getAll, update)
- **Output**: `{ "id": 123, "order_number": 1001, "total_price": "99.99", "financial_status": "paid" }`

#### WooCommerce
- **Type**: `n8n-nodes-base.wooCommerce`
- **Trigger**: `n8n-nodes-base.wooCommerceTrigger`
- **Operations**: Customer (create, delete, get, getAll, update), Order (create, delete, get, getAll, update), Product (create, delete, get, getAll, update)
- **Output**: `{ "id": 123, "status": "processing", "total": "59.99", "billing": { ... } }`

---

### 2.10 Customer Support & ITSM

#### Zendesk
- **Type**: `n8n-nodes-base.zendesk`
- **Trigger**: `n8n-nodes-base.zendeskTrigger`
- **Operations**: Ticket (create, delete, get, getAll, recover, update), TicketField (getAll), User (create, delete, get, getAll, search, update), Organization (create, delete, get, getAll, getRelatedData, update)
- **Output**: `{ "id": 123, "subject": "...", "status": "open", "priority": "high", "requester_id": 456 }`
- **Relevance**: AI agents for automated ticket triage and response

#### ServiceNow
- **Type**: `n8n-nodes-base.serviceNow`
- **Operations**: Attachment (get, getAll, upload), Business Service (getAll), Configuration Item (getAll), Department (getAll), Dictionary (getAll), Incident (create, delete, get, getAll, update), Table Record (create, delete, get, getAll, update), User (get, getAll), User Group (getAll), User Role (getAll)
- **Output**: `{ "sys_id": "...", "number": "INC0010001", "short_description": "...", "state": "1" }`
- **Relevance**: Enterprise ITSM AI automation

---

### 2.11 Marketing & Email

#### Mailchimp
- **Type**: `n8n-nodes-base.mailchimp`
- **Trigger**: `n8n-nodes-base.mailchimpTrigger`
- **Operations**: Campaign (create, delete, get, getAll, replicate, resend, send), List/Audience (getAll), ListMember (create, delete, get, getAll, update)
- **Output**: `{ "id": "...", "email_address": "...", "status": "subscribed", "list_id": "..." }`

#### SendGrid
- **Type**: `n8n-nodes-base.sendGrid`
- **Operations**: Contact (create, delete, get, getAll, upsert), List (create, delete, get, getAll, update), Mail (send)
- **Output**: `{ "statusCode": 202, "headers": { ... } }` (send) or contact data (CRUD)

---

### 2.12 Data Transformation & Utility

These core utility nodes are used frequently in workflows for data processing between AI agent steps.

#### Date & Time
- **Type**: `n8n-nodes-base.dateTime`
- **Operations**: Format date, add/subtract time, round date, extract date parts, get time between dates
- **Output**: `{ "date": "2026-01-15T10:30:00.000Z" }`

#### Crypto
- **Type**: `n8n-nodes-base.crypto`
- **Operations**: Generate (UUID, random string), Hash (MD5, SHA-256, etc.), HMAC, Sign
- **Output**: `{ "data": "a1b2c3..." }` (hash) or `{ "data": "550e8400-..." }` (UUID)

#### XML
- **Type**: `n8n-nodes-base.xml`
- **Operations**: JSON to XML, XML to JSON
- **Output**: Converted data as JSON or XML string

#### Markdown
- **Type**: `n8n-nodes-base.markdown`
- **Operations**: Markdown to HTML, HTML to Markdown
- **Output**: `{ "data": "<h1>Title</h1><p>Content</p>" }`
- **Relevance**: Used with AI output to format for email/web delivery

#### HTML
- **Type**: `n8n-nodes-base.html`
- **Operations**: Extract HTML content (CSS selector, table extraction), Generate HTML table from items
- **Output**: Extracted data as JSON items or HTML string: `{ "data": "<table>...</table>" }`
- **Relevance**: Common for web scraping pipelines and formatting AI output as HTML tables for reports

#### Sort
- **Type**: `n8n-nodes-base.sort`
- **Operations**: Sort items by field (ascending/descending), random order
- **Output**: Same items, reordered

#### Limit
- **Type**: `n8n-nodes-base.limit`
- **Operations**: Keep first N items
- **Output**: Subset of input items

#### Split Out
- **Type**: `n8n-nodes-base.splitOut`
- **Operations**: Split array field into separate items
- **Output**: One item per array element

#### Summarize
- **Type**: `n8n-nodes-base.summarize`
- **Operations**: Group and aggregate items (count, sum, average, min, max, concatenate)
- **Output**: Aggregated result items

#### Compare Datasets
- **Type**: `n8n-nodes-base.compareDatasets`
- **Operations**: Compare two inputs, find matching/missing/different items
- **Output**: Items split by match status

#### Remove Duplicates
- **Type**: `n8n-nodes-base.removeDuplicates`
- **Operations**: Remove duplicate items based on fields
- **Output**: Deduplicated items

#### Rename Keys
- **Type**: `n8n-nodes-base.renameKeys`
- **Operations**: Rename JSON keys in items
- **Output**: Items with renamed keys

#### Convert to File
- **Type**: `n8n-nodes-base.convertToFile`
- **Operations**: Convert JSON to CSV/HTML/ICS/ODS/RTF/XLS/XLSX, move binary data
- **Output**: Binary data in `{ "binary": { "data": { "mimeType": "...", "fileName": "..." } } }`

#### Compression
- **Type**: `n8n-nodes-base.compression`
- **Operations**: Compress (gzip, zip), Decompress
- **Output**: Binary data

#### GraphQL
- **Type**: `n8n-nodes-base.graphQl`
- **Operations**: Execute GraphQL query/mutation
- **Output**: `{ "data": { ... } }` (standard GraphQL response)

#### SSH
- **Type**: `n8n-nodes-base.ssh`
- **Operations**: Execute command, download file, upload file
- **Output**: `{ "stdout": "...", "stderr": "...", "exitCode": 0 }`

#### Item Lists (Legacy)
- **Type**: `n8n-nodes-base.itemLists`
- **Operations**: Split Out Items, Aggregate Items, Remove Duplicates, Sort, Limit, Concatenate Items
- **Output**: Transformed items
- **Note**: Older multi-purpose node, predecessor to the specialized `splitOut`, `removeDuplicates`, `sort`, `limit`, `summarize` nodes. Still very common in existing workflows.

#### RSS Feed Read
- **Type**: `n8n-nodes-base.rssFeedRead`
- **Operations**: Read RSS or Atom feed from URL
- **Output**: `{ "title": "...", "link": "https://...", "content": "...", "pubDate": "...", "creator": "..." }`
- **Relevance**: Common data source for content-driven AI workflows

#### Stop And Error
- **Type**: `n8n-nodes-base.stopAndError`
- **Operations**: Stop workflow with error message
- **Output**: None (terminates execution)

---

### 2.13 File I/O & Binary Operations

These nodes handle reading, writing, converting, and manipulating files and binary data. Many were found in the POC execution files.

#### Extract from File
- **Type**: `n8n-nodes-base.extractFromFile`
- **Found in**: `execution_1715.json` ("Extract from File")
- **Operations**: Extract data from binary files — supports CSV, Excel (.xlsx), HTML, ICS, JSON, ODS, PDF, RTF, Text, XML
- **Output**: Extracted rows/items as JSON: `{ "row_number": 1, "column_a": "...", "column_b": "..." }`
- **Relevance**: Critical for AI data pipeline workflows that process uploaded files. The counterpart to `convertToFile`.

#### Spreadsheet File (Legacy)
- **Type**: `n8n-nodes-base.spreadsheetFile`
- **Operations**: Read from file (CSV, Excel .xlsx/.xls, ODS, HTML, RTF) or Write to file (CSV, Excel .xlsx, ODS, HTML, RTF)
- **Output**: Row data as JSON (read) or binary data (write)
- **Note**: Older node, partially superseded by `extractFromFile` + `convertToFile`. Still widely used in existing workflows.

#### Move Binary Data (Legacy)
- **Type**: `n8n-nodes-base.moveBinaryData`
- **Operations**: Binary to JSON (base64 encode), JSON to Binary (base64 decode)
- **Output**: Converted data in target format
- **Note**: Older node for moving data between binary and JSON properties. Still common in file processing pipelines.

#### Edit Image
- **Type**: `n8n-nodes-base.editImage`
- **Operations**: Blur, Border, Composite, Crop, Draw, Information (get metadata), Resize, Rotate, Shear, Text (add text overlay), Transparent
- **Output**: Modified image as binary data: `{ "binary": { "data": { "mimeType": "image/png", "fileName": "output.png" } } }`
- **Relevance**: Image manipulation in content generation and AI vision workflows

#### FTP
- **Type**: `n8n-nodes-base.ftp`
- **Operations**: Delete, Download, List, Rename, Upload
- **Output**: File listing: `{ "path": "/files/data.csv", "size": 12345, "modifyTime": "..." }` or binary data (download)
- **Relevance**: Enterprise file transfer in data pipeline automation

---

### 2.14 Workflow Data & Triggers

These nodes provide in-workflow data storage and cross-workflow triggering capabilities.

#### Data Table
- **Type**: `n8n-nodes-base.dataTable`
- **Found in**: `execution_1715.json` ("Insert row", "Insert row1", "Insert row2", "Get row(s)")
- **Operations**: Insert row, Get row(s), Update row, Delete row
- **Output**: Row data as JSON: `{ "id": 1, "column_name": "value", ... }`
- **Relevance**: In-workflow structured data storage. Used for building lookup tables, staging data between AI agent steps, and accumulating results during loop iterations.

#### Execute Workflow Trigger
- **Type**: `n8n-nodes-base.executeWorkflowTrigger`
- **Found in**: `execution_1648.json` ("When Executed by Another Workflow"), `execution_1716.json`
- **Operations**: Receives data when this workflow is called by another workflow via the `Execute Workflow` node
- **Output**: Input data passed from the calling workflow
- **Note**: Different from `n8n-nodes-base.executeWorkflow` (which *calls* another workflow). This is the *trigger* that fires when the current workflow is invoked as a sub-workflow. Already in types.go via generic trigger detection but should be explicitly listed.

---

### 2.15 Microsoft & Azure

Since the platform is heavily Azure-driven, dedicated Microsoft and Azure app nodes are critical for enterprise AI agent automation workflows. These complement the Microsoft nodes already cataloged in other sections (Teams in 2.1, Outlook in 2.1, Excel in 2.2, OneDrive in 2.5, SQL Server in 2.7).

#### Azure Cosmos DB
- **Type**: `n8n-nodes-base.azureCosmosDb`
- **Operations**: Container (create, delete, get, getMany), Item (create, delete, get, getMany, executeQuery, update)
- **Output**: `{ "id": "...", "partitionKey": "...", "_rid": "...", "_ts": 1234567890, ... }` (item data) or `{ "id": "myContainer", ... }` (container)
- **Relevance**: Primary NoSQL database for Azure-native AI agent data storage, conversation logs, and document indexing

#### Azure Storage
- **Type**: `n8n-nodes-base.azureStorage`
- **Operations**: Blob (create, delete, get, getMany), Container (create, delete, get, getMany)
- **Output**: `{ "name": "report.pdf", "containerName": "documents", "contentLength": 12345, "contentType": "application/pdf" }`
- **Relevance**: Azure Blob Storage for AI document pipelines, file uploads, model artifacts, and binary data storage

#### Microsoft SharePoint
- **Type**: `n8n-nodes-base.microsoftSharePoint`
- **Operations**: File (download, get, getAll, upload), List (create, delete, get, getAll, update), ListItem (create, delete, get, getAll, update), Site (get, getAll)
- **Output**: `{ "id": "...", "name": "Document.docx", "webUrl": "https://tenant.sharepoint.com/...", "size": 12345 }`
- **Relevance**: Enterprise document management — AI agents accessing/creating SharePoint documents, list items, and site content

#### Microsoft Graph Security
- **Type**: `n8n-nodes-base.microsoftGraphSecurity`
- **Operations**: SecureScore (get, getAll), SecureScoreControlProfile (get, getAll, update)
- **Output**: `{ "id": "...", "azureTenantId": "...", "currentScore": 42.0, "maxScore": 100.0,  "controlScores": [...] }`
- **Relevance**: Security monitoring and compliance automation in Azure-centric environments

#### Microsoft To Do
- **Type**: `n8n-nodes-base.microsoftToDo`
- **Operations**: LinkedResource (create, delete, get, getAll, update), List (create, delete, get, getAll, update), Task (create, delete, get, getAll, update)
- **Output**: `{ "id": "...", "title": "Review AI output", "status": "notStarted", "importance": "high", "body": { "content": "..." } }`
- **Relevance**: Task management integration for AI agent workflows — creating follow-up tasks from AI processing results

#### Microsoft Entra ID
- **Type**: `n8n-nodes-base.microsoftEntra`
- **Operations**: Group (create, delete, get, getAll, update), User (create, delete, get, getAll, update), Group Member (add, get, getAll, remove)
- **Output**: `{ "id": "...", "displayName": "...", "userPrincipalName": "user@tenant.onmicrosoft.com", "mail": "..." }`
- **Relevance**: Identity and access management automation — AI agents managing user provisioning, group memberships, and access reviews in Azure AD

**Note**: Additional Azure services are available via the **HTTP Request node** with predefined Azure credential types (Azure Monitor, etc.). The LangChain Azure sub-nodes (Azure OpenAI Chat Model, Embeddings Azure OpenAI, Azure AI Search Vector Store) are covered in the [AI concept document](./N8N_NODE_TYPES_CONCEPT.md).

---

## 3. Output Data Structure

All automation nodes use the standard `main` output connection. The output structure is always:

```json
{
  "data": {
    "main": [
      [
        {
          "json": { /* node-specific output data */ },
          "pairedItem": { "item": 0 }
        }
      ]
    ]
  }
}
```

Some nodes additionally produce binary output:

```json
{
  "data": {
    "main": [
      [
        {
          "json": { "fileName": "report.pdf", "mimeType": "application/pdf" },
          "binary": {
            "data": {
              "data": "base64...",
              "mimeType": "application/pdf",
              "fileName": "report.pdf",
              "fileSize": "12345 B"
            }
          },
          "pairedItem": { "item": 0 }
        }
      ]
    ]
  }
}
```

**Impact on transformer**: Zero changes needed for output extraction. The existing `GetOutputItems()` cascade starts with `Main`, which is the only connection type these nodes use.

---

## 4. New NodeType Domain Constants

Currently in `internal/domain/models/trace.go`:

```go
const (
    NodeTypeAgent       NodeType = "agent"
    NodeTypeTool        NodeType = "tool"
    NodeTypeLLM         NodeType = "llm"
    NodeTypeChain       NodeType = "chain"
    NodeTypeRetriever   NodeType = "retriever"
    NodeTypeWorkflow    NodeType = "workflow"
    NodeTypeFunction    NodeType = "function"
    NodeTypeHTTP        NodeType = "http"
    NodeTypeCode        NodeType = "code"
    NodeTypeConditional NodeType = "conditional"
    NodeTypeLoop        NodeType = "loop"
    NodeTypeCustom      NodeType = "custom"
    NodeTypeMemory      NodeType = "memory"
    NodeTypeVectorStore NodeType = "vector_store"
    NodeTypeEmbedding   NodeType = "embedding"
    NodeTypeOutputParser NodeType = "output_parser"
    NodeTypeDocument    NodeType = "document"
    NodeTypeTextSplitter NodeType = "text_splitter"
)
```

### Recommended New NodeType Constants

To enable meaningful frontend visualization (icons, colors, grouping), add these domain types:

```go
const (
    // New automation node types
    NodeTypeMessaging      NodeType = "messaging"       // Slack, Discord, Telegram, Teams, WhatsApp, Twilio, Email
    NodeTypeSpreadsheet    NodeType = "spreadsheet"     // Google Sheets, Airtable, Excel, Notion
    NodeTypeProjectMgmt    NodeType = "project_mgmt"    // Jira, Trello, Asana, Linear, monday.com
    NodeTypeCRM            NodeType = "crm"             // HubSpot, Salesforce, Pipedrive
    NodeTypeStorage        NodeType = "storage"         // Google Drive, OneDrive, S3, Dropbox, Box
    NodeTypeDevOps         NodeType = "devops"          // GitHub, GitLab, Git
    NodeTypeDatabase       NodeType = "database"        // All DB nodes (including existing postgres/mysql/mongodb/redis)
    NodeTypeQueue          NodeType = "queue"           // Kafka, RabbitMQ, AMQP, MQTT
    NodeTypePayment        NodeType = "payment"         // Stripe, Shopify, WooCommerce
    NodeTypeSupport        NodeType = "support"         // Zendesk, ServiceNow
    NodeTypeMarketing      NodeType = "marketing"       // Mailchimp, SendGrid
    NodeTypeDataTransform  NodeType = "data_transform"  // Sort, Limit, Split Out, Summarize, etc.
)
```

**Alternative (minimal approach)**: If adding 12 new NodeType values is too many, group them:
- `NodeTypeApp` — all SaaS integrations (messaging, CRM, PM, storage, support, marketing, payment)
- `NodeTypeDataTransform` — all utility/transformation nodes
- `NodeTypeQueue` — message queue nodes
- Keep `NodeTypeDatabase` for all DB nodes

**Recommendation**: Start with the **minimal approach** (3 new types: `app`, `data_transform`, `queue`) and expand later if the frontend needs more granularity. The `GetNodeCategory()` function can still return fine-grained categories for internal use.

---

## 5. GetNodeCategory Updates

The `GetNodeCategory()` function in `types.go` should be extended with new suffix patterns. All new patterns go **before** the `default: return "custom"` fallback.

```go
func GetNodeCategory(nodeType string) string {
    suffix := extractNodeSuffix(nodeType)

    // ... existing patterns (form, lmChat, embeddings, memory, etc.) ...

    // NEW: Communication & Messaging
    case suffix == "slack" || suffix == "slackTrigger":
        return "messaging"
    case suffix == "discord":
        return "messaging"
    case suffix == "telegram" || suffix == "telegramTrigger":
        return "messaging"
    case suffix == "microsoftTeams" || suffix == "microsoftTeamsTrigger":
        return "messaging"
    case suffix == "gmail" || suffix == "gmailTrigger":
        return "messaging"
    case suffix == "microsoftOutlook" || suffix == "microsoftOutlookTrigger":
        return "messaging"
    case suffix == "sendEmail" || suffix == "emailImap":
        return "messaging"
    case suffix == "twilio" || suffix == "twilioTrigger":
        return "messaging"
    case suffix == "whatsApp" || suffix == "whatsAppTrigger":
        return "messaging"

    // NEW: Productivity & Spreadsheets
    case suffix == "googleSheets" || suffix == "googleSheetsTrigger":
        return "spreadsheet"
    case suffix == "airtable" || suffix == "airtableTrigger":
        return "spreadsheet"
    case suffix == "notion" || suffix == "notionTrigger":
        return "spreadsheet"
    case suffix == "microsoftExcel":
        return "spreadsheet"
    case suffix == "googleDocs":
        return "spreadsheet"
    case suffix == "googleCalendar" || suffix == "googleCalendarTrigger":
        return "spreadsheet"

    // NEW: Project Management
    case suffix == "jira" || suffix == "jiraTrigger":
        return "project_mgmt"
    case suffix == "trello" || suffix == "trelloTrigger":
        return "project_mgmt"
    case suffix == "asana" || suffix == "asanaTrigger":
        return "project_mgmt"
    case suffix == "linear" || suffix == "linearTrigger":
        return "project_mgmt"
    case suffix == "mondayCom":
        return "project_mgmt"

    // NEW: CRM
    case suffix == "hubSpot" || suffix == "hubSpotTrigger":
        return "crm"
    case suffix == "salesforce":
        return "crm"
    case suffix == "pipedrive":
        return "crm"

    // NEW: File Storage
    case suffix == "googleDrive" || suffix == "googleDriveTrigger":
        return "storage"
    case suffix == "microsoftOneDrive" || suffix == "microsoftOneDriveTrigger":
        return "storage"
    case suffix == "s3":
        return "storage"
    case suffix == "dropbox":
        return "storage"
    case suffix == "box":
        return "storage"

    // NEW: Developer Tools
    case suffix == "github" || suffix == "githubTrigger":
        return "devops"
    case suffix == "gitlab" || suffix == "gitlabTrigger":
        return "devops"
    case suffix == "git":
        return "devops"

    // NEW: Databases (additional)
    case suffix == "microsoftSql":
        return "database"
    case suffix == "elasticsearch":
        return "database"
    case suffix == "supabase":
        return "database"
    case suffix == "snowflake":
        return "database"

    // NEW: Message Queues
    case suffix == "kafka" || suffix == "kafkaTrigger":
        return "queue"
    case suffix == "rabbitMq" || suffix == "rabbitMqTrigger":
        return "queue"
    case suffix == "amqp" || suffix == "amqpTrigger":
        return "queue"
    case suffix == "mqtt" || suffix == "mqttTrigger":
        return "queue"

    // NEW: E-Commerce & Payments
    case suffix == "stripe" || suffix == "stripeTrigger":
        return "payment"
    case suffix == "shopify" || suffix == "shopifyTrigger":
        return "payment"
    case suffix == "wooCommerce" || suffix == "wooCommerceTrigger":
        return "payment"

    // NEW: Support & ITSM
    case suffix == "zendesk" || suffix == "zendeskTrigger":
        return "support"
    case suffix == "serviceNow":
        return "support"

    // NEW: Marketing
    case suffix == "mailchimp" || suffix == "mailchimpTrigger":
        return "marketing"
    case suffix == "sendGrid":
        return "marketing"

    // NEW: Microsoft & Azure
    case suffix == "azureCosmosDb":
        return "database"
    case suffix == "azureStorage":
        return "storage"
    case suffix == "microsoftSharePoint":
        return "storage"
    case suffix == "microsoftGraphSecurity":
        return "security"
    case suffix == "microsoftToDo":
        return "productivity"
    case suffix == "microsoftEntra":
        return "identity"

    // NEW: Data Transformation
    case suffix == "dateTime" || suffix == "crypto" || suffix == "xml" ||
         suffix == "markdown" || suffix == "html" || suffix == "sort" || suffix == "limit" ||
         suffix == "splitOut" || suffix == "summarize" ||
         suffix == "compareDatasets" || suffix == "removeDuplicates" ||
         suffix == "renameKeys" || suffix == "convertToFile" ||
         suffix == "compression" || suffix == "itemLists":
        return "data_transform"

    // NEW: File I/O & Binary Operations
    case suffix == "extractFromFile" || suffix == "spreadsheetFile" ||
         suffix == "moveBinaryData" || suffix == "editImage" || suffix == "ftp":
        return "file_io"

    // NEW: Workflow Data
    case suffix == "dataTable":
        return "data_store"
    case suffix == "executeWorkflowTrigger":
        return "trigger"

    // NEW: Data Sources
    case suffix == "rssFeedRead":
        return "data_source"

    // NEW: Misc Utility (still tool-like)
    case suffix == "graphQl":
        return "tool"
    case suffix == "ssh":
        return "tool"
    case suffix == "stopAndError":
        return "core"
```

**Note**: The trigger variant nodes (e.g., `slackTrigger`) currently match the generic `strings.HasSuffix(suffix, "Trigger")` rule and return `"trigger"`. The new category patterns should be placed **before** the generic trigger catch-all, or the trigger variants can intentionally be left to match the generic trigger category. Recommendation: keep triggers as `"trigger"` category since that's how they function, and only categorize the action nodes by domain.

**Updated recommendation**: Add domain categories only for **action nodes**. Trigger variants (`slackTrigger`, `telegramTrigger`, etc.) should continue to match the existing `HasSuffix(suffix, "Trigger")` → `"trigger"` path. This means only the base node identifiers (e.g., `slack`, `telegram`) need new category entries.

---

## 6. mapNodeType Updates

The `mapNodeType()` function maps N8N types to domain `NodeType` constants. Using the **minimal approach** (3 new types: `app`, `data_transform`, `queue`):

```go
func (t *Transformer) mapNodeType(n8nType string) models.NodeType {
    suffix := extractNodeSuffix(n8nType)

    switch {
    // ... existing patterns (lmChat, embeddings, memory, etc.) ...

    // Existing trigger catch-all stays (returns NodeTypeWorkflow)
    // Existing conditional/loop stays
    // Existing httpRequest, code, function stays

    // NEW: Database nodes (expand existing pattern)
    case suffix == "postgres" || suffix == "mongoDb" || suffix == "mySql" ||
         suffix == "redis" || suffix == "microsoftSql" || suffix == "elasticsearch" ||
         suffix == "supabase" || suffix == "snowflake" || suffix == "azureCosmosDb":
        return models.NodeTypeDatabase

    // NEW: Message Queues
    case suffix == "kafka" || suffix == "rabbitMq" || suffix == "amqp" || suffix == "mqtt":
        return models.NodeTypeQueue

    // NEW: Data Transformation
    case suffix == "dateTime" || suffix == "crypto" || suffix == "xml" ||
         suffix == "markdown" || suffix == "html" || suffix == "sort" || suffix == "limit" ||
         suffix == "splitOut" || suffix == "summarize" ||
         suffix == "compareDatasets" || suffix == "removeDuplicates" ||
         suffix == "renameKeys" || suffix == "convertToFile" ||
         suffix == "compression" || suffix == "itemLists" ||
         suffix == "rssFeedRead":
        return models.NodeTypeDataTransform

    // NEW: File I/O & Binary Operations
    case suffix == "extractFromFile" || suffix == "spreadsheetFile" ||
         suffix == "moveBinaryData" || suffix == "editImage" || suffix == "ftp":
        return models.NodeTypeTool

    // NEW: Workflow Data
    case suffix == "dataTable":
        return models.NodeTypeTool
    case suffix == "executeWorkflowTrigger":
        return models.NodeTypeWorkflow

    // NEW: App integrations (SaaS) → NodeTypeApp
    case suffix == "slack" || suffix == "discord" || suffix == "telegram" ||
         suffix == "microsoftTeams" || suffix == "gmail" || suffix == "microsoftOutlook" ||
         suffix == "sendEmail" || suffix == "twilio" || suffix == "whatsApp" ||
         suffix == "googleSheets" || suffix == "airtable" || suffix == "notion" ||
         suffix == "microsoftExcel" || suffix == "googleDocs" || suffix == "googleCalendar" ||
         suffix == "jira" || suffix == "trello" || suffix == "asana" ||
         suffix == "linear" || suffix == "mondayCom" ||
         suffix == "hubSpot" || suffix == "salesforce" || suffix == "pipedrive" ||
         suffix == "googleDrive" || suffix == "microsoftOneDrive" || suffix == "s3" ||
         suffix == "dropbox" || suffix == "box" || suffix == "azureStorage" ||
         suffix == "microsoftSharePoint" ||
         suffix == "github" || suffix == "gitlab" || suffix == "git" ||
         suffix == "stripe" || suffix == "shopify" || suffix == "wooCommerce" ||
         suffix == "zendesk" || suffix == "serviceNow" ||
         suffix == "mailchimp" || suffix == "sendGrid" ||
         suffix == "microsoftGraphSecurity" || suffix == "microsoftToDo" ||
         suffix == "microsoftEntra":
        return models.NodeTypeApp

    // Misc tools
    case suffix == "graphQl" || suffix == "ssh":
        return models.NodeTypeTool

    case suffix == "stopAndError":
        return models.NodeTypeWorkflow

    default:
        return models.NodeTypeCustom
    }
}
```

**Important**: The existing database nodes (`postgres`, `mongoDb`, `mySql`, `redis`) currently map to `NodeTypeTool`. If we introduce `NodeTypeDatabase`, these must be migrated. This is a **breaking change for the frontend** if it relies on the current `tool` type for these nodes. Consider this carefully.

**Conservative alternative**: Keep existing DB nodes mapping to `NodeTypeTool`, and only add new DB nodes to `NodeTypeDatabase`. Or better — map ALL database nodes to `NodeTypeDatabase` and update frontend together.

---

## 7. Summary Table

| # | Node | Type Identifier | Category | NodeType | Trigger Variant |
|---|------|----------------|----------|----------|-----------------|
| **Communication & Messaging** |||||
| 1 | Slack | `n8n-nodes-base.slack` | messaging | app | `slackTrigger` |
| 2 | Discord | `n8n-nodes-base.discord` | messaging | app | — |
| 3 | Telegram | `n8n-nodes-base.telegram` | messaging | app | `telegramTrigger` |
| 4 | Microsoft Teams | `n8n-nodes-base.microsoftTeams` | messaging | app | `microsoftTeamsTrigger` |
| 5 | Gmail | `n8n-nodes-base.gmail` | messaging | app | `gmailTrigger` |
| 6 | Microsoft Outlook | `n8n-nodes-base.microsoftOutlook` | messaging | app | `microsoftOutlookTrigger` |
| 7 | Send Email (SMTP) | `n8n-nodes-base.sendEmail` | messaging | app | — |
| 8 | Twilio | `n8n-nodes-base.twilio` | messaging | app | `twilioTrigger` |
| 9 | WhatsApp | `n8n-nodes-base.whatsApp` | messaging | app | `whatsAppTrigger` |
| **Productivity & Spreadsheets** |||||
| 10 | Google Sheets | `n8n-nodes-base.googleSheets` | spreadsheet | app | `googleSheetsTrigger` |
| 11 | Airtable | `n8n-nodes-base.airtable` | spreadsheet | app | `airtableTrigger` |
| 12 | Notion | `n8n-nodes-base.notion` | spreadsheet | app | `notionTrigger` |
| 13 | Microsoft Excel | `n8n-nodes-base.microsoftExcel` | spreadsheet | app | — |
| 14 | Google Docs | `n8n-nodes-base.googleDocs` | spreadsheet | app | — |
| 15 | Google Calendar | `n8n-nodes-base.googleCalendar` | spreadsheet | app | `googleCalendarTrigger` |
| **Project Management** |||||
| 16 | Jira Software | `n8n-nodes-base.jira` | project_mgmt | app | `jiraTrigger` |
| 17 | Trello | `n8n-nodes-base.trello` | project_mgmt | app | `trelloTrigger` |
| 18 | Asana | `n8n-nodes-base.asana` | project_mgmt | app | `asanaTrigger` |
| 19 | Linear | `n8n-nodes-base.linear` | project_mgmt | app | `linearTrigger` |
| 20 | monday.com | `n8n-nodes-base.mondayCom` | project_mgmt | app | — |
| **CRM & Sales** |||||
| 21 | HubSpot | `n8n-nodes-base.hubSpot` | crm | app | `hubSpotTrigger` |
| 22 | Salesforce | `n8n-nodes-base.salesforce` | crm | app | — |
| 23 | Pipedrive | `n8n-nodes-base.pipedrive` | crm | app | — |
| **File Storage & Cloud** |||||
| 24 | Google Drive | `n8n-nodes-base.googleDrive` | storage | app | `googleDriveTrigger` |
| 25 | Microsoft OneDrive | `n8n-nodes-base.microsoftOneDrive` | storage | app | `microsoftOneDriveTrigger` |
| 26 | S3 | `n8n-nodes-base.s3` | storage | app | — |
| 27 | Dropbox | `n8n-nodes-base.dropbox` | storage | app | — |
| 28 | Box | `n8n-nodes-base.box` | storage | app | — |
| **Developer Tools** |||||
| 29 | GitHub | `n8n-nodes-base.github` | devops | app | `githubTrigger` |
| 30 | GitLab | `n8n-nodes-base.gitlab` | devops | app | `gitlabTrigger` |
| 31 | Git | `n8n-nodes-base.git` | devops | app | — |
| **Databases (Additional)** |||||
| 32 | Microsoft SQL | `n8n-nodes-base.microsoftSql` | database | database | — |
| 33 | Elasticsearch | `n8n-nodes-base.elasticsearch` | database | database | — |
| 34 | Supabase | `n8n-nodes-base.supabase` | database | database | — |
| 35 | Snowflake | `n8n-nodes-base.snowflake` | database | database | — |
| **Message Queues** |||||
| 36 | Kafka | `n8n-nodes-base.kafka` | queue | queue | `kafkaTrigger` |
| 37 | RabbitMQ | `n8n-nodes-base.rabbitMq` | queue | queue | `rabbitMqTrigger` |
| 38 | AMQP | `n8n-nodes-base.amqp` | queue | queue | `amqpTrigger` |
| 39 | MQTT | `n8n-nodes-base.mqtt` | queue | queue | `mqttTrigger` |
| **E-Commerce & Payments** |||||
| 40 | Stripe | `n8n-nodes-base.stripe` | payment | app | `stripeTrigger` |
| 41 | Shopify | `n8n-nodes-base.shopify` | payment | app | `shopifyTrigger` |
| 42 | WooCommerce | `n8n-nodes-base.wooCommerce` | payment | app | `wooCommerceTrigger` |
| **Customer Support** |||||
| 43 | Zendesk | `n8n-nodes-base.zendesk` | support | app | `zendeskTrigger` |
| 44 | ServiceNow | `n8n-nodes-base.serviceNow` | support | app | — |
| **Marketing** |||||
| 45 | Mailchimp | `n8n-nodes-base.mailchimp` | marketing | app | `mailchimpTrigger` |
| 46 | SendGrid | `n8n-nodes-base.sendGrid` | marketing | app | — |
| **Data Transformation** |||||
| 47 | Date & Time | `n8n-nodes-base.dateTime` | data_transform | data_transform | — |
| 48 | Crypto | `n8n-nodes-base.crypto` | data_transform | data_transform | — |
| 49 | XML | `n8n-nodes-base.xml` | data_transform | data_transform | — |
| 50 | Markdown | `n8n-nodes-base.markdown` | data_transform | data_transform | — |
| 51 | HTML | `n8n-nodes-base.html` | data_transform | data_transform | — |
| 52 | Sort | `n8n-nodes-base.sort` | data_transform | data_transform | — |
| 53 | Limit | `n8n-nodes-base.limit` | data_transform | data_transform | — |
| 54 | Split Out | `n8n-nodes-base.splitOut` | data_transform | data_transform | — |
| 55 | Summarize | `n8n-nodes-base.summarize` | data_transform | data_transform | — |
| 56 | Compare Datasets | `n8n-nodes-base.compareDatasets` | data_transform | data_transform | — |
| 57 | Remove Duplicates | `n8n-nodes-base.removeDuplicates` | data_transform | data_transform | — |
| 58 | Rename Keys | `n8n-nodes-base.renameKeys` | data_transform | data_transform | — |
| 59 | Convert to File | `n8n-nodes-base.convertToFile` | data_transform | data_transform | — |
| 60 | Compression | `n8n-nodes-base.compression` | data_transform | data_transform | — |
| **Utility** |||||
| 61 | GraphQL | `n8n-nodes-base.graphQl` | tool | tool | — |
| 62 | SSH | `n8n-nodes-base.ssh` | tool | tool | — |
| 63 | Stop And Error | `n8n-nodes-base.stopAndError` | core | workflow | — |
| **Data Transformation (additional)** |||||
| 64 | Item Lists (Legacy) | `n8n-nodes-base.itemLists` | data_transform | data_transform | — |
| 65 | RSS Feed Read | `n8n-nodes-base.rssFeedRead` | data_source | data_transform | — |
| **File I/O & Binary Operations** |||||
| 66 | Extract from File | `n8n-nodes-base.extractFromFile` | file_io | tool | — |
| 67 | Spreadsheet File (Legacy) | `n8n-nodes-base.spreadsheetFile` | file_io | tool | — |
| 68 | Move Binary Data (Legacy) | `n8n-nodes-base.moveBinaryData` | file_io | tool | — |
| 69 | Edit Image | `n8n-nodes-base.editImage` | file_io | tool | — |
| 70 | FTP | `n8n-nodes-base.ftp` | file_io | tool | — |
| **Workflow Data & Triggers** |||||
| 71 | Data Table | `n8n-nodes-base.dataTable` | data_store | tool | — |
| 72 | Execute Workflow Trigger | `n8n-nodes-base.executeWorkflowTrigger` | trigger | workflow | — |
| **Microsoft & Azure** |||||
| 73 | Azure Cosmos DB | `n8n-nodes-base.azureCosmosDb` | database | database | — |
| 74 | Azure Storage | `n8n-nodes-base.azureStorage` | storage | app | — |
| 75 | Microsoft SharePoint | `n8n-nodes-base.microsoftSharePoint` | storage | app | — |
| 76 | Microsoft Graph Security | `n8n-nodes-base.microsoftGraphSecurity` | security | app | — |
| 77 | Microsoft To Do | `n8n-nodes-base.microsoftToDo` | productivity | app | — |
| 78 | Microsoft Entra ID | `n8n-nodes-base.microsoftEntra` | identity | app | — |

**Total**: 78 new nodes (+ ~25 trigger variants handled by existing `HasSuffix("Trigger")` rule)

---

## 8. Implementation Recommendations

### 8.1 Implementation Order

1. **Add new NodeType domain constants** to `internal/domain/models/trace.go`
   - Minimum: `NodeTypeApp`, `NodeTypeDataTransform`, `NodeTypeQueue`, `NodeTypeDatabase`
   - These are the 4 new types needed (database replaces mapping DB nodes → tool)

2. **Add new node type constants** to `types.go`
   - Add all 69 action node constants organized by category
   - Add trigger variant constants for the ~25 trigger nodes

3. **Update `GetNodeCategory()`** in `types.go`
   - Add suffix matches for all new nodes BEFORE the generic trigger catch-all
   - Trigger variants still match the existing `HasSuffix("Trigger")` → `"trigger"` rule

4. **Update `mapNodeType()`** in `transformer.go`
   - Add suffix matches mapping to the new NodeType constants
   - Move existing DB nodes (`postgres`, `mongoDb`, `mySql`, `redis`) from `NodeTypeTool` to `NodeTypeDatabase`

5. **Add unit tests** for all new mappings
   - Extend `TestMapNodeType` and `TestGetNodeCategory` test tables
   - Each new node type needs at least one test case

6. **Coordinate with frontend** if introducing new NodeType values
   - Frontend trace visualization needs icons/colors for new types

### 8.2 Node Type Identifier Casing

The type identifiers documented here use the URL slug pattern from N8N docs (e.g., `googleSheets`, `microsoftTeams`). The actual identifiers in N8N execution data use **camelCase** matching the TypeScript class names. The suffix-based matching in our transformer (via `extractNodeSuffix()`) is **case-sensitive**, so the constants must match exactly.

If there's any uncertainty about exact casing, verify against real N8N execution data. The most reliable approach is:
- Run a workflow using the specific node
- Inspect the execution JSON at `workflowData.nodes[].type`
- Use that exact string as the constant value

### 8.3 Trigger Node Handling

App-specific trigger nodes (e.g., `n8n-nodes-base.slackTrigger`, `n8n-nodes-base.telegramTrigger`) follow the pattern `{appName}Trigger`. These already match the existing `strings.HasSuffix(suffix, "Trigger")` rule in both `GetNodeCategory()` and `mapNodeType()`, which returns `"trigger"` and `NodeTypeWorkflow` respectively. No additional handling needed for triggers.

### 8.4 No Changes to Output Extraction

All automation nodes output through `data.main`, which is already the first check in `GetOutputItems()`. The existing `extractOutputData()` function handles them correctly without modification.

### 8.5 Metadata Handling

Most automation nodes don't produce specialized metadata like `tokenUsage` or `subRun`. The existing `buildNodeMetadata()` function handles this gracefully — it only extracts metadata fields if present. No changes needed.

### 8.6 Error Handling

Automation nodes that fail (e.g., Slack API error, Google Sheets auth failure) produce the standard `NodeExecutionError` structure with `name`, `message`, `description`. The existing `mapNodeStatus()` function handles this correctly.

### 8.7 Default Fallback & Error Resilience

Both the N8N and Foundry transformers **must** guarantee that unsupported node types and transformation errors never cause the entire trace import to fail. This is a cross-cutting concern that applies to **all** integrations.

#### 8.7.1 Unsupported Node Type Fallback

**Current state (partial):**
- N8N `mapNodeType()` returns `NodeTypeCustom` for unknown suffixes, but does NOT add a metadata signal
- Foundry `transformUnknown()` creates a node with `Name: "Unknown: {type}"` and stores `original_type` in metadata

**Required behavior (both integrations):**

When a node type is not recognized by the mapper, the transformer MUST:
1. Map it to `NodeTypeCustom` (already done)
2. Set `Metadata["_unsupported_node_type"] = true`
3. Set `Metadata["_original_node_type"] = "{original_type_string}"`
4. Set `Metadata["_fallback_reason"] = "node_type_not_mapped"`
5. Preserve **all** original data (input/output) unchanged

This allows the frontend to display a visual indicator (e.g., warning badge) and lets developers identify which node types need to be added.

**N8N `transformNodeExecution()` update:**
```go
traceNodeType := t.mapNodeType(nodeType)
if traceNodeType == models.NodeTypeCustom && !t.isKnownCustomType(nodeType) {
    metadata["_unsupported_node_type"] = true
    metadata["_original_node_type"] = nodeType
    metadata["_fallback_reason"] = "node_type_not_mapped"
}
```

**Foundry `transformUnknown()` update:**
```go
metadata["_unsupported_node_type"] = true
metadata["_original_node_type"] = item.Type
metadata["_fallback_reason"] = "item_type_not_recognized"
```

#### 8.7.2 Per-Node Error Recovery

**Current state:** If any single node transformation panics (e.g., nil pointer on unexpected data), the entire `TransformExecution()` / `Transform()` call fails. No trace is saved.

**Required behavior:** Each individual node transformation MUST be wrapped in a recovery block. If a node transformation fails, the transformer MUST:
1. Recover from the panic
2. Create a fallback `TraceNode` with:
   - `Type: NodeTypeCustom`
   - `Status: NodeStatusFailed`
   - `Name: "Error: {nodeName}"` (or `"Error: {itemType}"`)
   - `Metadata["_transformation_error"] = true`
   - `Metadata["_error_message"] = "{recovered panic message}"`
   - `Metadata["_fallback_reason"] = "transformation_error"`
3. Continue processing remaining nodes
4. Return all successfully transformed nodes plus the error fallback nodes

**N8N implementation pattern:**
```go
for nodeName, nodeExecutions := range runData {
    for runIndex := range nodeExecutions {
        traceNode, err := t.safeTransformNodeExecution(nodeName, nodeType, runIndex, &nodeExecutions[runIndex], createdBy)
        if err != nil {
            traceNode = t.buildErrorFallbackNode(nodeName, nodeType, runIndex, err, createdBy)
        }
        nodesByName[nodeName] = append(nodesByName[nodeName], traceNode)
    }
}
```

**Foundry implementation pattern:**
```go
func (t *Transformer) safeTransformItem(item ConversationItem, createdBy string) (node models.TraceNode, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic: %v", r)
        }
    }()
    // ... existing transform logic
}
```

#### 8.7.3 Importer-Level Error Recovery

**Current state:** If the transformer itself panics, the `Import()` method fails and no trace is persisted.

**Required behavior:** The importer MUST wrap the transformer call in a recovery block. If the transformer panics:
1. Recover from the panic
2. Create a trace with a single error node containing the panic details
3. Set `ReferenceMetadata["_import_error"] = true` and `ReferenceMetadata["_error_message"] = "{panic message}"`
4. Still persist the trace (so the user sees an error state rather than nothing)

#### 8.7.4 Metadata Key Convention

All fallback/error metadata keys use underscore prefix (`_`) to distinguish them from regular business metadata:

| Key | Type | When Set |
|-----|------|----------|
| `_unsupported_node_type` | `bool` | Node type not in mapper |
| `_original_node_type` | `string` | Original type identifier before fallback |
| `_fallback_reason` | `string` | Why fallback was used: `node_type_not_mapped`, `item_type_not_recognized`, `transformation_error` |
| `_transformation_error` | `bool` | Node transformation panicked |
| `_error_message` | `string` | Error/panic message |
| `_import_error` | `bool` | Entire transformer panicked |

---

## Appendix: Type Identifier Verification Notes

The type identifiers used in this document are derived from the official N8N docs URL slugs using the pattern:
- Docs URL: `https://docs.n8n.io/integrations/builtin/app-nodes/n8n-nodes-base.{slug}/`
- Type identifier in execution data: `n8n-nodes-base.{camelCase}`

Known verified identifiers (from existing types.go or confirmed execution data):
- `n8n-nodes-base.httpRequest` (not `httprequest`)
- `n8n-nodes-base.mongoDb` (not `mongodb`)
- `n8n-nodes-base.mySql` (not `mysql`)
- `n8n-nodes-base.splitInBatches` (not `splitinbatches`)
- `n8n-nodes-base.formTrigger` (not `formtrigger`)

Identifiers that need verification from real execution data before implementation:
- `n8n-nodes-base.microsoftTeams` vs `microsoftteams`
- `n8n-nodes-base.googleSheets` vs `googlesheets`
- `n8n-nodes-base.hubSpot` vs `hubspot`
- `n8n-nodes-base.whatsApp` vs `whatsapp`
- `n8n-nodes-base.microsoftOneDrive` vs `microsoftonedrive`
- `n8n-nodes-base.mondayCom` vs `mondaycom`
- `n8n-nodes-base.wooCommerce` vs `woocommerce`
- `n8n-nodes-base.azureCosmosDb` vs `azurecosmosdb`
- `n8n-nodes-base.azureStorage` vs `azurestorage`
- `n8n-nodes-base.microsoftSharePoint` vs `microsoftsharepoint`
- `n8n-nodes-base.microsoftGraphSecurity` vs `microsoftgraphsecurity`
- `n8n-nodes-base.microsoftToDo` vs `microsofttodo`
- `n8n-nodes-base.microsoftEntra` vs `microsoftentra`

**Strategy**: Implement with camelCase assumption (matches N8N's TypeScript convention). Add case-insensitive fallback matching if needed after testing with real execution data.
