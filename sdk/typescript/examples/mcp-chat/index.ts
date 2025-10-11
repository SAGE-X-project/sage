/**
 * SAGE MCP Chat Example
 * Demonstrates secure agent-to-agent communication using SAGE
 */

import { SAGEClient } from '@sage-x/sdk';

async function main() {
  console.log('SAGE MCP Chat Example\n');

  // Initialize two agents
  console.log('1. Initializing agents...');
  const alice = new SAGEClient();
  const bob = new SAGEClient();

  await alice.initialize();
  await bob.initialize();

  console.log(`   Alice DID: ${alice.getDID()}`);
  console.log(`   Bob DID: ${bob.getDID()}\n`);

  // Alice initiates handshake with Bob
  console.log('2. Alice initiating handshake with Bob...');
  const aliceEphemeral = await alice.generateKeyPair('X25519');
  const bobEphemeral = await bob.generateKeyPair('X25519');

  const handshakeInit = await alice.initiateHandshake(bob.getPublicKey());
  console.log('   Handshake initiated\n');

  // Bob receives and responds
  console.log('3. Bob processing handshake...');
  // In real implementation, Bob would process the initiation
  // and send back a HandshakeResponse
  console.log('   Handshake processed\n');

  // Complete handshake (simulated)
  console.log('4. Completing handshake...');
  const sessionID = 'test-session-' + Date.now();
  console.log(`   Session ID: ${sessionID}\n`);

  // Alice sends encrypted message to Bob
  console.log('5. Alice sending encrypted message...');
  const message = 'Hello Bob! This is a secure message from Alice.';
  console.log(`   Plain text: "${message}"`);

  try {
    const encrypted = await alice.sendMessage(
      sessionID,
      new TextEncoder().encode(message)
    );
    console.log(`   Encrypted (${encrypted.ciphertext.length} bytes)\n`);

    // Bob receives and decrypts
    console.log('6. Bob receiving encrypted message...');
    const decrypted = await bob.receiveMessage(sessionID, encrypted);
    const receivedMessage = new TextDecoder().decode(decrypted);
    console.log(`   Decrypted: "${receivedMessage}"\n`);

    // Verify messages match
    if (message === receivedMessage) {
      console.log('✓ Success: Secure communication established!');
    } else {
      console.log('✗ Error: Messages do not match');
    }
  } catch (error) {
    console.error('Error during secure communication:', error);
  }

  // Display session info
  console.log('\n7. Session information:');
  const aliceSessions = alice.getAllSessions();
  const bobSessions = bob.getAllSessions();
  console.log(`   Alice active sessions: ${aliceSessions.length}`);
  console.log(`   Bob active sessions: ${bobSessions.length}`);

  console.log('\nExample complete!');
}

main().catch(console.error);
