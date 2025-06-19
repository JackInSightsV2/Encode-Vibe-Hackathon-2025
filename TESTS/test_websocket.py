import asyncio
import pytest
import websockets
import json
from typing import Optional, List

# Test configuration
WS_URI = "ws://localhost:8080/ws"
TEST_MESSAGE = "Hello from Python test!"

@pytest.mark.asyncio
async def test_websocket_connection():
    """Test WebSocket connection and message exchange."""
    messages_received = []
    
    async with websockets.connect(WS_URI) as websocket:
        print(f"Connected to WebSocket at {WS_URI}")
        
        # Send test message
        await websocket.send(TEST_MESSAGE)
        print(f"Sent: {TEST_MESSAGE}")
        
        # Receive message with timeout
        try:
            response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
            messages_received.append(response)
            print(f"Received: {response}")
        except asyncio.TimeoutError:
            print("No response received within timeout period")
    
    # Basic assertion - at least one message should be received
    assert len(messages_received) > 0, "No messages received from the server"
    
    # Print all received messages for debugging
    for i, msg in enumerate(messages_received, 1):
        print(f"Message {i}: {msg}")
    
    # If you expect a specific response format, you can add more assertions here
    # For example, if the server echoes the message back:
    # assert TEST_MESSAGE in messages_received[0]

# This allows running the test directly with python
if __name__ == "__main__":
    import sys
    import asyncio
    
    async def main():
        try:
            await test_websocket_connection()
        except Exception as e:
            print(f"Test failed: {e}", file=sys.stderr)
            sys.exit(1)
    
    asyncio.run(main())
