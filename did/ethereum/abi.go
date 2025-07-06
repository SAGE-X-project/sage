package ethereum

// SageRegistryABI is the ABI of the SageRegistry contract
const SageRegistryABI = `[
	{
		"inputs": [
			{"name": "did", "type": "string"},
			{"name": "name", "type": "string"},
			{"name": "description", "type": "string"},
			{"name": "endpoint", "type": "string"},
			{"name": "publicKey", "type": "bytes"},
			{"name": "capabilities", "type": "string"},
			{"name": "signature", "type": "bytes"}
		],
		"name": "registerAgent",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "did", "type": "string"}
		],
		"name": "getAgent",
		"outputs": [
			{"name": "exists", "type": "bool"},
			{"name": "name", "type": "string"},
			{"name": "description", "type": "string"},
			{"name": "endpoint", "type": "string"},
			{"name": "publicKey", "type": "bytes"},
			{"name": "capabilities", "type": "string"},
			{"name": "owner", "type": "address"},
			{"name": "isActive", "type": "bool"},
			{"name": "createdAt", "type": "uint256"},
			{"name": "updatedAt", "type": "uint256"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "did", "type": "string"},
			{"name": "name", "type": "string"},
			{"name": "description", "type": "string"},
			{"name": "endpoint", "type": "string"},
			{"name": "capabilities", "type": "string"},
			{"name": "signature", "type": "bytes"}
		],
		"name": "updateAgent",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "did", "type": "string"},
			{"name": "signature", "type": "bytes"}
		],
		"name": "deactivateAgent",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "owner", "type": "address"}
		],
		"name": "getAgentsByOwner",
		"outputs": [
			{"name": "", "type": "string[]"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "did", "type": "string"},
			{"indexed": true, "name": "owner", "type": "address"},
			{"indexed": false, "name": "name", "type": "string"}
		],
		"name": "AgentRegistered",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "did", "type": "string"},
			{"indexed": false, "name": "updatedFields", "type": "string[]"}
		],
		"name": "AgentUpdated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "did", "type": "string"}
		],
		"name": "AgentDeactivated",
		"type": "event"
	}
]`