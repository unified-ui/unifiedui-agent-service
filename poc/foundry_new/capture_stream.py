"""Capture raw SSE stream from Microsoft Foundry agent to a file.

Uses the OpenAI SDK with Foundry endpoint for streaming.

Usage:
    1. Create a .env file in this directory with:
       FOUNDRY_PROJECT_ENDPOINT=https://your-resource.services.ai.azure.com/api/projects/your-project
       FOUNDRY_API_KEY=your-api-key
       FOUNDRY_AGENT_NAME=your-agent-name

    2. Install deps:
       pip install openai python-dotenv

    3. Run:
       python capture_stream.py "Your message here"

    The raw events will be appended to stream_output.txt.
    If API key auth fails (403), a bearer token from Azure AD is required.
    Run with --interactive to use browser-based login instead.
"""

import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

from dotenv import load_dotenv
from openai import OpenAI

SCRIPT_DIR = Path(__file__).parent
load_dotenv(SCRIPT_DIR / ".env")

ENDPOINT = os.environ["FOUNDRY_PROJECT_ENDPOINT"].rstrip("/")
API_KEY = os.environ["FOUNDRY_API_KEY"]
AGENT_NAME = os.environ["FOUNDRY_AGENT_NAME"]
API_VERSION = os.getenv("FOUNDRY_API_VERSION", "2025-11-15-preview")

OUTPUT_FILE = SCRIPT_DIR / "stream_output.txt"


def get_api_key(use_interactive: bool) -> str:
    """Get key or bearer token for authentication."""
    if use_interactive:
        from azure.identity import InteractiveBrowserCredential
        credential = InteractiveBrowserCredential()
        token = credential.get_token("https://ai.azure.com/.default")
        return token.token
    return API_KEY


def capture_stream(message: str, use_interactive: bool = False) -> None:
    """Send a message and capture the raw streaming events."""
    api_key = get_api_key(use_interactive)
    client = OpenAI(
        base_url=f"{ENDPOINT}/openai",
        api_key=api_key,
        default_query={"api-version": API_VERSION},
        default_headers={"api-key": api_key},
    )

    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")
    separator = f"\n{'='*80}\n"
    header_text = (
        f"{separator}"
        f"CAPTURE: {timestamp}\n"
        f"AGENT:   {AGENT_NAME}\n"
        f"MESSAGE: {message}\n"
        f"{separator}\n"
    )

    with open(OUTPUT_FILE, "a", encoding="utf-8") as f:
        f.write(header_text)

        print(f"Sending message to agent '{AGENT_NAME}': {message}")
        print(f"Streaming to {OUTPUT_FILE} ...")

        response_stream = client.responses.create(
            model="",
            input=message,
            stream=True,
            extra_body={
                "agent": {"type": "agent_reference", "name": AGENT_NAME},
            },
        )

        for event in response_stream:
            event_type = type(event).__name__
            event_dict = {}
            for attr in dir(event):
                if not attr.startswith("_"):
                    try:
                        val = getattr(event, attr)
                        if not callable(val):
                            event_dict[attr] = repr(val)
                    except Exception:
                        pass

            line = f"[{event_type}] {json.dumps(event_dict, default=str)}"
            f.write(line + "\n")
            f.flush()
            print(f"  {event_type}: {json.dumps(event_dict, default=str)[:200]}")

        f.write("\n--- END ---\n")

    print(f"\nDone. Output appended to {OUTPUT_FILE}")


if __name__ == "__main__":
    interactive = "--interactive" in sys.argv
    args = [a for a in sys.argv[1:] if a != "--interactive"]

    if len(args) < 1:
        print('Usage: python capture_stream.py "Your message" [--interactive]')
        sys.exit(1)

    user_message = args[0]
    capture_stream(user_message, use_interactive=interactive)
