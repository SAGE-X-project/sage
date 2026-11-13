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
	return readProviderHeadObserved(r, maximum, nil)
}

func readProviderHeadObserved(r *bufio.Reader, maximum int, meter *providerWorkMeter) ([]byte, error) {
	var head []byte
	for len(head) <= maximum {
		if next, handled, complete, err := tryReadProviderBufferedHeadObserved(r, maximum, head, meter); handled {
			if err != nil {
				return nil, err
			}
			head = next
			if complete {
				return head, nil
			}
			continue
		}
		if meter != nil && !meter.preflight(providerReadEvent, 1) {
			return nil, ErrProviderCapture
		}
		b, err := r.ReadByte()
		if err != nil {
			if meter != nil {
				meter.event(providerReadEvent, 0)
			}
			return nil, ErrProviderCapture
		}
		if meter != nil && (!meter.event(providerReadEvent, 1) || !meter.event(providerCopyEvent, 1)) {
			return nil, ErrProviderCapture
		}
		head = append(head, b)
		if len(head) >= 4 && meter != nil && !meter.event(providerCompareEvent, 8) {
			return nil, ErrProviderCapture
		}
		if len(head) >= 4 && bytes.Equal(head[len(head)-4:], []byte("\r\n\r\n")) {
			if len(head) > maximum {
				return nil, ErrProviderCapture
			}
			return head, nil
		}
	}
	return nil, ErrProviderCapture
}

// Admit only an already buffered, callback-free group primitive. Its original
// store lock and this meter lock keep every checked owner/window stable while
// ReadByte consumes at most Buffered bytes, so no fill or external read occurs.
// The complete upper must fit before using this path; otherwise the unchanged
// byte path preserves the exact refusal prefix and sticky failure behavior.
// No positive result survives this call, and every logical event is retained.
func tryReadProviderBufferedHeadObserved(r *bufio.Reader, maximum int, head []byte, meter *providerWorkMeter) ([]byte, bool, bool, error) {
	if meter == nil || meter.historyGroup == nil || r == nil || maximum <= len(head) {
		return head, false, false, nil
	}
	count := min(r.Buffered(), maximum-len(head))
	if count == 0 {
		return head, false, false, nil
	}
	// Each successful byte has read(1)+copy(1) = 4 logical units.
	// Starting with byte four, the original compare(8) adds another 9.
	comparisons := count
	if len(head) < 3 {
		comparisons = max(0, count-(3-len(head)))
	}
	byteWork, byteOK := providerUpperProduct(uint64(count), 4)
	compareWork, compareOK := providerUpperProduct(uint64(comparisons), 9)
	units, sumOK := providerUpperSum(byteWork, compareWork)
	if !byteOK || !compareOK || !sumOK {
		return head, false, false, nil
	}
	meter.mu.Lock()
	defer meter.mu.Unlock()
	// Other roles retain their original add/scan-output behavior. In
	// particular an unrelated scan body is never silently skipped here.
	if meter.historyGroup == nil || meter.historyRecovery != nil || meter.scanBody != nil || meter.owner != meter || meter.failed || meter.closed || !meter.hasOwner() {
		return head, false, false, nil
	}
	var upper candidateWorkVector
	upper[10] = units
	if !meter.fitsOwnedWindow(upper) {
		return head, false, false, nil
	}
	for i, value := range upper {
		if !candidateCapacity(0, meter.actual[i], value, meter.upper[i]) {
			return head, false, false, nil
		}
	}
	for range count {
		b, err := r.ReadByte()
		if err != nil {
			// Impossible for the exclusively held Buffered prefix, but
			// retain the original failed-read(0) logical event if refused.
			meter.actual[10]++
			return nil, true, false, ErrProviderCapture
		}
		meter.actual[10] += 4
		head = append(head, b)
		if len(head) >= 4 {
			meter.actual[10] += 9
			if bytes.Equal(head[len(head)-4:], []byte("\r\n\r\n")) {
				return head, true, true, nil
			}
		}
	}
	return head, true, false, nil
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
	return readProviderHTTPObserved(reader, limits, nil, 0)
}

func readProviderHTTPObserved(reader *bufio.Reader, limits ProviderCaptureLimits, meter *providerWorkMeter, wireBytes uint64) ([]byte, []ProviderHeader, []byte, json.RawMessage, error) {
	return readProviderHTTPDiagnosed(reader, limits, meter, wireBytes, nil)
}

// diagnostic is fixed DATA output only; nil preserves the other callers.
// The existing reads, parser order, caps and returned errors are unchanged.
func readProviderHTTPDiagnosed(reader *bufio.Reader, limits ProviderCaptureLimits, meter *providerWorkMeter, wireBytes uint64, diagnostic *providerFirstOperationDiagnostic) ([]byte, []ProviderHeader, []byte, json.RawMessage, error) {
	if !limits.valid() || meter != nil && (wireBytes == 0 || wireBytes > uint64(limits.WireBytes)) {
		diagnostic.reject(providerFirstFailureCaptureEntry, providerFirstReasonRefused, 0)
		return nil, nil, nil, nil, ErrConfiguration
	}
	head, err := readProviderHeadObserved(reader, limits.HeaderBytes, meter)
	if err != nil {
		if diagnostic != nil {
			diagnostic.reject(providerFirstFailureHead, providerFirstNativeFailureReason(err), 0)
		}
		return nil, nil, nil, nil, err
	}
	if meter != nil && !meter.event(providerCodecEvent, uint64(len(head))) {
		diagnostic.reject(providerFirstFailureCaptureWork, providerFirstReasonRefused, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	headers, err := providerHeaderFields(head)
	if err != nil {
		diagnostic.reject(providerFirstFailureHeaders, providerFirstReasonFormat, 0)
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
		diagnostic.reject(providerFirstFailureEncoding, providerFirstReasonFormat, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	length, hasLength := values["content-length"]
	transfer, hasTransfer := values["transfer-encoding"]
	if hasLength {
		n, e := strconv.ParseUint(length, 10, 64)
		if e != nil || strconv.FormatUint(n, 10) != length || n > uint64(limits.BodyBytes) {
			if diagnostic != nil {
				reason := providerFirstReasonFormat
				if e == nil && n > uint64(limits.BodyBytes) {
					reason = providerFirstReasonLimit
				}
				diagnostic.reject(providerFirstFailureLength, reason, n)
			}
			return nil, nil, nil, nil, ErrProviderCapture
		}
	}
	if hasTransfer && (!strings.EqualFold(transfer, "chunked") || hasLength) {
		diagnostic.reject(providerFirstFailureTransfer, providerFirstReasonFormat, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	request := &http.Request{Method: http.MethodPost}
	framed := bufio.NewReader(io.MultiReader(bytes.NewReader(head), reader))
	if meter != nil && !meter.event(providerCodecEvent, wireBytes) {
		diagnostic.reject(providerFirstFailureCaptureWork, providerFirstReasonRefused, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	response, err := http.ReadResponse(framed, request)
	if err != nil {
		if diagnostic != nil {
			diagnostic.reject(providerFirstFailureHTTP, providerFirstNativeFailureReason(err), 0)
		}
		return nil, nil, nil, nil, ErrProviderCapture
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK || response.Proto != "HTTP/1.1" || response.Uncompressed {
		diagnostic.reject(providerFirstFailureStatus, providerFirstReasonFormat, uint64(response.StatusCode))
		return nil, nil, nil, nil, ErrProviderCapture
	}
	if meter != nil && !meter.preflight(providerReadEvent, uint64(limits.BodyBytes)+1) {
		diagnostic.reject(providerFirstFailureCaptureWork, providerFirstReasonRefused, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(limits.BodyBytes)+1))
	if meter != nil && !meter.event(providerReadEvent, uint64(len(body))) || err != nil || len(body) > limits.BodyBytes || len(response.Trailer) != 0 {
		if diagnostic != nil {
			reason := providerFirstNativeFailureReason(err)
			if len(body) > limits.BodyBytes {
				reason = providerFirstReasonLimit
			} else if len(response.Trailer) != 0 {
				reason = providerFirstReasonTrailer
			}
			diagnostic.reject(providerFirstFailureBody, reason, uint64(len(body)))
		}
		return nil, nil, nil, nil, ErrProviderCapture
	}
	// Exactly one exchange on a Connection: close request must reach real EOF.
	// Check the SAME buffered reader: a second reader would hide prefetched tail.
	if meter != nil && !meter.preflight(providerReadEvent, 1) {
		diagnostic.reject(providerFirstFailureCaptureWork, providerFirstReasonRefused, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	_, eofErr := framed.ReadByte()
	n := uint64(0)
	if eofErr == nil {
		n = 1
	}
	if meter != nil && !meter.event(providerReadEvent, n) || eofErr != io.EOF {
		if diagnostic != nil {
			diagnostic.reject(providerFirstFailureEOF, providerFirstNativeFailureReason(eofErr), n)
		}
		return nil, nil, nil, nil, ErrProviderCapture
	}
	result, err := decodeRPCResultObserved(body, 1, RPCResponseLimits{limits.BodyBytes, limits.JSONDepth}, meter)
	if err != nil {
		diagnostic.reject(providerFirstFailureRPC, providerFirstReasonFormat, 0)
		return nil, nil, nil, nil, ErrProviderCapture
	}
	return head, headers, body, result, nil
}
