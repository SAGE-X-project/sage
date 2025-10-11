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

package rfc9421

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Canonicalizer builds signature base strings according to RFC 9421
type Canonicalizer struct{}

// NewCanonicalizer creates a new canonicalizer
func NewCanonicalizer() *Canonicalizer {
	return &Canonicalizer{}
}

// BuildSignatureBase creates the signature base string for the given request and components
func (c *Canonicalizer) BuildSignatureBase(req *http.Request, sigName string, params *SignatureInputParams) (string, error) {
	var lines []string

	// Process each covered component
	for _, component := range params.CoveredComponents {
		line, err := c.canonicalizeComponent(req, component)
		if err != nil {
			return "", err
		}
		lines = append(lines, line)
	}

	// Add signature parameters as the last line
	sigParams := c.buildSignatureParams(sigName, params)
	lines = append(lines, sigParams)

	return strings.Join(lines, "\n"), nil
}

// canonicalizeComponent processes a single component
func (c *Canonicalizer) canonicalizeComponent(req *http.Request, component string) (string, error) {
	component = strings.TrimSpace(component)

	// Check for @query-param
	if strings.Contains(component, "@query-param") {
		return c.canonicalizeQueryParam(req, component)
	}

	// Remove quotes if present for lookup
	lookupComponent := strings.Trim(component, `"`)

	// Handle HTTP signature components
	if strings.HasPrefix(lookupComponent, "@") {
		return c.canonicalizeHTTPComponent(req, lookupComponent)
	}

	// Handle regular headers
	return c.canonicalizeHeader(req, lookupComponent)
}

// canonicalizeHTTPComponent handles @-prefixed components
func (c *Canonicalizer) canonicalizeHTTPComponent(req *http.Request, component string) (string, error) {
	var value string

	switch component {
	case "@method":
		value = req.Method

	case "@target-uri":
		// Reconstruct full URI
		scheme := req.URL.Scheme
		if scheme == "" {
			if req.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		host := req.Host
		if host == "" {
			host = req.URL.Host
		}
		value = fmt.Sprintf("%s://%s%s", scheme, host, req.URL.RequestURI())

	case "@authority":
		value = req.Host
		if value == "" {
			value = req.URL.Host
		}

	case "@scheme":
		value = req.URL.Scheme
		if value == "" {
			if req.TLS != nil {
				value = "https"
			} else {
				value = "http"
			}
		}

	case "@request-target":
		// Method + space + request-target
		target := req.URL.Path
		if target == "" {
			target = "/"
		}
		if req.URL.RawQuery != "" {
			target += "?" + req.URL.RawQuery
		}
		value = fmt.Sprintf("%s %s", req.Method, target)

	case "@path":
		value = req.URL.Path
		if value == "" {
			value = "/"
		}

	case "@query":
		if req.URL.RawQuery != "" {
			value = "?" + req.URL.RawQuery
		} else {
			value = "?"
		}

	case "@status":
		// Status is only available for responses
		return "", fmt.Errorf("component not found: @status (only available for responses)")

	default:
		return "", fmt.Errorf("unknown HTTP component: %s", component)
	}

	return fmt.Sprintf(`"%s": %s`, component, value), nil
}

// canonicalizeHeader handles regular HTTP headers
func (c *Canonicalizer) canonicalizeHeader(req *http.Request, headerName string) (string, error) {
	// Headers are case-insensitive
	values := req.Header[http.CanonicalHeaderKey(headerName)]
	if len(values) == 0 {
		return "", fmt.Errorf("component not found: header %s", headerName)
	}

	// Join multiple values with comma and space
	value := strings.Join(values, ", ")

	// Trim leading and trailing whitespace
	value = strings.TrimSpace(value)

	// Format as lowercase header name
	return fmt.Sprintf(`"%s": %s`, strings.ToLower(headerName), value), nil
}

// canonicalizeQueryParam handles @query-param components
func (c *Canonicalizer) canonicalizeQueryParam(req *http.Request, component string) (string, error) {
	// Parse the parameter name
	paramName, err := parseQueryParam(component)
	if err != nil {
		return "", fmt.Errorf("invalid @query-param component: %w", err)
	}

	// Get query parameters
	query := req.URL.Query()
	values, exists := query[paramName]
	if !exists || len(values) == 0 {
		return "", fmt.Errorf("component not found: query parameter %s", paramName)
	}

	// Use the first value if multiple exist
	value := values[0]

	// The component identifier includes the parameter
	return fmt.Sprintf(`%s: %s`, component, value), nil
}

// buildSignatureParams creates the @signature-params line
func (c *Canonicalizer) buildSignatureParams(sigName string, params *SignatureInputParams) string {
	var parts []string

	// Add covered components
	components := make([]string, len(params.CoveredComponents))
	for i, comp := range params.CoveredComponents {
		// Don't re-quote components that already have proper formatting
		// e.g., "@query-param";name="id" should stay as is
		components[i] = comp
	}
	parts = append(parts, "("+strings.Join(components, " ")+")")

	// Add parameters
	if params.KeyID != "" {
		parts = append(parts, fmt.Sprintf(`keyid="%s"`, params.KeyID))
	}
	if params.Algorithm != "" {
		parts = append(parts, fmt.Sprintf(`alg="%s"`, params.Algorithm))
	}
	if params.Created > 0 {
		parts = append(parts, fmt.Sprintf(`created=%d`, params.Created))
	}
	if params.Expires > 0 {
		parts = append(parts, fmt.Sprintf(`expires=%d`, params.Expires))
	}
	if params.Nonce != "" {
		parts = append(parts, fmt.Sprintf(`nonce="%s"`, params.Nonce))
	}

	return fmt.Sprintf(`"@signature-params": %s`, strings.Join(parts, ";"))
}

// Helper function to parse query parameters from URL
func parseQueryString(rawQuery string) url.Values {
	values := make(url.Values)
	if rawQuery == "" {
		return values
	}

	for rawQuery != "" {
		var key, value string
		// Find the next key-value pair
		idx := strings.IndexAny(rawQuery, "&")
		var pair string
		if idx >= 0 {
			pair = rawQuery[:idx]
			rawQuery = rawQuery[idx+1:]
		} else {
			pair = rawQuery
			rawQuery = ""
		}

		// Split key and value
		if eqIdx := strings.Index(pair, "="); eqIdx >= 0 {
			key = pair[:eqIdx]
			value = pair[eqIdx+1:]
		} else {
			key = pair
		}

		// Add to values without decoding
		values.Add(key, value)
	}

	return values
}
