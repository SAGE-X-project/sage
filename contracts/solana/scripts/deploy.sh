#!/bin/bash

# SAGE Solana Contract Deployment Script

set -e

echo "Building Solana programs..."

# Build the registry program
cd programs/sage-registry
anchor build
cd ../..

# Build the verification hook program
cd programs/sage-verification-hook
anchor build
cd ../..

echo "Deploying programs..."

# Get the program IDs
REGISTRY_ID=$(solana address -k target/deploy/sage_registry-keypair.json)
HOOK_ID=$(solana address -k target/deploy/sage_verification_hook-keypair.json)

echo "Registry Program ID: $REGISTRY_ID"
echo "Hook Program ID: $HOOK_ID"

# Deploy the programs
anchor deploy

echo "Initializing registry..."

# Create initialization script
cat > scripts/initialize.ts << EOF
import * as anchor from "@project-serum/anchor";
import { Program } from "@project-serum/anchor";
import { SageRegistry } from "../target/types/sage_registry";
import { SageVerificationHook } from "../target/types/sage_verification_hook";

async function main() {
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);

  const registryProgram = anchor.workspace.SageRegistry as Program<SageRegistry>;
  const hookProgram = anchor.workspace.SageVerificationHook as Program<SageVerificationHook>;

  // Initialize registry
  const [registryPDA] = await anchor.web3.PublicKey.findProgramAddress(
    [Buffer.from("registry")],
    registryProgram.programId
  );

  try {
    await registryProgram.methods
      .initialize()
      .accounts({
        registry: registryPDA,
        authority: provider.wallet.publicKey,
        systemProgram: anchor.web3.SystemProgram.programId,
      })
      .rpc();
    console.log("Registry initialized");
  } catch (e) {
    console.log("Registry already initialized");
  }

  // Initialize hook
  const [hookStatePDA] = await anchor.web3.PublicKey.findProgramAddress(
    [Buffer.from("hook_state")],
    hookProgram.programId
  );

  try {
    await hookProgram.methods
      .initialize()
      .accounts({
        hookState: hookStatePDA,
        authority: provider.wallet.publicKey,
        systemProgram: anchor.web3.SystemProgram.programId,
      })
      .rpc();
    console.log("Hook initialized");
  } catch (e) {
    console.log("Hook already initialized");
  }

  // Set the hook in registry
  await registryProgram.methods
    .setBeforeRegisterHook(hookProgram.programId)
    .accounts({
      registry: registryPDA,
      authority: provider.wallet.publicKey,
    })
    .rpc();
  console.log("Hook set in registry");

  // Save deployment info
  const deploymentInfo = {
    network: provider.connection.rpcEndpoint,
    programs: {
      registry: registryProgram.programId.toString(),
      hook: hookProgram.programId.toString(),
    },
    accounts: {
      registry: registryPDA.toString(),
      hookState: hookStatePDA.toString(),
    },
    deployer: provider.wallet.publicKey.toString(),
    timestamp: new Date().toISOString(),
  };

  const fs = require("fs");
  fs.writeFileSync(
    "deployments/\${provider.connection.rpcEndpoint.includes('devnet') ? 'devnet' : 'mainnet'}.json",
    JSON.stringify(deploymentInfo, null, 2)
  );
}

main().catch(console.error);
EOF

# Run initialization
npx ts-node scripts/initialize.ts

echo "Deployment complete!"