#!/usr/bin/env python3
"""Example usage of SAGE contract bindings"""

from web3 import Web3
from sage_contracts import SageRegistryV2, SageVerificationHook
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

def main():
    # Connect to Kaia testnet
    w3 = Web3(Web3.HTTPProvider('https://public-en-kairos.node.kaia.io'))
    
    if not w3.is_connected():
        print("Failed to connect to network")
        return
    
    print(f"Connected to network. Chain ID: {w3.eth.chain_id}")
    
    # Contract address (replace with actual deployed address)
    registry_address = os.getenv('SAGE_REGISTRY_ADDRESS', '0x...')
    
    # Create contract instance (read-only)
    registry = SageRegistryV2(w3, registry_address)
    
    # Get owner
    owner = registry.owner()
    print(f"Contract owner: {owner}")
    
    # Get agent by DID
    did = "did:sage:example"
    try:
        agent = registry.get_agent_by_did(did)
        print(f"Agent name: {agent['name']}")
        print(f"Agent active: {agent['active']}")
    except Exception as e:
        print(f"Agent not found: {e}")
    
    # For write operations, provide private key
    # private_key = os.getenv('PRIVATE_KEY')
    # registry_with_signer = SageRegistryV2(w3, registry_address, private_key=private_key)
    # 
    # # Register agent
    # receipt = registry_with_signer.register_agent(
    #     did="did:sage:test",
    #     name="Test Agent",
    #     description="Test Description",
    #     endpoint="https://test.example.com",
    #     public_key="0x04...",
    #     capabilities='["test"]',
    #     signature="0x..."
    # )
    # print(f"Transaction hash: {receipt['transactionHash'].hex()}")


if __name__ == "__main__":
    main()
