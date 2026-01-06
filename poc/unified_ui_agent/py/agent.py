import os

from abc import ABC, abstractmethod
from typing import AsyncGenerator
from pydantic import BaseModel
from langchain_openai import AzureChatOpenAI
from langchain.agents import create_agent
from langchain.tools import tool
from langchain_core.callbacks.streaming_stdout import StreamingStdOutCallbackHandler
from langchain_core.messages import HumanMessage, AIMessage, SystemMessage

from agent_config import AgentConfig


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
        self._setup_agent(config)

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
            

    def _setup_agent(self, config: AgentConfig) -> None:
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
        # Additional setup for tools and prompts can be added here
        self.agent = create_agent(self.llm, tools=[])

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
