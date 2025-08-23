#!/usr/bin/env node

/**
 * Generate Python bindings from contract ABIs
 * Uses web3.py contract factory pattern
 */

const fs = require('fs');
const path = require('path');

// Color codes for console output
const colors = {
  reset: "\x1b[0m",
  bright: "\x1b[1m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  red: "\x1b[31m",
  cyan: "\x1b[36m"
};

function log(message, color = "reset") {
  console.log(colors[color] + message + colors.reset);
}

async function generatePythonBindings() {
  try {
    log("\n🐍 Generating Python Bindings for Smart Contracts", "cyan");
    log("=" .repeat(50), "bright");

    // Create Python bindings directory
    const pythonDir = path.join(__dirname, '..', 'bindings', 'python');
    const sageDir = path.join(pythonDir, 'sage_contracts');
    
    if (!fs.existsSync(sageDir)) {
      fs.mkdirSync(sageDir, { recursive: true });
      log("✅ Created bindings/python directory structure", "green");
    }

    // Contracts to generate bindings for
    const contracts = [
      {
        name: 'SageRegistryV2',
        abi: 'abi/SageRegistryV2.abi.json',
        bin: 'artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json',
        className: 'SageRegistryV2'
      },
      {
        name: 'SageVerificationHook',
        abi: 'abi/SageVerificationHook.abi.json',
        bin: 'artifacts/contracts/SageVerificationHook.sol/SageVerificationHook.json',
        className: 'SageVerificationHook'
      }
    ];

    // Create __init__.py
    const initContent = `"""
SAGE Smart Contract Python Bindings
Generated: ${new Date().toISOString()}
"""

from .registry import SageRegistryV2
from .hook import SageVerificationHook
from .base import ContractBase

__all__ = ['SageRegistryV2', 'SageVerificationHook', 'ContractBase']
__version__ = '1.0.0'
`;

    fs.writeFileSync(path.join(sageDir, '__init__.py'), initContent);

    // Create base contract class
    const baseContent = `"""Base contract class for SAGE contracts"""

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
`;

    fs.writeFileSync(path.join(sageDir, 'base.py'), baseContent);
    log("✅ Created base.py", "green");

    // Generate Python bindings for each contract
    for (const contract of contracts) {
      log(`\n📦 Processing ${contract.name}...`, "blue");

      const abiPath = path.join(__dirname, '..', contract.abi);
      const binPath = path.join(__dirname, '..', contract.bin);

      // Check if files exist
      if (!fs.existsSync(abiPath)) {
        log(`  ⚠️  ABI not found: ${contract.abi}`, "yellow");
        log(`  Run 'npm run extract-abi' first`, "yellow");
        continue;
      }

      // Read ABI
      const abi = JSON.parse(fs.readFileSync(abiPath, 'utf8'));
      
      // Read bytecode
      let bytecode = '';
      if (fs.existsSync(binPath)) {
        const artifact = JSON.parse(fs.readFileSync(binPath, 'utf8'));
        bytecode = artifact.bytecode;
      }

      // Generate Python class
      const fileName = contract.name === 'SageRegistryV2' ? 'registry.py' : 'hook.py';
      const pythonContent = generatePythonClass(contract, abi, bytecode);
      
      fs.writeFileSync(path.join(sageDir, fileName), pythonContent);
      log(`  ✅ Generated ${fileName}`, "green");
    }

    // Create example script
    const exampleContent = `#!/usr/bin/env python3
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
`;

    fs.writeFileSync(path.join(pythonDir, 'example.py'), exampleContent);
    log("✅ Created example.py", "green");

    // Create requirements.txt
    const requirementsContent = `web3>=6.11.0
eth-account>=0.10.0
python-dotenv>=1.0.0
hexbytes>=0.3.0
`;

    fs.writeFileSync(path.join(pythonDir, 'requirements.txt'), requirementsContent);
    log("✅ Created requirements.txt", "green");

    // Create setup.py
    const setupContent = `from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="sage-contracts",
    version="1.0.0",
    author="SAGE Team",
    description="Python bindings for SAGE smart contracts",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/sage-x-project/sage",
    packages=find_packages(),
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.8",
    install_requires=[
        "web3>=6.11.0",
        "eth-account>=0.10.0",
        "hexbytes>=0.3.0",
    ],
)
`;

    fs.writeFileSync(path.join(pythonDir, 'setup.py'), setupContent);
    log("✅ Created setup.py", "green");

    // Create README
    const readmeContent = `# Python Bindings for SAGE Contracts

## Installation

\`\`\`bash
pip install -r requirements.txt
\`\`\`

## Usage

\`\`\`python
from web3 import Web3
from sage_contracts import SageRegistryV2

# Connect to network
w3 = Web3(Web3.HTTPProvider('https://public-en-kairos.node.kaia.io'))

# Load contract (read-only)
registry = SageRegistryV2(w3, contract_address)

# Read data
agent = registry.get_agent_by_did("did:sage:example")
print(f"Agent: {agent}")

# Write data (requires private key)
registry_with_key = SageRegistryV2(w3, contract_address, private_key=private_key)
receipt = registry_with_key.register_agent(
    did="did:sage:test",
    name="Test Agent",
    description="Test",
    endpoint="https://test.com",
    public_key="0x04...",
    capabilities='["test"]',
    signature="0x..."
)
\`\`\`

## Development

\`\`\`bash
# Install in development mode
pip install -e .

# Run example
python example.py
\`\`\`

## Regenerate Bindings

\`\`\`bash
npm run generate:python
\`\`\`
`;

    fs.writeFileSync(path.join(pythonDir, 'README.md'), readmeContent);
    log("✅ Created README.md", "green");

    log("\n" + "=".repeat(50), "bright");
    log("✨ Python binding generation complete!", "green");
    log("\nGenerated files:", "yellow");
    log(`  📁 ${pythonDir}/`, "cyan");
    log("  📄 sage_contracts/registry.py - SageRegistryV2 binding", "cyan");
    log("  📄 sage_contracts/hook.py - SageVerificationHook binding", "cyan");
    log("  📄 sage_contracts/base.py - Base contract class", "cyan");
    log("  📄 example.py - Usage example", "cyan");
    log("  📄 requirements.txt - Dependencies", "cyan");
    log("  📄 setup.py - Package setup", "cyan");

  } catch (error) {
    log("\n❌ Error generating Python bindings:", "red");
    console.error(error);
    process.exit(1);
  }
}

// Helper function to generate Python class
function generatePythonClass(contract, abi, bytecode) {
  const functions = abi.filter(item => item.type === 'function');
  const events = abi.filter(item => item.type === 'event');
  
  const readFunctions = functions.filter(f => 
    f.stateMutability === 'view' || f.stateMutability === 'pure'
  );
  const writeFunctions = functions.filter(f => 
    f.stateMutability !== 'view' && f.stateMutability !== 'pure'
  );

  let content = `"""${contract.name} Python binding"""

from typing import Optional, Dict, Any, List
from web3 import Web3
from .base import ContractBase
import json
import os


class ${contract.className}(ContractBase):
    """${contract.name} contract interface"""
    
    # Contract bytecode for deployment
    BYTECODE = "${bytecode}"
    
    def __init__(self, web3: Web3, address: str, private_key: Optional[str] = None):
        super().__init__(web3, address, private_key=private_key)
    
    def _load_default_abi(self) -> list:
        """Load the default ABI for ${contract.name}"""
        abi_path = os.path.join(
            os.path.dirname(__file__),
            '..',
            '..',
            'abi',
            '${contract.name}.abi.json'
        )
        if os.path.exists(abi_path):
            with open(abi_path, 'r') as f:
                return json.load(f)
        return ${JSON.stringify(abi)}
    
    @classmethod
    def deploy(
        cls,
        web3: Web3,
        private_key: str,
        *constructor_args,
        gas: Optional[int] = None,
        gas_price: Optional[int] = None
    ) -> '${contract.className}':
        """Deploy a new instance of the contract"""
        account = web3.eth.account.from_key(private_key)
        
        # Create contract factory
        contract = web3.eth.contract(
            abi=cls(web3, '0x' + '0' * 40)._load_default_abi(),
            bytecode=cls.BYTECODE
        )
        
        # Build constructor transaction
        tx = contract.constructor(*constructor_args).build_transaction({
            'from': account.address,
            'nonce': web3.eth.get_transaction_count(account.address),
            'gas': gas or 5000000,
            'gasPrice': gas_price or web3.eth.gas_price
        })
        
        # Sign and send
        signed_tx = account.sign_transaction(tx)
        tx_hash = web3.eth.send_raw_transaction(signed_tx.rawTransaction)
        
        # Wait for receipt
        receipt = web3.eth.wait_for_transaction_receipt(tx_hash)
        
        return cls(web3, receipt.contractAddress, private_key=private_key)
`;

  // Add read functions
  content += `\n    # Read-only functions\n`;
  for (const func of readFunctions) {
    const pythonName = func.name.replace(/([A-Z])/g, '_$1').toLowerCase().replace(/^_/, '');
    const params = func.inputs.map(i => i.name || 'arg').join(', ');
    
    content += `    
    def ${pythonName}(self${params ? ', ' + params : ''}):
        """${func.name} - Read function"""
        return self.call_function('${func.name}'${params ? ', ' + params : ''})
`;
  }

  // Add write functions
  content += `\n    # State-changing functions\n`;
  for (const func of writeFunctions) {
    const pythonName = func.name.replace(/([A-Z])/g, '_$1').toLowerCase().replace(/^_/, '');
    const params = func.inputs.map(i => i.name || 'arg').join(', ');
    
    content += `    
    def ${pythonName}(self${params ? ', ' + params : ''}, gas: Optional[int] = None, gas_price: Optional[int] = None) -> Dict:
        """${func.name} - Write function (requires private key)"""
        return self.send_transaction('${func.name}'${params ? ', ' + params : ''}, gas=gas, gas_price=gas_price)
`;
  }

  // Add event getters
  if (events.length > 0) {
    content += `\n    # Event functions\n`;
    for (const event of events) {
      const pythonName = 'get_' + event.name.replace(/([A-Z])/g, '_$1').toLowerCase().replace(/^_/, '') + '_events';
      
      content += `    
    def ${pythonName}(self, from_block: int = 0, to_block: str = 'latest') -> List[Dict]:
        """Get ${event.name} events"""
        return self.get_events('${event.name}', from_block, to_block)
`;
    }
  }

  return content;
}

// Run generation
generatePythonBindings();