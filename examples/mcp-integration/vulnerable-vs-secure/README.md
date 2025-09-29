# Vulnerable vs Secure AI Chat Example

This example demonstrates the security difference between a vulnerable AI chat system and one protected by SAGE.

## The Vulnerability

Without SAGE, AI chat systems are vulnerable to:
- **Identity spoofing** - Anyone can pretend to be any agent
- **Message tampering** - Attackers can modify messages in transit
- **Replay attacks** - Old messages can be resent
- **Unauthorized access** - No verification of agent capabilities

## Running the Demo

### 1. Start the Vulnerable Chat Server
```bash
cd vulnerable-chat
go run .
# Server starts on :8082
```

### 2. Try the Attack
```bash
cd attacker
go run .
# Shows how easy it is to exploit the vulnerable server
```

### 3. Start the Secure Chat Server
```bash
cd secure-chat
go run .
# Server starts on :8083
```

### 4. Try the Attack Again
```bash
cd attacker
go run . --secure
# Attack fails! SAGE protects the server
```

## What You'll See

### Vulnerable Server Output:
```
🚨 VULNERABLE Chat Server (NO SECURITY)
📍 Listening on http://localhost:8082

  Received message from: evil-hacker-bot
💬 Message: DELETE * FROM users; --
 Processed successfully (THIS IS BAD!)
```

### Secure Server Output:
```
 SECURE Chat Server (SAGE PROTECTED)
📍 Listening on http://localhost:8083

 Request rejected: SAGE verification failed: missing X-Agent-DID header
  Attack blocked!
```

## The Code Difference

### Vulnerable (3 lines):
```go
func handleChat(w http.ResponseWriter, r *http.Request) {
    // Process request directly - NO SECURITY!
    processMessage(r)
}
```

### Secure (6 lines):
```go
func handleChat(w http.ResponseWriter, r *http.Request) {
    // Add SAGE verification
    if err := sage.VerifyRequest(r); err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }
    processMessage(r)
}
```

That's it! Just 3 extra lines protect your entire AI system.