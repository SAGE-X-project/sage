package com.sage.client;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Decentralized Identifier (DID) parser and validator
 */
public class Did {
    private static final Pattern DID_PATTERN = Pattern.compile("^did:sage:([a-z]+):(.+)$");

    private final String didString;
    private final String network;
    private final String address;

    /**
     * Parse DID string
     *
     * @param didString DID string (e.g., "did:sage:ethereum:0x123...")
     * @throws SageException.DidException if DID format is invalid
     */
    public Did(String didString) throws SageException.DidException {
        if (didString == null || didString.isEmpty()) {
            throw new SageException.DidException("DID cannot be null or empty");
        }

        this.didString = didString;

        Matcher matcher = DID_PATTERN.matcher(didString);
        if (!matcher.matches()) {
            throw new SageException.DidException("Invalid DID format: " + didString);
        }

        this.network = matcher.group(1);
        this.address = matcher.group(2);

        validateNetwork();
        validateAddress();
    }

    /**
     * Create DID from parts
     */
    public static Did fromParts(String network, String address) throws SageException.DidException {
        String didString = String.format("did:sage:%s:%s", network, address);
        return new Did(didString);
    }

    /**
     * Validate network
     */
    private void validateNetwork() throws SageException.DidException {
        if (network == null || network.isEmpty()) {
            throw new SageException.DidException("Network cannot be empty");
        }

        // Common networks
        if (!network.matches("^(ethereum|polygon|solana|bitcoin|test)$")) {
            // Allow other networks but warn
        }
    }

    /**
     * Validate address
     */
    private void validateAddress() throws SageException.DidException {
        if (address == null || address.isEmpty()) {
            throw new SageException.DidException("Address cannot be empty");
        }

        // Validate Ethereum-style addresses
        if (network.equals("ethereum") || network.equals("polygon")) {
            if (!address.matches("^0x[a-fA-F0-9]{40}$")) {
                throw new SageException.DidException("Invalid Ethereum address: " + address);
            }
        }

        // Validate Solana addresses
        if (network.equals("solana")) {
            if (!address.matches("^[1-9A-HJ-NP-Za-km-z]{32,44}$")) {
                throw new SageException.DidException("Invalid Solana address: " + address);
            }
        }
    }

    /**
     * Get full DID string
     */
    @Override
    public String toString() {
        return didString;
    }

    /**
     * Get network
     */
    public String getNetwork() {
        return network;
    }

    /**
     * Get address
     */
    public String getAddress() {
        return address;
    }

    /**
     * Check if DID is valid
     */
    public static boolean isValid(String didString) {
        try {
            new Did(didString);
            return true;
        } catch (SageException.DidException e) {
            return false;
        }
    }

    @Override
    public boolean equals(Object obj) {
        if (this == obj) return true;
        if (obj == null || getClass() != obj.getClass()) return false;
        Did did = (Did) obj;
        return didString.equals(did.didString);
    }

    @Override
    public int hashCode() {
        return didString.hashCode();
    }
}
