import json

from pydantic import BaseModel


class LLMCredentials(BaseModel):
    type: str
    deployment_name: str
    api_version: str
    endpoint: str
    api_key: str


class ToolConfig(BaseModel):
    type: str
    name: str
    trigger_description: str
    mcp_config: dict


class Settings(BaseModel):
    agent_version: str
    agent_type: str
    instructions: str
    llm_credentials: LLMCredentials
    tools: list[ToolConfig]


class User(BaseModel):
    id: str
    display_name: str
    principal_name: str
    mail: str


class AgentConfig(BaseModel):
    docversion: str
    type: str
    tenant_id: str
    application_id: str
    settings: Settings
    user: User


def load_config(path: str) -> AgentConfig:
    """Load configuration from a JSON file and parse into Pydantic model."""
    with open(path, "r") as file:
        data = json.load(file)
    return AgentConfig(**data)