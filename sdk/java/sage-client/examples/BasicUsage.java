import com.sage.client.*;
import com.sage.client.types.HealthStatus;

/**
 * Basic usage example for SAGE Java client
 */
public class BasicUsage {
    public static void main(String[] args) {
        try {
            System.out.println("SAGE Java Client - Basic Usage Example");
            System.out.println("======================================\n");

            // Initialize client
            ClientConfig config = ClientConfig.builder("http://localhost:8080")
                    .timeoutSeconds(30)
                    .maxSessions(100)
                    .build();

            SageClient client = new SageClient(config);

            // Register agent
            String clientDid = "did:sage:ethereum:0xAlice";
            client.registerAgent(clientDid, "Alice Agent");
            System.out.println("✓ Registered: " + clientDid);

            // Check server health
            HealthStatus health = client.healthCheck();
            System.out.println("✓ Server status: " + health.getStatus());

            // Get server DID
            String serverDid = client.getServerDid();
            System.out.println("✓ Server DID: " + serverDid);

            // Initiate handshake
            String sessionId = client.handshake(serverDid);
            System.out.println("✓ Session established: " + sessionId);

            // Send message
            byte[] message = "Hello, Server!".getBytes();
            byte[] response = client.sendMessage(sessionId, message);
            System.out.println("✓ Response: " + new String(response));

            // Send another message
            byte[] message2 = "How are you?".getBytes();
            byte[] response2 = client.sendMessage(sessionId, message2);
            System.out.println("✓ Response 2: " + new String(response2));

            // Show active sessions
            System.out.println("\n✓ Active sessions: " + client.activeSessions());

            System.out.println("\n✓ All operations completed successfully!");

        } catch (SageException e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }
}
