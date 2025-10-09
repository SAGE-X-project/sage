"""
SAGE Python Client - Basic Usage Example
"""

import asyncio
from sage_client import SAGEClient


async def main():
    # Initialize client
    async with SAGEClient("http://localhost:8080") as client:
        # Register agent
        client_did = "did:sage:ethereum:0xAlice"
        await client.register_agent(client_did, "Alice Agent")
        print(f"Registered: {client_did}")

        # Check server health
        health = await client.health_check()
        print(f"Server status: {health.status}")

        # Get server DID
        server_did = await client.get_server_did()
        print(f"Server DID: {server_did}")

        # Initiate handshake
        session_id = await client.handshake(server_did)
        print(f"Session established: {session_id}")

        # Send message
        message = b"Hello, Server!"
        response = await client.send_message(session_id, message)
        print(f"Response: {response.decode('utf-8')}")

        # Send another message
        message2 = b"How are you?"
        response2 = await client.send_message(session_id, message2)
        print(f"Response 2: {response2.decode('utf-8')}")

        # View active sessions
        sessions = client.get_active_sessions()
        print(f"Active sessions: {len(sessions)}")


if __name__ == "__main__":
    asyncio.run(main())
