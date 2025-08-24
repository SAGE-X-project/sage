"""Base contract class for SAGE contracts"""

from web3 import Web3
from web3.contract import Contract
from typing import Optional, Dict, Any
import json
import os


class ContractBase:
    """Base class for all SAGE contracts"""
    
    def __init__(
        self,
        web3: Web3,
        address: str,
        abi_path: Optional[str] = None,
        private_key: Optional[str] = None
    ):
        self.web3 = web3
        self.address = Web3.to_checksum_address(address)
        self.private_key = private_key
        
        # Load ABI
        if abi_path:
            with open(abi_path, 'r') as f:
                self.abi = json.load(f)
        else:
            self.abi = self._load_default_abi()
        
        # Create contract instance
        self.contract: Contract = self.web3.eth.contract(
            address=self.address,
            abi=self.abi
        )
        
        # Setup account if private key provided
        if private_key:
            self.account = self.web3.eth.account.from_key(private_key)
            self.web3.eth.default_account = self.account.address
    
    def _load_default_abi(self) -> list:
        """Override in subclasses to provide default ABI"""
        raise NotImplementedError("Subclasses must implement _load_default_abi")
    
    def call_function(self, function_name: str, *args, **kwargs) -> Any:
        """Call a read-only contract function"""
        func = getattr(self.contract.functions, function_name)
        return func(*args).call(**kwargs)
    
    def send_transaction(
        self,
        function_name: str,
        *args,
        gas: Optional[int] = None,
        gas_price: Optional[int] = None,
        **kwargs
    ) -> Dict:
        """Send a transaction to the contract"""
        if not self.private_key:
            raise ValueError("Private key required for transactions")
        
        func = getattr(self.contract.functions, function_name)
        
        # Build transaction
        tx = func(*args).build_transaction({
            'from': self.account.address,
            'nonce': self.web3.eth.get_transaction_count(self.account.address),
            'gas': gas or 3000000,
            'gasPrice': gas_price or self.web3.eth.gas_price,
            **kwargs
        })
        
        # Sign and send
        signed_tx = self.account.sign_transaction(tx)
        tx_hash = self.web3.eth.send_raw_transaction(signed_tx.rawTransaction)
        
        # Wait for receipt
        receipt = self.web3.eth.wait_for_transaction_receipt(tx_hash)
        return receipt
    
    def get_events(self, event_name: str, from_block: int = 0, to_block: str = 'latest') -> list:
        """Get contract events"""
        event = getattr(self.contract.events, event_name)
        return event.create_filter(fromBlock=from_block, toBlock=to_block).get_all_entries()
