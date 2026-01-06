import asyncio

from dotenv import load_dotenv
from agent_config import load_config
from agent import ReACTAgent, UnifiedUIMessage


load_dotenv()


CONFIG_PATH = "../config/config_2_mcp.json"


async def main():
    """Main function to create and run the agent based on config."""
    # Setup
    messages = []
    config = load_config(CONFIG_PATH)
    
    agent = ReACTAgent(config)
    await agent.initialize()

    while True:
        user_input = input("\nUser: ")
        if user_input.lower() in {"exit", "quit"}:
            break

        messages.append(UnifiedUIMessage(role="user", content=user_input))

        print("Agent:", end=" ", flush=True)
        agent_response_content = ""
        async for response in agent.invoke_stream(messages):
            if response.get("type") == "TEXT_STREAM":
                agent_response_content += response.get("content", "")
            if response.get("type") == "TOOL_START":
                tool_name = response.get("content", "")
                print(f"\n[Tool Started: {tool_name}]", end=" ", flush=True)
            if response.get("type") == "TOOL_END":
                tool_output = response.get("content", "")
                print(f"\n[Tool Output: {tool_output}]", end=" ", flush=True)

        print()  # Newline after streaming
        # Append agent's response to messages
        if agent_response_content:
            messages.append(UnifiedUIMessage(role="assistant", content=agent_response_content))


if __name__ == "__main__":
    asyncio.run(main())
