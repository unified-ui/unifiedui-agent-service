"""
Simple FastMCP Calculator Server with Bearer Token Authentication.

Run with: python server.py
"""
import os
from functools import wraps
from fastmcp import FastMCP
from starlette.requests import Request
from starlette.responses import Response


# Initialize FastMCP server
mcp = FastMCP("Calculator MCP Server")

# Simple token for authentication
VALID_TOKEN = os.getenv("MCP_AUTH_TOKEN", "MyToken")


def check_auth(request: Request) -> bool:
    """Check if the request has a valid Bearer token."""
    auth_header = request.headers.get("Authorization", "")
    if auth_header.startswith("Bearer "):
        token = auth_header[7:]  # Remove "Bearer " prefix
        return token == VALID_TOKEN
    return False


# Calculator Tools
@mcp.tool()
def add(a: float, b: float) -> float:
    """Add two numbers together.
    
    Args:
        a: First number
        b: Second number
    
    Returns:
        The sum of a and b
    """
    return a + b


@mcp.tool()
def subtract(a: float, b: float) -> float:
    """Subtract the second number from the first.
    
    Args:
        a: First number (minuend)
        b: Second number (subtrahend)
    
    Returns:
        The difference (a - b)
    """
    return a - b


@mcp.tool()
def multiply(a: float, b: float) -> float:
    """Multiply two numbers together.
    
    Args:
        a: First number
        b: Second number
    
    Returns:
        The product of a and b
    """
    return a * b


@mcp.tool()
def divide(a: float, b: float) -> float:
    """Divide the first number by the second.
    
    Args:
        a: Numerator (dividend)
        b: Denominator (divisor)
    
    Returns:
        The quotient (a / b)
    
    Raises:
        ValueError: If b is zero
    """
    if b == 0:
        raise ValueError("Cannot divide by zero!")
    return a / b


if __name__ == "__main__":
    import uvicorn
    
    print("Starting Calculator MCP Server on http://localhost:8000")
    print(f"SSE endpoint: http://localhost:8000/sse")
    print(f"Auth Token: {VALID_TOKEN}")
    print("\nAvailable tools: add, subtract, multiply, divide")
    
    # Run with SSE transport
    mcp.run(transport="sse", host="0.0.0.0", port=8000)
