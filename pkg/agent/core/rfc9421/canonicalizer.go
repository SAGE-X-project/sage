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
	"strconv"
	"strings"
)

// Canonicalizer builds signature base strings according to RFC 9421
type Canonicalizer struct{}

// NewCanonicalizer creates a new canonicalizer
func NewCanonicalizer() *Canonicalizer {
	return &Canonicalizer{}
}

// httpMessage is the message a signature base is built over: a request, or a
// response together with the request it answers. For a response, components
// carrying the ";req" parameter (RFC 9421 Section 2.4) are taken from the
// request, which binds the response to that exact request.
type httpMessage struct {
	req  *http.Request
	resp *http.Response
}

// BuildSignatureBase creates the signature base string for the given request and components
func (c *Canonicalizer) BuildSignatureBase(req *http.Request, sigName string, params *SignatureInputParams) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}
	return c.buildSignatureBase(httpMessage{req: req}, sigName, params)
}

// BuildResponseSignatureBase creates the signature base string for a response.
// req is the request the response answers and may be nil when no covered
// component carries the ";req" parameter.
func (c *Canonicalizer) BuildResponseSignatureBase(resp *http.Response, req *http.Request, sigName string, params *SignatureInputParams) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}
	if req == nil {
		req = resp.Request
	}
	return c.buildSignatureBase(httpMessage{req: req, resp: resp}, sigName, params)
}

func (c *Canonicalizer) buildSignatureBase(m httpMessage, sigName string, params *SignatureInputParams) (string, error) {
	var lines []string

	// Process each covered component
	for _, component := range params.CoveredComponents {
		line, err := c.canonicalizeComponent(m, component)
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

// splitComponentIdentifier separates a component identifier such as
// `"@query-param";name="id"` or `"@method";req` into the unquoted component
// name and the raw parameter suffix (including the leading ';', or "").
func splitComponentIdentifier(component string) (name, paramStr string) {
	component = strings.TrimSpace(component)
	if strings.HasPrefix(component, `"`) {
		if end := strings.Index(component[1:], `"`); end >= 0 {
			return component[1 : end+1], component[end+2:]
		}
		return strings.Trim(component, `"`), ""
	}
	if i := strings.Index(component, ";"); i >= 0 {
		return component[:i], component[i:]
	}
	return component, ""
}

// hasComponentParam reports whether the raw parameter suffix carries the
// given flag parameter (for example "req").
func hasComponentParam(paramStr, flag string) bool {
	for _, p := range strings.Split(paramStr, ";") {
		if strings.TrimSpace(p) == flag {
			return true
		}
	}
	return false
}

// normalizeComponentIdentifier returns the canonical spelling of a component
// identifier: the lowercased name in quotes followed by its parameters with
// surrounding whitespace removed. Used to compare identifiers.
func normalizeComponentIdentifier(component string) string {
	name, paramStr := splitComponentIdentifier(component)
	out := `"` + strings.ToLower(strings.TrimSpace(name)) + `"`
	for _, p := range strings.Split(paramStr, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out += ";" + p
		}
	}
	return out
}

// canonicalizeComponent processes a single component
func (c *Canonicalizer) canonicalizeComponent(m httpMessage, component string) (string, error) {
	component = strings.TrimSpace(component)
	name, paramStr := splitComponentIdentifier(component)
	identifier := `"` + strings.ToLower(name) + `"` + paramStr
	fromRequest := hasComponentParam(paramStr, "req")

	// Decide which message the value comes from.
	var req *http.Request
	var resp *http.Response
	switch {
	case m.resp == nil:
		if fromRequest {
			return "", fmt.Errorf("component %s: the req parameter is only valid in response signatures", component)
		}
		req = m.req
	case fromRequest:
		if m.req == nil {
			return "", fmt.Errorf("component %s: response has no associated request", component)
		}
		req = m.req
	default:
		resp = m.resp
	}

	// @query-param carries its own parameter; keep the identifier exactly as given.
	if name == "@query-param" {
		if req == nil {
			return "", fmt.Errorf("component not found: @query-param (only available from the request)")
		}
		value, err := c.queryParamValue(req, component)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`%s: %s`, component, value), nil
	}

	var value string
	var err error
	switch {
	case strings.HasPrefix(name, "@") && resp != nil:
		value, err = responseComponentValue(resp, name)
	case strings.HasPrefix(name, "@"):
		value, err = requestComponentValue(req, name)
	case resp != nil:
		value, err = headerValue(resp.Header, name)
	default:
		value, err = headerValue(req.Header, name)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`%s: %s`, identifier, value), nil
}

// responseComponentValue returns the value of a derived component of a response.
func responseComponentValue(resp *http.Response, component string) (string, error) {
	switch component {
	case "@status":
		return strconv.Itoa(resp.StatusCode), nil
	default:
		return "", fmt.Errorf("component not found: %s (only available from the request; add the req parameter)", component)
	}
}

// requestComponentValue returns the value of a derived component of a request.
func requestComponentValue(req *http.Request, component string) (string, error) {
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
		// RFC 9421 section 2.2.5: the request target as it appears in the
		// request line, without the method (path and query for origin form).
		target := req.URL.Path
		if target == "" {
			target = "/"
		}
		if req.URL.RawQuery != "" {
			target += "?" + req.URL.RawQuery
		}
		value = target

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

	return value, nil
}

// headerValue returns the canonical value of a header field.
func headerValue(h http.Header, headerName string) (string, error) {
	// Headers are case-insensitive
	values := h[http.CanonicalHeaderKey(headerName)]
	if len(values) == 0 {
		return "", fmt.Errorf("component not found: header %s", headerName)
	}

	// Join multiple values with comma and space, trimmed
	return strings.TrimSpace(strings.Join(values, ", ")), nil
}

// queryParamValue returns the value of the query parameter named in an
// @query-param component.
func (c *Canonicalizer) queryParamValue(req *http.Request, component string) (string, error) {
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
	return values[0], nil
}

// buildSignatureParams creates the @signature-params line. When params
// came from a received Signature-Input field, its member value is used
// byte for byte (RFC 9421 section 2.3): the verifier must not re-order or
// drop parameters it does not know, such as tag.
func (c *Canonicalizer) buildSignatureParams(sigName string, params *SignatureInputParams) string {
	if params.Raw != "" {
		return fmt.Sprintf(`"@signature-params": %s`, params.Raw)
	}
	return fmt.Sprintf(`"@signature-params": %s`, FormatSignatureParams(params))
}

// FormatSignatureParams serialises the covered components and parameters in
// the order the specification requires of signers: keyid, alg, created,
// expires, nonce, tag, omitting empty ones.
func FormatSignatureParams(params *SignatureInputParams) string {
	var parts []string

	// Add covered components
	components := make([]string, len(params.CoveredComponents))
	copy(components, params.CoveredComponents)
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
	if params.Tag != "" {
		parts = append(parts, fmt.Sprintf(`tag="%s"`, params.Tag))
	}

	return strings.Join(parts, ";")
}
