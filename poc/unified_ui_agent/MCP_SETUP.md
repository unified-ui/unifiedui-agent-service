# MCP Server Setup für LangChain Agent

## Installation

### 1. LangChain Packages (ohne langchain-mcp wegen Dependency Konflikt)
```bash
pip install --upgrade langchain langchain-openai langchain-core langgraph
```

### 2. MCP SDK (direkt)
```bash
pip install mcp
```

### 2. MCP Filesystem Server (Node.js)
Der Server wird automatisch via `npx` geladen (kein Installation nötig).

Alternative MCP Server zum Testen:
- **Filesystem**: `@modelcontextprotocol/server-filesystem`
- **Everything**: `@modelcontextprotocol/server-everything` (Demo-Server mit allen Features)
- **Brave Search**: `@modelcontextprotocol/server-brave-search`
- **Fetch**: `@modelcontextprotocol/server-fetch`

## Verwendung

### Mit config_2_mcp.json:
```bash
cd /Users/enricogoerlitz/Developer/repos/unified-ui-agent-service/poc/unified_ui_agent/py
python main.py --config ../config/config_2_mcp.json
```

### Config Struktur:
```json
{
  "tools": [
    {
      "type": "mcp_server",
      "name": "filesystem",
      "description": "Use for file operations",
      "mcp_config": {
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
        "env": null
      }
    }
  ]
}
```

## Andere MCP Server Beispiele

### Everything Server (Demo):
```json
{
  "type": "mcp_server",
  "name": "everything",
  "description": "Demo server with all MCP features",
  "mcp_config": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-everything"],
    "env": null
  }
}
```

### Brave Search:
```json
{
  "type": "mcp_server",
  "name": "brave_search",
  "description": "Web search via Brave API",
  "mcp_config": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-brave-search"],
    "env": {
      "BRAVE_API_KEY": "YOUR_API_KEY"
    }
  }
}
```

## Full Async MCP Integration (TODO)

Die aktuelle Implementierung ist ein Placeholder. Für vollständige MCP Integration:

```python
async def _load_mcp_tools_async(self, tool_config) -> list:
    """Load tools from MCP server asynchronously."""
    from langchain_core.tools import StructuredTool
    
    server_params = StdioServerParameters(
        command=tool_config.mcp_config.command,
        args=tool_config.mcp_config.args,
        env=tool_config.mcp_config.env
    )
    
    tools = []
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            
            # List available tools
            response = await session.list_tools()
            
            for mcp_tool in response.tools:
                # Create LangChain tool wrapper
                async def tool_func(**kwargs):
                    result = await session.call_tool(mcp_tool.name, arguments=kwargs)
                    return result.content
                
                lc_tool = StructuredTool.from_function(
                    func=tool_func,
                    name=mcp_tool.name,
                    description=mcp_tool.description
                )
                tools.append(lc_tool)
    
    return tools
```

## Testing

```bash
# Test mit Filesystem MCP Server
User: Create a file /tmp/test.txt with content "Hello MCP"
User: List all files in /tmp
User: Read the content of /tmp/test.txt
```
