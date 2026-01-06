import os
import asyncio

from abc import ABC, abstractmethod
from typing import AsyncGenerator
from pydantic import BaseModel
from langchain_openai import AzureChatOpenAI
from langchain.agents import create_agent
from langchain.tools import tool
from langchain_core.callbacks.streaming_stdout import StreamingStdOutCallbackHandler
from langchain_core.messages import HumanMessage, AIMessage, SystemMessage

try:
    from mcp import ClientSession, StdioServerParameters
    from mcp.client.stdio import stdio_client
    MCP_AVAILABLE = True
except ImportError:
    MCP_AVAILABLE = False
    print("Warning: MCP not installed. Install with: pip install mcp")

from agent_config import AgentConfig, MCPServerConfig


class UnifiedUIMessage(BaseModel):
    role: str
    content: str


class BaseUnifiedUIAgent(ABC):
    def __init__(self, config: AgentConfig):
        """Initialize the Base Unified UI Agent with the given configuration."""
        self._config = config

    @abstractmethod
    def invoke(self, messages: list[UnifiedUIMessage]) -> dict:
        pass

    @abstractmethod
    async def invoke_stream(self, messages: list[UnifiedUIMessage]) -> AsyncGenerator[dict, None]:
        pass


class ReACTAgent(BaseUnifiedUIAgent):
    def __init__(self, config: AgentConfig):
        """Initialize the ReACT agent with the given configuration."""
        self._config = config
        self.llm = None
        self.agent = None
    
    async def initialize(self):
        """Async initialization of the agent."""
        await self._setup_agent(self._config)
        return self

    def invoke(self, messages: list[UnifiedUIMessage]) -> dict:
        """Invoke the agent with the given messages."""
        langchain_messages = self._convert_to_langchain_messages(messages)
        return self.agent.invoke(
            {"messages": langchain_messages},
            version="v1"
        )

    async def invoke_stream(self, messages: list[UnifiedUIMessage]) -> AsyncGenerator[dict, None]:
        """Invoke the agent in streaming mode with the given messages."""
        langchain_messages = self._convert_to_langchain_messages(messages)
        async for event in self.agent.astream_events(
            {"messages": langchain_messages},
            version="v1"
        ):
            kind = event["event"]
            if kind == "on_chain_start":
                # print(f"on_chain_start: {event}")
                pass
            elif kind == "on_chain_end":
                # print(f"on_chain_end: {event}")
                pass
            elif kind == "on_chat_model_stream":
                content = event["data"]["chunk"].content
                # print(content, end="", flush=True)
                yield {"type": "TEXT_STREAM", "content": content}
            elif kind == "on_tool_start":
                # print(f"\nUsing tool: {event}")
                yield {"type": "TOOL_START", "content": event["name"]}
            elif kind == "on_tool_end":
                # print(f"Tool result: {event}")
                yield {"type": "TOOL_END", "content": event["data"]["output"]}
            

    async def _setup_agent(self, config: AgentConfig) -> None:
        """Setup the LangChain agent based on the configuration."""
        api_key = os.getenv(config.settings.llm_credentials.api_key)
        self.llm = AzureChatOpenAI(
            api_key=api_key,
            azure_endpoint=config.settings.llm_credentials.endpoint,
            azure_deployment=config.settings.llm_credentials.deployment_name,
            api_version=config.settings.llm_credentials.api_version,
            streaming=True,
            callbacks=[StreamingStdOutCallbackHandler()]
        )
        
        # Load tools from config
        tools = await self._load_tools(config)
        self.agent = create_agent(self.llm, tools=tools)

    async def _load_tools(self, config: AgentConfig) -> list:
        """Load tools from configuration."""
        tools = []
        
        for tool_config in config.settings.tools:
            if tool_config.type == "mcp_server":
                if MCP_AVAILABLE:
                    mcp_tools = await self._load_mcp_tools(tool_config)
                    tools.extend(mcp_tools)
                else:
                    print(f"Skipping MCP tool {tool_config.name} - MCP not installed")
            # Add more tool types here as needed
        
        return tools

    async def _load_mcp_tools(self, tool_config) -> list:
        """Load tools from an MCP server."""
        from langchain_core.tools import StructuredTool
        
        mcp_config = tool_config.mcp_config
        if isinstance(mcp_config, dict):
            mcp_config = MCPServerConfig(**mcp_config)
        
        tools = []
        
        try:
            # Start MCP server and get available tools
            mcp_tools_info = await self._get_mcp_tools_async(mcp_config, tool_config.name)
            
            # Create LangChain tools from MCP tool definitions
            for tool_info in mcp_tools_info:
                # Create a closure to capture the current tool info
                def make_tool_func(ti, mc, tn):
                    async def tool_func(**kwargs) -> str:
                        """Execute MCP tool with given parameters."""
                        return await self._call_mcp_tool_async(mc, tn, ti['name'], kwargs)
                    
                    # Make it synchronous for LangChain
                    def sync_tool_func(**kwargs) -> str:
                        return asyncio.run(tool_func(**kwargs))
                    
                    return sync_tool_func
                
                # Create the structured tool
                lc_tool = StructuredTool.from_function(
                    func=make_tool_func(tool_info, mcp_config, tool_config.name),
                    name=tool_info['name'],
                    description=tool_info['description'],
                    args_schema=tool_info.get('input_schema')
                )
                tools.append(lc_tool)
            
            print(f"Loaded {len(tools)} MCP tools from {tool_config.name}")
            
        except Exception as e:
            print(f"Error loading MCP tools from {tool_config.name}: {e}")
            import traceback
            traceback.print_exc()
        
        return tools
    
    async def _get_mcp_tools_async(self, mcp_config: MCPServerConfig, server_name: str) -> list:
        """Get available tools from MCP server."""
        server_params = StdioServerParameters(
            command=mcp_config.command,
            args=mcp_config.args,
            env=mcp_config.env
        )
        
        tools_info = []
        async with stdio_client(server_params) as (read, write):
            async with ClientSession(read, write) as session:
                await session.initialize()
                
                # List available tools
                tools_response = await session.list_tools()
                
                for tool in tools_response.tools:
                    tools_info.append({
                        'name': tool.name,
                        'description': tool.description or f"MCP tool: {tool.name}",
                        'input_schema': tool.inputSchema if hasattr(tool, 'inputSchema') else None
                    })
        
        return tools_info
    
    async def _call_mcp_tool_async(self, mcp_config: MCPServerConfig, server_name: str, tool_name: str, arguments: dict) -> str:
        """Call an MCP tool with the given arguments."""
        server_params = StdioServerParameters(
            command=mcp_config.command,
            args=mcp_config.args,
            env=mcp_config.env
        )
        
        result = ""
        async with stdio_client(server_params) as (read, write):
            async with ClientSession(read, write) as session:
                await session.initialize()
                
                # Call the tool
                response = await session.call_tool(tool_name, arguments=arguments)
                
                # Extract content from response
                if hasattr(response, 'content') and response.content:
                    for content in response.content:
                        if hasattr(content, 'text'):
                            result += content.text
                        elif isinstance(content, dict) and 'text' in content:
                            result += content['text']
                else:
                    result = str(response)
        
        return result

    def _convert_to_langchain_messages(self, messages: list[UnifiedUIMessage]) -> list:
        """Convert UnifiedUIMessage to LangChain message objects."""
        langchain_messages = []
        
        # Add system instructions as first message
        if self._config.settings.instructions:
            langchain_messages.append(SystemMessage(content=self._config.settings.instructions))
        
        for msg in messages:
            if msg.role == "user":
                langchain_messages.append(HumanMessage(content=msg.content))
            elif msg.role in ["assistant", "agent", "ai"]:
                langchain_messages.append(AIMessage(content=msg.content))
            else:
                # Fallback to HumanMessage for unknown roles
                langchain_messages.append(HumanMessage(content=msg.content))
        return langchain_messages
