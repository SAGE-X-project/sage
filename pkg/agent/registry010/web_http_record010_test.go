package registry010

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestWebRegistryHTTPRecord010(t *testing.T) {
	body := proofBody010(t, proofFixture010(t))
	response := func(header string, content []byte) *bufio.Reader {
		return bufio.NewReader(strings.NewReader(header + string(content)))
	}
	valid := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nContent-Length: %d\r\n\r\n", len(body))
	got, err := readWebRegistryHTTP010(response(valid, body), proofDID010, 100)
	if err != nil || string(got) != string(body) {
		t.Fatalf("valid record: %v", err)
	}
	for _, tc := range []struct {
		name, header string
		content      []byte
		want         error
	}{
		{"redirect", strings.Replace(valid, "200 OK", "301 Moved", 1), body, ErrUnreachable},
		{"wrong media", strings.Replace(valid, "application/json", "text/plain", 1), body, ErrInvalidRecord010},
		{"missing length", strings.Replace(valid, fmt.Sprintf("Content-Length: %d\r\n", len(body)), "", 1), body, ErrInvalidRecord010},
		{"duplicate length", strings.Replace(valid, "\r\n\r\n", "\r\nContent-Length: 1\r\n\r\n", 1), body, ErrInvalidRecord010},
		{"transfer coding", strings.Replace(valid, "\r\n\r\n", "\r\nTransfer-Encoding: chunked\r\n\r\n", 1), body, ErrInvalidRecord010},
		{"oversize", strings.Replace(valid, fmt.Sprintf("Content-Length: %d", len(body)), "Content-Length: 69633", 1), body, ErrSizeExceeded010},
		{"huge length", strings.Replace(valid, fmt.Sprintf("Content-Length: %d", len(body)), "Content-Length: 999999999999999999999999999999", 1), body, ErrSizeExceeded010},
		{"truncated", valid, body[:len(body)-1], ErrUnreachable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, got := readWebRegistryHTTP010(response(tc.header, tc.content), proofDID010, 100)
			if !errors.Is(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
