package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	
	"github.com/sage-x-project/sage/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	fmt.Println("\n SAGE Deployment Verification (Go)")
	fmt.Println("=" + string(make([]byte, 60)))
	
	// 1. Determine network environment
	network := os.Getenv("SAGE_NETWORK")
	if network == "" {
		network = "local"
	}
	fmt.Printf("📍 Network: %s\n", network)
	
	// 2. Load configuration
	cfg, err := config.LoadConfig(network)
	if err != nil {
		log.Fatalf(" Failed to load config: %v", err)
	}
	
	fmt.Println("\n Blockchain Configuration:")
	fmt.Printf("  RPC URL: %s\n", cfg.NetworkRPC)
	fmt.Printf("  Chain ID: %s\n", cfg.ChainID)
	fmt.Printf("  Contract Address: %s\n", cfg.ContractAddr)
	
	// 3. Load deployment info
	deployInfo, err := config.LoadDeploymentInfo(network)
	if err != nil {
		fmt.Printf("  Failed to load deployment info: %v\n", err)
	} else {
		fmt.Println("\n Deployment Info:")
		fmt.Printf("  Deployer: %s\n", deployInfo.Deployer)
		fmt.Printf("  Timestamp: %s\n", deployInfo.Timestamp)
		fmt.Printf("  Registry: %s\n", deployInfo.Contracts.SageRegistryV2.Address)
		fmt.Printf("  Hook: %s\n", deployInfo.Contracts.SageVerificationHook.Address)
		fmt.Printf("  Registered Agents: %d\n", len(deployInfo.Agents))
		
		if len(deployInfo.Agents) > 0 {
			fmt.Println("\nAgent List:")
			for _, agent := range deployInfo.Agents {
				fmt.Printf("  - %s (%s)\n", agent.Name, agent.DID)
			}
		}
	}
	
	// 4. Test blockchain connection
	fmt.Println("\n🔗 Blockchain Connection Test:")
	client, err := ethclient.Dial(cfg.NetworkRPC)
	if err != nil {
		log.Fatalf(" Connection failed: %v", err)
	}
	defer client.Close()
	
	// Check chain ID
	chainID, err := client.ChainID(nil)
	if err != nil {
		fmt.Printf(" Failed to get Chain ID: %v\n", err)
	} else {
		fmt.Printf("   Chain ID: %s\n", chainID)
	}
	
	// Check latest block
	block, err := client.BlockNumber(nil)
	if err != nil {
		fmt.Printf(" Failed to get block number: %v\n", err)
	} else {
		fmt.Printf("   Latest Block: %d\n", block)
	}
	
	// 5. Check contract code
	if cfg.ContractAddr != "" {
		addr := common.HexToAddress(cfg.ContractAddr)
		code, err := client.CodeAt(nil, addr, nil)
		if err != nil {
			fmt.Printf(" Failed to get contract code: %v\n", err)
		} else if len(code) == 0 {
			fmt.Printf("  Contract not deployed or invalid address\n")
		} else {
			fmt.Printf("   Contract Code Size: %d bytes\n", len(code))
		}
	}
	
	// 6. Check environment variables
	fmt.Println("\n Environment Variables Status:")
	envVars := []string{
		"SAGE_REGISTRY_ADDRESS",
		"SAGE_CONTRACT_ADDRESS",
		"SAGE_NETWORK",
		"SAGE_CHAIN_ID",
		"DEPLOYED_CONTRACT_ADDRESS",
	}
	
	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value != "" {
			fmt.Printf("   %s = %s\n", envVar, value)
		} else {
			fmt.Printf("   %s (not set)\n", envVar)
		}
	}
	
	// 7. Result summary
	fmt.Println("\n" + string(make([]byte, 60)) + "=")
	
	if cfg.ContractAddr != "" && err == nil {
		fmt.Println(" Verification Successful!")
		fmt.Println("\n Next Steps:")
		fmt.Println("  1. Test sage-multi-agent")
		fmt.Println("  2. Test frontend integration")
		fmt.Println("  3. Run scenario tests")
	} else {
		fmt.Println("  Partial verification failure")
		fmt.Println("\n Things to Check:")
		fmt.Println("  1. Check if contracts are deployed")
		fmt.Println("  2. Check if environment variables are set")
		fmt.Println("  3. Check if network is running")
	}
	
	// JSON output option
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		output := map[string]interface{}{
			"network": network,
			"config": cfg,
			"deployment": deployInfo,
			"connected": err == nil,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println("\n📄 JSON Output:")
		fmt.Println(string(jsonData))
	}
}