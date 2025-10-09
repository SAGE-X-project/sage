package com.sage.client;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.sage.client.types.*;
import okhttp3.*;

import java.io.IOException;
import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

/**
 * SAGE client for secure agent communication
 */
public class SageClient {
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    private final ClientConfig config;
    private final OkHttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final SessionManager sessionManager;

    private KeyPair identityKeyPair;
    private KeyPair kemKeyPair;
    private String clientDid;

    /**
     * Create new SAGE client
     */
    public SageClient(ClientConfig config) throws SageException {
        this.config = config;
        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(config.getTimeoutSeconds(), TimeUnit.SECONDS)
                .readTimeout(config.getTimeoutSeconds(), TimeUnit.SECONDS)
                .writeTimeout(config.getTimeoutSeconds(), TimeUnit.SECONDS)
                .build();
        this.objectMapper = new ObjectMapper();
        this.objectMapper.registerModule(new JavaTimeModule());
        this.sessionManager = new SessionManager(config.getMaxSessions());

        initialize();
    }

    /**
     * Initialize client with keypairs
     */
    private void initialize() throws SageException {
        this.identityKeyPair = Crypto.generateEd25519KeyPair();
        this.kemKeyPair = Crypto.generateX25519KeyPair();
    }

    /**
     * Get server's KEM public key
     */
    public byte[] getServerKemKey() throws SageException {
        try {
            String url = config.getBaseUrl() + "/debug/kem-pub";
            Request request = new Request.Builder()
                    .url(url)
                    .get()
                    .build();

            try (Response response = httpClient.newCall(request).execute()) {
                if (!response.isSuccessful()) {
                    throw new SageException.NetworkException("Failed to get server KEM key: " + response.code());
                }

                String body = response.body().string();
                KemPublicKeyResponse kemResponse = objectMapper.readValue(body, KemPublicKeyResponse.class);
                return Crypto.base64Decode(kemResponse.getKemPublicKey());
            }
        } catch (IOException e) {
            throw new SageException.NetworkException("Network error", e);
        }
    }

    /**
     * Get server's DID
     */
    public String getServerDid() throws SageException {
        try {
            String url = config.getBaseUrl() + "/debug/server-did";
            Request request = new Request.Builder()
                    .url(url)
                    .get()
                    .build();

            try (Response response = httpClient.newCall(request).execute()) {
                if (!response.isSuccessful()) {
                    throw new SageException.NetworkException("Failed to get server DID: " + response.code());
                }

                String body = response.body().string();
                ServerDidResponse didResponse = objectMapper.readValue(body, ServerDidResponse.class);
                return didResponse.getDid();
            }
        } catch (IOException e) {
            throw new SageException.NetworkException("Network error", e);
        }
    }

    /**
     * Health check
     */
    public HealthStatus healthCheck() throws SageException {
        try {
            String url = config.getBaseUrl() + "/debug/health";
            Request request = new Request.Builder()
                    .url(url)
                    .get()
                    .build();

            try (Response response = httpClient.newCall(request).execute()) {
                if (!response.isSuccessful()) {
                    throw new SageException.NetworkException("Health check failed: " + response.code());
                }

                String body = response.body().string();
                return objectMapper.readValue(body, HealthStatus.class);
            }
        } catch (IOException e) {
            throw new SageException.NetworkException("Network error", e);
        }
    }

    /**
     * Register agent (development only)
     */
    public void registerAgent(String did, String name) throws SageException {
        if (identityKeyPair == null || kemKeyPair == null) {
            throw new SageException.NotInitializedException("Client not initialized");
        }

        try {
            AgentMetadata metadata = new AgentMetadata(
                    did,
                    name,
                    true,
                    Crypto.base64Encode(identityKeyPair.getPublicKey()),
                    Crypto.base64Encode(kemKeyPair.getPublicKey())
            );

            String url = config.getBaseUrl() + "/debug/register-agent";
            String json = objectMapper.writeValueAsString(metadata);
            RequestBody body = RequestBody.create(json, JSON);

            Request request = new Request.Builder()
                    .url(url)
                    .post(body)
                    .build();

            try (Response response = httpClient.newCall(request).execute()) {
                if (!response.isSuccessful()) {
                    throw new SageException.NetworkException("Failed to register agent: " + response.code());
                }
            }

            this.clientDid = did;
        } catch (IOException e) {
            throw new SageException.NetworkException("Network error", e);
        }
    }

    /**
     * Initiate HPKE handshake
     */
    public String handshake(String serverDid) throws SageException {
        if (clientDid == null) {
            throw new SageException.NotInitializedException("Client not registered");
        }
        if (identityKeyPair == null) {
            throw new SageException.NotInitializedException("Identity keypair not initialized");
        }

        try {
            // Get server KEM key
            byte[] serverKemKey = getServerKemKey();

            // Setup HPKE sender
            Crypto.HpkeSetupResult hpkeSetup = Crypto.setupHpkeSender(serverKemKey);
            Crypto.HpkeContext hpkeContext = hpkeSetup.getContext();
            byte[] encapsulatedKey = hpkeSetup.getEncapsulatedKey();

            // Create handshake data
            Map<String, Object> handshakeData = new HashMap<>();
            handshakeData.put("type", "handshake");
            handshakeData.put("client_did", clientDid);
            handshakeData.put("timestamp", Instant.now().getEpochSecond());

            byte[] plaintext = objectMapper.writeValueAsBytes(handshakeData);
            byte[] ciphertext = hpkeContext.seal(plaintext);

            // Combine encapsulated key and ciphertext
            byte[] message = new byte[encapsulatedKey.length + ciphertext.length];
            System.arraycopy(encapsulatedKey, 0, message, 0, encapsulatedKey.length);
            System.arraycopy(ciphertext, 0, message, encapsulatedKey.length, ciphertext.length);

            String messageB64 = Crypto.base64Encode(message);

            // Sign the request
            long timestamp = Instant.now().getEpochSecond();
            String toSign = String.format("%s|%s|%s|%d", clientDid, serverDid, messageB64, timestamp);
            byte[] signature = Crypto.sign(toSign.getBytes(), identityKeyPair.getPrivateKey());
            String signatureB64 = Crypto.base64Encode(signature);

            // Create request
            HandshakeRequest handshakeRequest = new HandshakeRequest(
                    clientDid,
                    serverDid,
                    messageB64,
                    timestamp,
                    signatureB64
            );

            // Send request
            String url = config.getBaseUrl() + "/v1/a2a:sendMessage";
            String json = objectMapper.writeValueAsString(handshakeRequest);
            RequestBody requestBody = RequestBody.create(json, JSON);

            Request httpRequest = new Request.Builder()
                    .url(url)
                    .post(requestBody)
                    .build();

            try (Response response = httpClient.newCall(httpRequest).execute()) {
                if (!response.isSuccessful()) {
                    throw new SageException.NetworkException("Handshake failed: " + response.code());
                }

                String responseBody = response.body().string();
                HandshakeResponse handshakeResponse = objectMapper.readValue(responseBody, HandshakeResponse.class);

                // Create session
                String sessionId = handshakeResponse.getSessionId();
                Session session = new Session(sessionId, clientDid, serverDid, hpkeContext, 3600);
                sessionManager.addSession(session);

                return sessionId;
            }
        } catch (IOException e) {
            throw new SageException.NetworkException("Network error", e);
        }
    }

    /**
     * Send encrypted message
     */
    public byte[] sendMessage(String sessionId, byte[] message) throws SageException {
        Session session = sessionManager.getSession(sessionId);
        if (session == null) {
            throw new SageException.SessionException("Session not found: " + sessionId);
        }

        if (clientDid == null) {
            throw new SageException.NotInitializedException("Client not registered");
        }
        if (identityKeyPair == null) {
            throw new SageException.NotInitializedException("Identity keypair not initialized");
        }

        try {
            // Encrypt message
            byte[] ciphertext = session.encrypt(message);
            String messageB64 = Crypto.base64Encode(ciphertext);

            // Sign the request
            long timestamp = Instant.now().getEpochSecond();
            String toSign = String.format("%s|%s|%s|%d", clientDid, session.getServerDid(), messageB64, timestamp);
            byte[] signature = Crypto.sign(toSign.getBytes(), identityKeyPair.getPrivateKey());
            String signatureB64 = Crypto.base64Encode(signature);

            // Create request
            MessageRequest messageRequest = new MessageRequest(
                    clientDid,
                    session.getServerDid(),
                    messageB64,
                    timestamp,
                    signatureB64
            );

            // Send request
            String url = config.getBaseUrl() + "/v1/a2a:sendMessage";
            String json = objectMapper.writeValueAsString(messageRequest);
            RequestBody requestBody = RequestBody.create(json, JSON);

            Request httpRequest = new Request.Builder()
                    .url(url)
                    .post(requestBody)
                    .header("X-Session-ID", sessionId)
                    .build();

            try (Response response = httpClient.newCall(httpRequest).execute()) {
                if (!response.isSuccessful()) {
                    throw new SageException.NetworkException("Send message failed: " + response.code());
                }

                String responseBody = response.body().string();
                MessageResponse messageResponse = objectMapper.readValue(responseBody, MessageResponse.class);

                // Decrypt response
                byte[] responseBytes = Crypto.base64Decode(messageResponse.getResponse());
                return session.decrypt(responseBytes);
            }
        } catch (IOException e) {
            throw new SageException.NetworkException("Network error", e);
        }
    }

    /**
     * Get active session count
     */
    public int activeSessions() {
        return sessionManager.count();
    }

    /**
     * Get client DID
     */
    public String getClientDid() {
        return clientDid;
    }
}
