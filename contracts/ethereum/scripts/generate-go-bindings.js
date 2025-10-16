#!/usr/bin/env node

/**
 * Generate Go bindings from contract ABIs
 * Requires: go-ethereum abigen tool
 * Install: go install github.com/ethereum/go-ethereum/cmd/abigen@latest
 */

const fs = require('fs');
const path = require('path');
const { execSync, execFileSync } = require('child_process');

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

async function generateGoBindings() {
  try {
    log("\n Generating Go Bindings for Smart Contracts", "cyan");
    log("=" .repeat(50), "bright");

    // Check if abigen is installed
    try {
      execSync('which abigen', { stdio: 'ignore' });
    } catch (error) {
      log("\n abigen not found!", "red");
      log("\nPlease install abigen first:", "yellow");
      log("  go install github.com/ethereum/go-ethereum/cmd/abigen@latest", "cyan");
      log("\nOr on macOS with Homebrew:", "yellow");
      log("  brew install ethereum", "cyan");
      process.exit(1);
    }

    // Create go bindings directory
    const goDir = path.join(__dirname, '..', 'bindings', 'go');
    if (!fs.existsSync(goDir)) {
      fs.mkdirSync(goDir, { recursive: true });
      log(" Created bindings/go directory", "green");
    }

    // Contracts to generate bindings for
    const contracts = [
      {
        name: 'SageRegistryV2',
        abi: 'abi/SageRegistryV2.abi.json',
        bin: 'artifacts/contracts/SageRegistryV2.sol/SageRegistryV2.json',
        pkg: 'registry',
        type: 'SageRegistryV2'
      },
      {
        name: 'SageVerificationHook',
        abi: 'abi/SageVerificationHook.abi.json',
        bin: 'artifacts/contracts/SageVerificationHook.sol/SageVerificationHook.json',
        pkg: 'hook',
        type: 'SageVerificationHook'
      }
    ];

    // Generate Go bindings
    for (const contract of contracts) {
      log(`\n Processing ${contract.name}...`, "blue");

      const abiPath = path.join(__dirname, '..', contract.abi);
      const binPath = path.join(__dirname, '..', contract.bin);
      const outputPath = path.join(goDir, `${contract.pkg}.go`);

      // Check if files exist
      if (!fs.existsSync(abiPath)) {
        log(`    ABI not found: ${contract.abi}`, "yellow");
        log(`  Run 'npm run extract-abi' first`, "yellow");
        continue;
      }

      // Get bytecode from artifact
      let bytecode = '';
      if (fs.existsSync(binPath)) {
        const artifact = JSON.parse(fs.readFileSync(binPath, 'utf8'));
        bytecode = artifact.bytecode;
      }

      // Create temporary bin file with bytecode
      const tempBinPath = path.join(__dirname, '..', 'temp.bin');
      fs.writeFileSync(tempBinPath, bytecode.replace('0x', ''));

      // Generate Go binding using abigen
      // Use execFileSync to prevent shell injection
      try {
        execFileSync('abigen', [
          `--abi=${abiPath}`,
          `--bin=${tempBinPath}`,
          `--pkg=${contract.pkg}`,
          `--type=${contract.type}`,
          `--out=${outputPath}`
        ], { stdio: 'pipe' });
        log(`   Generated ${contract.pkg}.go`, "green");
        
        // Clean up temp file
        fs.unlinkSync(tempBinPath);
        
        // Show file info
        const stats = fs.statSync(outputPath);
        log(`  📄 ${outputPath}`, "cyan");
        log(`   Size: ${(stats.size / 1024).toFixed(2)} KB`, "cyan");
      } catch (error) {
        log(`   Failed to generate binding for ${contract.name}`, "red");
        console.error(error.message);
        // Clean up temp file if exists
        if (fs.existsSync(tempBinPath)) {
          fs.unlinkSync(tempBinPath);
        }
      }
    }

    // Create go.mod file
    const goModContent = `module github.com/sage-x-project/sage/contracts

go 1.21

require (
    github.com/ethereum/go-ethereum v1.13.5
)
`;

    const goModPath = path.join(goDir, 'go.mod');
    fs.writeFileSync(goModPath, goModContent);
    log("\n Created go.mod file", "green");

    // Create example usage file
    const exampleContent = `package main

import (
    "fmt"
    "log"
    "math/big"
    
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    
    registry "./registry"
)

func main() {
    // Connect to Kaia node
    client, err := ethclient.Dial("https://public-en-kairos.node.kaia.io")
    if err != nil {
        log.Fatal(err)
    }

    // Contract address (replace with actual deployed address)
    contractAddress := common.HexToAddress("0x...")
    
    // Create contract instance
    instance, err := registry.NewSageRegistryV2(contractAddress, client)
    if err != nil {
        log.Fatal(err)
    }

    // Call contract method (example: get owner)
    owner, err := instance.Owner(&bind.CallOpts{})
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Contract owner: %s\\n", owner.Hex())
    
    // Get agent by DID
    did := "did:sage:example"
    agent, err := instance.GetAgentByDID(&bind.CallOpts{}, did)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Agent name: %s\\n", agent.Name)
    fmt.Printf("Agent active: %v\\n", agent.Active)
}
`;

    const examplePath = path.join(goDir, 'example.go');
    fs.writeFileSync(examplePath, exampleContent);
    log(" Created example.go", "green");

    // Create README for Go bindings
    const readmeContent = `# Go Bindings for SAGE Contracts

## Installation

\`\`\`bash
go get github.com/ethereum/go-ethereum
\`\`\`

## Usage

\`\`\`go
import (
    "github.com/ethereum/go-ethereum/ethclient"
    registry "./bindings/go/registry"
)

// Connect to network
client, err := ethclient.Dial("https://public-en-kairos.node.kaia.io")

// Load contract
contract, err := registry.NewSageRegistryV2(address, client)

// Read data
agent, err := contract.GetAgentByDID(&bind.CallOpts{}, "did:sage:example")

// Write data (requires private key)
auth := bind.NewKeyedTransactor(privateKey)
tx, err := contract.RegisterAgent(auth, did, name, desc, endpoint, pubKey, capabilities, signature)
\`\`\`

## Regenerate Bindings

\`\`\`bash
npm run generate:go
\`\`\`
`;

    const readmePath = path.join(goDir, 'README.md');
    fs.writeFileSync(readmePath, readmeContent);
    log(" Created README.md", "green");

    log("\n" + "=".repeat(50), "bright");
    log(" Go binding generation complete!", "green");
    log("\nGenerated files:", "yellow");
    log(`  📁 ${goDir}/`, "cyan");
    log("  📄 registry.go - SageRegistryV2 binding", "cyan");
    log("  📄 hook.go - SageVerificationHook binding", "cyan");
    log("  📄 example.go - Usage example", "cyan");
    log("  📄 go.mod - Go module file", "cyan");
    log("  📄 README.md - Documentation", "cyan");

  } catch (error) {
    log("\n Error generating Go bindings:", "red");
    console.error(error);
    process.exit(1);
  }
}

// Run generation
generateGoBindings();