/**
 * SAGE Cryptographic Operations
 */

import * as ed25519 from '@noble/ed25519';
import * as secp256k1 from '@noble/secp256k1';
import { x25519 } from '@noble/curves/ed25519';
import { sha256 } from '@noble/hashes/sha256';
import { hkdf } from '@noble/hashes/hkdf';
import type { KeyPair, KeyType, CryptoProvider } from './types';

export class SAGECrypto implements CryptoProvider {
  /**
   * Generate a new key pair
   */
  async generateKeyPair(type: KeyType): Promise<KeyPair> {
    switch (type) {
      case 'Ed25519':
        return this.generateEd25519KeyPair();
      case 'Secp256k1':
        return this.generateSecp256k1KeyPair();
      case 'X25519':
        return this.generateX25519KeyPair();
      default:
        throw new Error(`Unsupported key type: ${type}`);
    }
  }

  /**
   * Sign a message
   */
  async sign(message: Uint8Array, privateKey: Uint8Array, type: KeyType): Promise<Uint8Array> {
    switch (type) {
      case 'Ed25519':
        return ed25519.sign(message, privateKey);
      case 'Secp256k1':
        const msgHash = sha256(message);
        return secp256k1.sign(msgHash, privateKey).toCompactRawBytes();
      default:
        throw new Error(`Signing not supported for key type: ${type}`);
    }
  }

  /**
   * Verify a signature
   */
  async verify(
    message: Uint8Array,
    signature: Uint8Array,
    publicKey: Uint8Array,
    type: KeyType
  ): Promise<boolean> {
    try {
      switch (type) {
        case 'Ed25519':
          return ed25519.verify(signature, message, publicKey);
        case 'Secp256k1':
          const msgHash = sha256(message);
          return secp256k1.verify(signature, msgHash, publicKey);
        default:
          throw new Error(`Verification not supported for key type: ${type}`);
      }
    } catch {
      return false;
    }
  }

  /**
   * Derive session keys using X25519 key agreement and HKDF
   */
  async deriveSessionKeys(
    myPrivateKey: Uint8Array,
    peerPublicKey: Uint8Array
  ): Promise<{ encryptKey: Uint8Array; decryptKey: Uint8Array }> {
    // X25519 key agreement
    const sharedSecret = x25519.getSharedSecret(myPrivateKey, peerPublicKey);

    // Derive session ID using HKDF
    const sessionID = hkdf(sha256, sharedSecret, new Uint8Array(), 'SAGE-Session-v1', 32);

    // Derive directional keys
    const encryptKey = hkdf(sha256, sessionID, new Uint8Array(), 'client-to-server', 32);
    const decryptKey = hkdf(sha256, sessionID, new Uint8Array(), 'server-to-client', 32);

    return { encryptKey, decryptKey };
  }

  /**
   * Encrypt data using AES-GCM (via WebCrypto API)
   */
  async encrypt(plaintext: Uint8Array, key: Uint8Array): Promise<{
    ciphertext: Uint8Array;
    nonce: Uint8Array;
    tag: Uint8Array;
  }> {
    // Generate random nonce
    const nonce = crypto.getRandomValues(new Uint8Array(12));

    // Import key
    const cryptoKey = await crypto.subtle.importKey(
      'raw',
      key,
      { name: 'AES-GCM', length: 256 },
      false,
      ['encrypt']
    );

    // Encrypt
    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv: nonce, tagLength: 128 },
      cryptoKey,
      plaintext
    );

    const result = new Uint8Array(encrypted);
    const ciphertext = result.slice(0, -16);
    const tag = result.slice(-16);

    return { ciphertext, nonce, tag };
  }

  /**
   * Decrypt data using AES-GCM (via WebCrypto API)
   */
  async decrypt(
    ciphertext: Uint8Array,
    nonce: Uint8Array,
    tag: Uint8Array,
    key: Uint8Array
  ): Promise<Uint8Array> {
    // Import key
    const cryptoKey = await crypto.subtle.importKey(
      'raw',
      key,
      { name: 'AES-GCM', length: 256 },
      false,
      ['decrypt']
    );

    // Combine ciphertext and tag
    const combined = new Uint8Array(ciphertext.length + tag.length);
    combined.set(ciphertext, 0);
    combined.set(tag, ciphertext.length);

    // Decrypt
    const decrypted = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: nonce, tagLength: 128 },
      cryptoKey,
      combined
    );

    return new Uint8Array(decrypted);
  }

  // Private helper methods

  private async generateEd25519KeyPair(): Promise<KeyPair> {
    const privateKey = ed25519.utils.randomPrivateKey();
    const publicKey = await ed25519.getPublicKey(privateKey);

    return {
      publicKey,
      privateKey,
      type: 'Ed25519',
    };
  }

  private async generateSecp256k1KeyPair(): Promise<KeyPair> {
    const privateKey = secp256k1.utils.randomPrivateKey();
    const publicKey = secp256k1.getPublicKey(privateKey, true);

    return {
      publicKey,
      privateKey,
      type: 'Secp256k1',
    };
  }

  private generateX25519KeyPair(): KeyPair {
    const privateKey = x25519.utils.randomPrivateKey();
    const publicKey = x25519.getPublicKey(privateKey);

    return {
      publicKey,
      privateKey,
      type: 'X25519',
    };
  }
}

/**
 * Utility functions
 */
export const utils = {
  /**
   * Convert hex string to Uint8Array
   */
  hexToBytes(hex: string): Uint8Array {
    if (hex.startsWith('0x')) {
      hex = hex.slice(2);
    }
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < hex.length; i += 2) {
      bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
    }
    return bytes;
  },

  /**
   * Convert Uint8Array to hex string
   */
  bytesToHex(bytes: Uint8Array): string {
    return '0x' + Array.from(bytes)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  },

  /**
   * Convert base64 to Uint8Array
   */
  base64ToBytes(base64: string): Uint8Array {
    const binaryString = atob(base64);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    return bytes;
  },

  /**
   * Convert Uint8Array to base64
   */
  bytesToBase64(bytes: Uint8Array): string {
    const binaryString = Array.from(bytes)
      .map(b => String.fromCharCode(b))
      .join('');
    return btoa(binaryString);
  },

  /**
   * Generate random bytes
   */
  randomBytes(length: number): Uint8Array {
    return crypto.getRandomValues(new Uint8Array(length));
  },
};
