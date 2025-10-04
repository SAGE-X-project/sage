// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package rfc9421

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SignatureInputParams represents the parameters from a Signature-Input header
type SignatureInputParams struct {
	CoveredComponents []string
	KeyID             string
	Algorithm         string
	Created           int64
	Expires           int64
	Nonce             string
}

// ParseSignatureInput parses the Signature-Input header according to RFC 9421
func ParseSignatureInput(input string) (map[string]*SignatureInputParams, error) {
	result := make(map[string]*SignatureInputParams)
	
	// Split by comma to handle multiple signatures
	signatures := splitSignatures(input)
	
	for _, sig := range signatures {
		sig = strings.TrimSpace(sig)
		if sig == "" {
			continue
		}
		
		// Parse signature name and value
		parts := strings.SplitN(sig, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid signature format: %s", sig)
		}
		
		sigName := strings.TrimSpace(parts[0])
		sigValue := strings.TrimSpace(parts[1])
		
		params, err := parseSignatureValue(sigValue)
		if err != nil {
			return nil, fmt.Errorf("failed to parse signature '%s': %w", sigName, err)
		}
		
		result[sigName] = params
	}
	
	return result, nil
}

// ParseSignature parses the Signature header containing base64-encoded signatures
func ParseSignature(input string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	
	// Split by comma to handle multiple signatures
	signatures := splitSignatures(input)
	
	for _, sig := range signatures {
		sig = strings.TrimSpace(sig)
		if sig == "" {
			continue
		}
		
		// Parse signature name and value
		parts := strings.SplitN(sig, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid signature format: %s", sig)
		}
		
		sigName := strings.TrimSpace(parts[0])
		sigValue := strings.TrimSpace(parts[1])
		
		// RFC 8941 byte sequence format: :base64:
		if !strings.HasPrefix(sigValue, ":") || !strings.HasSuffix(sigValue, ":") {
			return nil, fmt.Errorf("invalid byte sequence format for signature '%s'", sigName)
		}
		
		// Extract base64 content
		b64Content := sigValue[1 : len(sigValue)-1]
		
		// Decode base64
		decoded, err := base64.StdEncoding.DecodeString(b64Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 for signature '%s': %w", sigName, err)
		}
		
		result[sigName] = decoded
	}
	
	return result, nil
}

// parseSignatureValue parses the value part of a signature input
func parseSignatureValue(value string) (*SignatureInputParams, error) {
	params := &SignatureInputParams{}
	
	// Find the components list in parentheses
	compStart := strings.Index(value, "(")
	compEnd := strings.Index(value, ")")
	
	if compStart == -1 || compEnd == -1 || compEnd < compStart {
		return nil, fmt.Errorf("invalid component list format")
	}
	
	// Parse components
	componentStr := value[compStart+1 : compEnd]
	components, err := parseComponents(componentStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse components: %w", err)
	}
	params.CoveredComponents = components
	
	// Parse parameters after the closing parenthesis
	if compEnd+1 < len(value) {
		paramStr := value[compEnd+1:]
		if err := parseParameters(paramStr, params); err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	}
	
	return params, nil
}

// parseComponents parses the component list
func parseComponents(componentStr string) ([]string, error) {
	var components []string
	current := ""
	inQuote := false
	depth := 0
	
	for _, ch := range componentStr {
		switch ch {
		case '"':
			inQuote = !inQuote
			current += string(ch)
		case ';':
			// Semicolon is part of structured components like "@query-param";name="id"
			current += string(ch)
		case ' ', '\t':
			if inQuote || depth > 0 {
				current += string(ch)
			} else if current != "" {
				// End of component
				components = append(components, current)
				current = ""
			}
		default:
			current += string(ch)
		}
	}
	
	// Handle any remaining component
	if current != "" {
		components = append(components, current)
	}
	
	if inQuote {
		return nil, fmt.Errorf("unclosed quote in component list")
	}
	
	// Validate components
	for _, comp := range components {
		comp = strings.TrimSpace(comp)
		// Components must be quoted strings or structured components
		if !strings.HasPrefix(comp, `"`) || (!strings.HasSuffix(comp, `"`) && !strings.Contains(comp, ";")) {
			return nil, fmt.Errorf("invalid component format: %s", comp)
		}
	}
	
	return components, nil
}

// parseParameters parses the signature parameters
func parseParameters(paramStr string, params *SignatureInputParams) error {
	// Split by semicolon
	parts := strings.Split(paramStr, ";")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Split key=value
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		
		key := strings.TrimSpace(strings.ToLower(kv[0]))
		value := strings.TrimSpace(kv[1])
		
		// Remove quotes if present
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}
		
		switch key {
		case "keyid":
			params.KeyID = value
		case "alg":
			params.Algorithm = value
		case "created":
			created, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid created timestamp: %w", err)
			}
			params.Created = created
		case "expires":
			expires, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid expires timestamp: %w", err)
			}
			params.Expires = expires
		case "nonce":
			params.Nonce = value
		}
	}
	
	return nil
}

// splitSignatures splits multiple signatures by comma, handling nested structures
func splitSignatures(input string) []string {
	var signatures []string
	current := ""
	depth := 0
	inQuote := false
	
	for _, ch := range input {
		switch ch {
		case '"':
			inQuote = !inQuote
			current += string(ch)
		case '(':
			if !inQuote {
				depth++
			}
			current += string(ch)
		case ')':
			if !inQuote {
				depth--
			}
			current += string(ch)
		case ',':
			if !inQuote && depth == 0 {
				signatures = append(signatures, current)
				current = ""
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}
	
	if current != "" {
		signatures = append(signatures, current)
	}
	
	return signatures
}

// parseQueryParam extracts the parameter name from a @query-param component
func parseQueryParam(component string) (string, error) {
	// Remove quotes and trim
	component = strings.TrimSpace(component)
	
	// Check if it starts with "@query-param"
	if !strings.Contains(component, "@query-param") {
		return "", fmt.Errorf("not a query-param component")
	}
	
	// Find the name parameter
	re := regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(component)
	if len(matches) < 2 {
		return "", fmt.Errorf("missing name parameter in @query-param")
	}
	
	return matches[1], nil
}