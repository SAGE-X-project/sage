// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package httpsig

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sage-x-project/sage/core/obs"
)

// Request is an HTTP request as a signer or verifier sees it.
type Request struct {
	// Method is the request method as sent, for example "POST".
	Method string
	// Target is the absolute URI the signer sent the request to: scheme,
	// authority, path and query, as sent (03 §2, E03-22). A verifier behind
	// a proxy builds it from its configured external origin, never from an
	// internal Host.
	Target string
	// Header holds the request's field lines.
	Header http.Header
	// Body is the complete, buffered request content; a non-empty body
	// must be covered by content-digest (03 §3). A verifier passes the body
	// it read, never nil for a request whose body it has not read.
	Body []byte
}

// Response is an HTTP response as a signer or verifier sees it.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// ContentDigest returns the Content-Digest field value of body: one sha-256
// member (03 §1).
func ContentDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha-256=:" + base64.StdEncoding.EncodeToString(sum[:]) + ":"
}

// target is a parsed absolute target URI and the derived component values
// taken from it (RFC 9421 §2.2).
type target struct {
	uri, scheme, authority, query string
}

func parseTarget(s string) (target, error) {
	if !uriChars(s) {
		return target{}, canonicalize("target URI has a character outside RFC 3986 or a bad percent-encoding")
	}
	scheme, rest, ok := strings.Cut(s, "://")
	if !ok || (scheme != "http" && scheme != "https") {
		return target{}, canonicalize("target URI scheme is not lower-case http or https")
	}
	// The authority is taken as sent, up to the path or query, so that
	// @authority is the text inside @target-uri (03 §2).
	end := strings.IndexAny(rest, "/?")
	if end < 0 {
		end = len(rest)
	}
	authority := rest[:end]
	if strings.ContainsAny(authority, "@%") || strings.Contains(rest, "#") || !brackets(authority) ||
		strings.ContainsAny(rest[end:], "[]") {
		return target{}, canonicalize("target URI has user information, an encoded authority, a stray bracket or a fragment")
	}
	if (&url.URL{Host: authority}).Hostname() == "" {
		return target{}, canonicalize("target URI has no host")
	}
	if u, err := url.Parse(s); err != nil || u.Host != authority {
		return target{}, canonicalize("target URI is not an absolute URI")
	}
	_, query, _ := strings.Cut(rest[end:], "?")
	// @query is "?" and the query as sent; "?" for a request without one
	// (03 §2, RFC 9421 §2.2.7).
	return target{uri: s, scheme: scheme, authority: authority, query: "?" + query}, nil
}

// uriChars reports whether s is non-empty and holds only RFC 3986
// characters, with every "%" followed by two hex digits.
func uriChars(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '%':
			if i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2]) {
				return false
			}
			i += 2
		case isAlpha(c), isDigit(c), strings.IndexByte("-._~:/?#[]@!$&'()*+,;=", c) >= 0:
		default:
			return false
		}
	}
	return true
}

// brackets reports whether "[" and "]" occur in the authority only around
// an IP-literal host (RFC 3986 §3.2.2).
func brackets(authority string) bool {
	if !strings.HasPrefix(authority, "[") {
		return !strings.ContainsAny(authority, "[]")
	}
	end := strings.IndexByte(authority, ']')
	return end > 1 && !strings.ContainsAny(authority[1:end], "[") &&
		!strings.ContainsAny(authority[end+1:], "[]") &&
		(end+1 == len(authority) || authority[end+1] == ':')
}

func isHex(c byte) bool { return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') }

// normalAuthority reports whether the authority is the lower-case host,
// followed by ":port" only when the port is not the scheme's default (03 §2).
// A host with a trailing dot is a second spelling of the same host and is
// not normal.
func (t target) normalAuthority() bool {
	u := url.URL{Host: t.authority}
	if t.authority != strings.ToLower(t.authority) || strings.HasSuffix(t.authority, ":") ||
		strings.HasSuffix(u.Hostname(), ".") {
		return false
	}
	switch port := u.Port(); {
	case port == "":
		return true
	case t.scheme == "http" && port == "80", t.scheme == "https" && port == "443":
		return false
	default:
		n, err := strconv.Atoi(port)
		return err == nil && n > 0 && n < 65536 && strconv.Itoa(n) == port
	}
}

// fieldValue returns the combined value of the field name (lower case):
// each field line trimmed of surrounding whitespace, joined with ", "
// (03 §2, RFC 9421 §2.1). The keys of h are compared case-insensitively;
// a field stored under two keys is rejected because the order of its lines
// would be undefined.
func fieldValue(h http.Header, name string) (string, bool, error) {
	var lines []string
	found := 0
	for k, vs := range h {
		if strings.EqualFold(k, name) {
			found++
			lines = vs
		}
	}
	switch {
	case found == 0 || len(lines) == 0:
		return "", false, nil
	case found > 1:
		return "", false, canonicalize("covered field stored under two header keys")
	}
	trimmed := make([]string, len(lines))
	for i, l := range lines {
		trimmed[i] = strings.Trim(l, " \t")
	}
	v := strings.Join(trimmed, ", ")
	if unsafeValue(v) {
		return "", false, canonicalize("covered field value has a control or non-ASCII byte")
	}
	return v, true, nil
}

func hasField(h http.Header, name string) bool {
	for k, vs := range h {
		if strings.EqualFold(k, name) && len(vs) > 0 {
			return true
		}
	}
	return false
}

// unsafeValue reports a byte that cannot appear in the signature base:
// CR, LF, NUL and the other controls, which would add a line to it, and
// bytes above 0x7f, since the base is US-ASCII (RFC 9421 §2.5) and ;bs is
// not allowed (E03-09).
func unsafeValue(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c < 0x20 && c != '\t') || c >= 0x7f {
			return true
		}
	}
	return false
}

// validMethod reports whether m is an RFC 9110 token.
func validMethod(m string) bool {
	if m == "" {
		return false
	}
	for i := 0; i < len(m); i++ {
		if !isTchar(m[i]) {
			return false
		}
	}
	return true
}

// validFieldName reports whether n is a lower-case field name token, the
// form a covered field identifier takes (03 §2).
func validFieldName(n string) bool {
	return validMethod(n) && n == strings.ToLower(n)
}

func canonicalize(detail string) error { return reject(obs.ReasonCanonicalize, detail) }

func coverage(detail string) error { return reject(obs.ReasonCoverage, detail) }
