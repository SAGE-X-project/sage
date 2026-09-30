package registry010

import "strings"

// WebRegistryRequestURL010 constructs the sole REG-08 read target from a web
// DID and a locally configured origin allowlist. The caller must still use a
// trusted TLS transport, check the connected destination, and disable caches
// and redirects; a URL alone is never an authoritative observation.
func WebRegistryRequestURL010(did string, allowedOrigins []string) (string, error) {
	if !validWebDID010(did) {
		return "", ErrInvalidRecord010
	}
	rest := strings.TrimPrefix(did, "did:sage:web:")
	domain, agent, _ := strings.Cut(rest, ":")
	origin := "https://" + domain
	allowed := false
	for _, candidate := range allowedOrigins {
		if candidate == origin {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", ErrUnreachable
	}
	return origin + "/.well-known/sage/agents/" + agent, nil
}

// CheckWebRegistryResponsePolicy010 checks the status, media, and origin cache
// directive before a body is accepted. The transport must retain separate
// field lines and trailers, avoid intermediary caches, and reject redirects.
// This check is not proof of TLS origin, connected destination, or authority.
func CheckWebRegistryResponsePolicy010(status int, header, trailer []HeaderField010) error {
	if status != 200 {
		return ErrUnreachable
	}
	if err := CheckWebRegistryMedia010(header, trailer); err != nil {
		return err
	}
	for _, field := range header {
		if !asciiEqualFold010(field.Name, "cache-control") {
			continue
		}
		for _, directive := range strings.Split(field.Value, ",") {
			if asciiEqualFold010(strings.Trim(directive, " \t"), "no-store") {
				return nil
			}
		}
	}
	return ErrInvalidRecord010
}
