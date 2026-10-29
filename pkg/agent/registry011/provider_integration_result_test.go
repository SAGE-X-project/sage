package registry011

import (
	"crypto/tls"
	"strconv"
	"testing"
)

func TestProviderIntegrationCipherKeepsExactTLSVersionAndCipherWidth(t *testing.T) {
	for _, version := range []uint16{tls.VersionTLS12, tls.VersionTLS13} {
		for _, cipher := range []uint16{tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384, tls.TLS_CHACHA20_POLY1305_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256} {
			m := map[string]any{"tls_version": strconv.FormatUint(uint64(version), 10), "cipher_suite": strconv.FormatUint(uint64(cipher), 10)}
			expected := (version == tls.VersionTLS13) == (cipher == tls.TLS_AES_128_GCM_SHA256 || cipher == tls.TLS_AES_256_GCM_SHA384 || cipher == tls.TLS_CHACHA20_POLY1305_SHA256)
			if providerIntegrationCipher(m) != expected {
				t.Fatal("TLS version/cipher suite boundary changed", version, cipher)
			}
			m["cipher_suite"] = strconv.FormatUint(uint64(cipher)+65536, 10)
			if providerIntegrationCipher(m) {
				t.Fatal("oversized inert integer wrapped into an allowed cipher")
			}
		}
	}
	for _, m := range []map[string]any{nil, {}, {"tls_version": "771", "cipher_suite": "0"}, {"tls_version": "770", "cipher_suite": "4865"}, {"tls_version": "772", "cipher_suite": "65536"}, {"tls_version": "772", "cipher_suite": "18446744073709551615"}, {"tls_version": "772", "cipher_suite": "invalid"}, {"tls_version": "772", "cipher_suite": "1"}} {
		if providerIntegrationCipher(m) {
			t.Fatal("unsupported or malformed TLS evidence")
		}
	}
}
