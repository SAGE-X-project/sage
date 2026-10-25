package registry011

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func providerRequestHead(length int) []byte {
	return fmt.Appendf(nil, "POST / HTTP/1.1\r\nHost: rpc.sepolia.ethpandaops.io\r\nContent-Type: application/json\r\nAccept: application/json\r\nAccept-Encoding: identity\r\nConnection: close\r\nContent-Length: %d\r\n\r\n", length)
}

// readProviderHead stops at the bounded original header terminator. It does not
// let an HTTP library merge duplicate framing fields or discard wire order.
func readProviderHead(r *bufio.Reader, maximum int) ([]byte, error) {
	var head []byte
	for len(head) <= maximum {
		b, err := r.ReadByte()
		if err != nil {
			return nil, ErrProviderCapture
		}
		head = append(head, b)
		if len(head) >= 4 && bytes.Equal(head[len(head)-4:], []byte("\r\n\r\n")) {
			if len(head) > maximum {
				return nil, ErrProviderCapture
			}
			return head, nil
		}
	}
	return nil, ErrProviderCapture
}

func providerHeaderFields(head []byte) ([]ProviderHeader, error) {
	if !utf8.Valid(head) || !bytes.HasSuffix(head, []byte("\r\n\r\n")) {
		return nil, ErrProviderCapture
	}
	lines := strings.Split(string(head[:len(head)-4]), "\r\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "HTTP/1.1 200 ") || len(lines[0]) > 256 {
		return nil, ErrProviderCapture
	}
	for i := 0; i < len(lines[0]); i++ {
		if lines[0][i] < 32 || lines[0][i] == 127 {
			return nil, ErrProviderCapture
		}
	}
	var out []ProviderHeader
	single := map[string]bool{}
	for _, line := range lines[1:] {
		name, value, ok := strings.Cut(line, ":")
		if !ok || name == "" || len(name) > 256 || len(out) >= 128 {
			return nil, ErrProviderCapture
		}
		for i := 0; i < len(name); i++ {
			b := name[i]
			if (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') && (b < '0' || b > '9') && !strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
				return nil, ErrProviderCapture
			}
		}
		for i := 0; i < len(value); i++ {
			if value[i] < 32 && value[i] != '\t' || value[i] == 127 {
				return nil, ErrProviderCapture
			}
		}
		lower := strings.ToLower(name)
		switch lower {
		case "content-type", "content-encoding", "content-length", "transfer-encoding":
			if single[lower] {
				return nil, ErrProviderCapture
			}
			single[lower] = true
		case "trailer":
			return nil, ErrProviderCapture
		}
		out = append(out, ProviderHeader{name, strings.Trim(value, " \t")})
	}
	return out, nil
}

func readProviderHTTP(reader *bufio.Reader, limits ProviderCaptureLimits) ([]byte, []ProviderHeader, []byte, json.RawMessage, error) {
	if !limits.valid() {
		return nil, nil, nil, nil, ErrConfiguration
	}
	head, err := readProviderHead(reader, limits.HeaderBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	headers, err := providerHeaderFields(head)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	values := map[string]string{}
	for _, h := range headers {
		values[strings.ToLower(h.Name)] = h.Value
	}
	encoding, hasEncoding := values["content-encoding"]
	contentType, _, err := mime.ParseMediaType(values["content-type"])
	if err != nil || contentType != "application/json" ||
		hasEncoding && encoding != "identity" {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	length, hasLength := values["content-length"]
	transfer, hasTransfer := values["transfer-encoding"]
	if hasLength {
		n, e := strconv.ParseUint(length, 10, 64)
		if e != nil || strconv.FormatUint(n, 10) != length || n > uint64(limits.BodyBytes) {
			return nil, nil, nil, nil, ErrProviderCapture
		}
	}
	if hasTransfer && (!strings.EqualFold(transfer, "chunked") || hasLength) {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	request := &http.Request{Method: http.MethodPost}
	framed := bufio.NewReader(io.MultiReader(bytes.NewReader(head), reader))
	response, err := http.ReadResponse(framed, request)
	if err != nil {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK || response.Proto != "HTTP/1.1" || response.Uncompressed {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(limits.BodyBytes)+1))
	if err != nil || len(body) > limits.BodyBytes || len(response.Trailer) != 0 {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	// Exactly one exchange on a Connection: close request must reach real EOF.
	// Check the SAME buffered reader: a second reader would hide prefetched tail.
	if _, err := framed.ReadByte(); err != io.EOF {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	result, err := DecodeRPCResult(body, 1, RPCResponseLimits{limits.BodyBytes, limits.JSONDepth})
	if err != nil {
		return nil, nil, nil, nil, ErrProviderCapture
	}
	return head, headers, body, result, nil
}
