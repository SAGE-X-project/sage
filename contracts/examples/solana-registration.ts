import * as anchor from "@project-serum/anchor";
import { Program } from "@project-serum/anchor";
import { SageRegistry } from "../solana/target/types/sage_registry";
import { Keypair, PublicKey } from "@solana/web3.js";
import * as ed from "@noble/ed25519";
import * as fs from "fs";

async function registerAgent() {
  // Setup provider
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);

  // Load the program
  const program = anchor.workspace.SageRegistry as Program<SageRegistry>;
  
  // Load deployment info
  const deploymentInfo = JSON.parse(
    fs.readFileSync("../solana/deployments/devnet.json", "utf8")
  );

  // Agent metadata
  const agentData = {
    did: "did:sage:sol-agent001",
    name: "SAGE Solana Assistant",
    description: "An AI agent for Solana program development",
    endpoint: "https://api.sage.ai/solana/agent/001",
    capabilities: JSON.stringify({
      models: ["gpt-4", "claude-3"],
      skills: ["anchor-development", "program-analysis"],
      chains: ["solana"],
    }),
  };

  // Create message for signature
  const message = `${agentData.did}:${agentData.name}:${agentData.description}:${agentData.endpoint}:${agentData.capabilities}:${provider.wallet.publicKey.toString()}`;
  const messageBytes = new TextEncoder().encode(message);

  // Sign with Ed25519
  const signature = await ed.sign(
    messageBytes,
    provider.wallet.payer.secretKey.slice(0, 32)
  );

  console.log("Registering agent...");
  console.log("DID:", agentData.did);
  console.log("Owner:", provider.wallet.publicKey.toString());

  try {
    // Derive PDAs
    const [registryPDA] = await PublicKey.findProgramAddress(
      [Buffer.from("registry")],
      program.programId
    );

    const [agentPDA] = await PublicKey.findProgramAddress(
      [Buffer.from("agent"), Buffer.from(agentData.did)],
      program.programId
    );

    // Register the agent
    const tx = await program.methods
      .registerAgent(
        agentData.did,
        agentData.name,
        agentData.description,
        agentData.endpoint,
        agentData.capabilities,
        Buffer.from(signature)
      )
      .accounts({
        agent: agentPDA,
        registry: registryPDA,
        owner: provider.wallet.publicKey,
        systemProgram: anchor.web3.SystemProgram.programId,
        ed25519Program: new PublicKey("Ed25519SigVerify111111111111111111111111111"),
      })
      .rpc();

    console.log("Transaction signature:", tx);

    // Wait for confirmation
    await provider.connection.confirmTransaction(tx, "confirmed");
    console.log("Transaction confirmed");

    // Fetch the registered agent
    const agent = await program.account.agent.fetch(agentPDA);
    console.log("\nRegistered agent:");
    console.log("- Name:", agent.name);
    console.log("- DID:", agent.did);
    console.log("- Active:", agent.active);
    console.log("- Owner:", agent.owner.toString());
    console.log("- Registered at:", new Date(agent.registeredAt.toNumber() * 1000));

  } catch (error) {
    console.error("Registration failed:", error);
    if (error.logs) {
      console.error("Program logs:", error.logs);
    }
  }
}

// Example: Update agent
async function updateAgent(agentDID: string) {
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);
  const program = anchor.workspace.SageRegistry as Program<SageRegistry>;

  const [agentPDA] = await PublicKey.findProgramAddress(
    [Buffer.from("agent"), Buffer.from(agentDID)],
    program.programId
  );

  const updatedData = {
    name: "SAGE Solana Assistant v2",
    description: "An upgraded AI agent for Solana development",
    endpoint: "https://api.sage.ai/solana/agent/001/v2",
    capabilities: JSON.stringify({
      models: ["gpt-4", "claude-3", "llama-3"],
      skills: ["anchor-development", "program-analysis", "security-audit"],
      chains: ["solana"],
      version: "2.0",
    }),
  };

  const tx = await program.methods
    .updateAgent(
      updatedData.name,
      updatedData.description,
      updatedData.endpoint,
      updatedData.capabilities
    )
    .accounts({
      agent: agentPDA,
      owner: provider.wallet.publicKey,
    })
    .rpc();

  console.log("Agent updated. Transaction:", tx);
}

// Example: Deactivate agent
async function deactivateAgent(agentDID: string) {
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);
  const program = anchor.workspace.SageRegistry as Program<SageRegistry>;

  const [agentPDA] = await PublicKey.findProgramAddress(
    [Buffer.from("agent"), Buffer.from(agentDID)],
    program.programId
  );

  const tx = await program.methods
    .deactivateAgent()
    .accounts({
      agent: agentPDA,
      owner: provider.wallet.publicKey,
    })
    .rpc();

  console.log("Agent deactivated. Transaction:", tx);
}

// Run registration
registerAgent().catch(console.error);