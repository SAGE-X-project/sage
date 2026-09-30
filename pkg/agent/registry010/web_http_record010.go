package registry010

import (
	"bufio"
	"context"
	"io"
	"net/netip"
	"strconv"
	"strings"
)

const maxWebHTTPHeader010 = 16384

// FetchWebRegistryRecord010 reads one bounded Content-Length HTTP/1.1 response
// on the same newly authenticated TLS connection and checks its record proofs.
// Other response framing and controller or mutation history are outside this
// bounded adapter; success is not complete REG-08 authority.
func FetchWebRegistryRecord010(ctx context.Context, did string, allowedOrigins []string, destination netip.AddrPort, allowedDestinations []netip.AddrPort, rootDER []byte, now int64) ([]byte, error) {
	connection, err := dialWebRegistryTLS010(ctx, did, allowedOrigins, destination, allowedDestinations, rootDER)
	if err != nil {
		return nil, err
	}
	defer func() { _ = connection.Close() }()
	rest := strings.TrimPrefix(did, "did:sage:web:")
	domain, agent, _ := strings.Cut(rest, ":")
	request := "GET /.well-known/sage/agents/" + agent + " HTTP/1.1\r\nHost: " + domain + "\r\nAccept: application/json\r\nCache-Control: no-cache, no-store\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		return nil, ErrUnreachable
	}
	return readWebRegistryHTTP010(bufio.NewReader(connection), did, now)
}

func readWebRegistryHTTP010(reader *bufio.Reader, did string, now int64) ([]byte, error) {
	remaining := maxWebHTTPHeader010
	line := func() (string, error) {
		part, err := reader.ReadSlice('\n')
		remaining -= len(part)
		if err != nil || remaining < 0 || len(part) < 2 || part[len(part)-2] != '\r' {
			return "", ErrInvalidRecord010
		}
		return string(part[:len(part)-2]), nil
	}
	statusLine, err := line()
	if err != nil || len(statusLine) < 12 || !strings.HasPrefix(statusLine, "HTTP/1.1 ") || statusLine[9] < '0' || statusLine[9] > '9' || statusLine[10] < '0' || statusLine[10] > '9' || statusLine[11] < '0' || statusLine[11] > '9' || (len(statusLine) > 12 && statusLine[12] != ' ') {
		return nil, ErrInvalidRecord010
	}
	status, _ := strconv.Atoi(statusLine[9:12])
	fields := make([]HeaderField010, 0, 16)
	for {
		value, readErr := line()
		if readErr != nil {
			return nil, readErr
		}
		if value == "" {
			break
		}
		if len(fields) == 64 || strings.HasPrefix(value, " ") || strings.HasPrefix(value, "\t") {
			return nil, ErrInvalidRecord010
		}
		name, fieldValue, found := strings.Cut(value, ":")
		field := HeaderField010{Name: name, Value: strings.Trim(fieldValue, " \t")}
		if !found || !fieldLine010(field) {
			return nil, ErrInvalidRecord010
		}
		fields = append(fields, field)
	}
	if err := CheckWebRegistryResponsePolicy010(status, fields, nil); err != nil {
		return nil, err
	}
	seenLength := false
	length := 0
	for _, field := range fields {
		if asciiEqualFold010(field.Name, "transfer-encoding") || asciiEqualFold010(field.Name, "trailer") {
			return nil, ErrInvalidRecord010
		}
		if !asciiEqualFold010(field.Name, "content-length") {
			continue
		}
		if seenLength || field.Value == "" {
			return nil, ErrInvalidRecord010
		}
		seenLength = true
		for i := 0; i < len(field.Value); i++ {
			c := field.Value[i]
			if c < '0' || c > '9' {
				return nil, ErrInvalidRecord010
			}
			digit := int(c - '0')
			if length > (maxWebBody010-digit)/10 {
				return nil, ErrSizeExceeded010
			}
			length = length*10 + digit
		}
	}
	if !seenLength {
		return nil, ErrInvalidRecord010
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, ErrUnreachable
	}
	if err := CheckWebRegistryProofs010(body, did, now); err != nil {
		return nil, err
	}
	return body, nil
}
