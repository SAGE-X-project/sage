package com.sage.client;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for DID parser
 */
class DidTest {

    @Test
    void testValidEthereumDid() throws SageException.DidException {
        String didString = "did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC";
        Did did = new Did(didString);

        assertEquals(didString, did.toString());
        assertEquals("ethereum", did.getNetwork());
        assertEquals("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC", did.getAddress());
    }

    @Test
    void testValidSolanaDid() throws SageException.DidException {
        String didString = "did:sage:solana:DYw8jCTfwHNRJhhmFcbXvVDTqWMEVFBX6ZKUmG5CNSKK";
        Did did = new Did(didString);

        assertEquals(didString, did.toString());
        assertEquals("solana", did.getNetwork());
        assertEquals("DYw8jCTfwHNRJhhmFcbXvVDTqWMEVFBX6ZKUmG5CNSKK", did.getAddress());
    }

    @Test
    void testValidTestDid() throws SageException.DidException {
        String didString = "did:sage:test:alice123";
        Did did = new Did(didString);

        assertEquals(didString, did.toString());
        assertEquals("test", did.getNetwork());
        assertEquals("alice123", did.getAddress());
    }

    @Test
    void testFromParts() throws SageException.DidException {
        Did did = Did.fromParts("ethereum", "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC");

        assertEquals("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC", did.toString());
        assertEquals("ethereum", did.getNetwork());
    }

    @Test
    void testInvalidFormat() {
        assertThrows(SageException.DidException.class, () -> new Did("invalid-did"));
        assertThrows(SageException.DidException.class, () -> new Did("did:invalid"));
        assertThrows(SageException.DidException.class, () -> new Did("did:sage"));
    }

    @Test
    void testInvalidEthereumAddress() {
        assertThrows(SageException.DidException.class, () -> new Did("did:sage:ethereum:invalid"));
        assertThrows(SageException.DidException.class, () -> new Did("did:sage:ethereum:0x123"));
    }

    @Test
    void testNullOrEmpty() {
        assertThrows(SageException.DidException.class, () -> new Did(null));
        assertThrows(SageException.DidException.class, () -> new Did(""));
    }

    @Test
    void testIsValid() {
        assertTrue(Did.isValid("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC"));
        assertTrue(Did.isValid("did:sage:test:alice"));
        assertFalse(Did.isValid("invalid-did"));
        assertFalse(Did.isValid("did:sage:ethereum:invalid"));
    }

    @Test
    void testEquality() throws SageException.DidException {
        Did did1 = new Did("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC");
        Did did2 = new Did("did:sage:ethereum:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEbC");
        Did did3 = new Did("did:sage:ethereum:0x123456789abcdef0123456789abcdef012345678");

        assertEquals(did1, did2);
        assertNotEquals(did1, did3);
        assertEquals(did1.hashCode(), did2.hashCode());
    }
}
