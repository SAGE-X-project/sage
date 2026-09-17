package hpke

import (
	"bytes"
	"strconv"
	"strings"
)

func parseHTTPBytes010(raw []byte, target string, response bool) (HTTPMessage010, error) {
	fail := func() (HTTPMessage010, error) { return HTTPMessage010{}, errCompletion010 }
	authority, x := httpEndpoint010(target)
	if x != nil || len(raw) > 73728 {
		return fail()
	}
	at := bytes.Index(raw, []byte("\r\n\r\n"))
	if at < 0 || at > 36864 {
		return fail()
	}
	lines := strings.Split(string(raw[:at]), "\r\n")
	if len(lines) < 2 || len(lines[0]) > 4096 {
		return fail()
	}
	fieldBytes := 0
	for _, line := range lines[1:] {
		fieldBytes += len(line) + 2
	}
	if fieldBytes > 32768 {
		return fail()
	}
	m := HTTPMessage010{Body: append([]byte{}, raw[at+4:]...)}
	if response {
		parts := strings.SplitN(lines[0], " ", 3)
		if len(parts) != 3 || parts[0] != "HTTP/1.1" || len(parts[1]) != 3 || !httpASCII010(parts[2]) {
			return fail()
		}
		m.Status, x = strconv.Atoi(parts[1])
		if x != nil || m.Status < 200 || m.Status > 599 || m.Status == 204 {
			return fail()
		}
	} else {
		path := strings.TrimPrefix(target, "https://"+authority)
		if lines[0] != "POST "+path+" HTTP/1.1" {
			return fail()
		}
		m.Method = "POST"
		m.Target = target
		m.Authority = authority
	}
	seen := map[string]bool{}
	length := ""
	host := ""
	for _, line := range lines[1:] {
		k, v, ok := strings.Cut(line, ":")
		if !ok || !httpToken010(k) {
			return fail()
		}
		k = strings.ToLower(k)
		v = strings.Trim(v, " \t")
		if k == "content-length" {
			if seen[k] {
				return fail()
			}
			length = v
		}
		if k == "host" {
			if seen[k] {
				return fail()
			}
			host = v
		}
		if k == "transfer-encoding" || k == "upgrade" || k == "expect" || k == "trailer" || k == "content-encoding" || (k == "connection" && (seen[k] || v != "close")) {
			return fail()
		}
		seen[k] = true
		m.Headers = append(m.Headers, [2]string{k, v})
	}
	if length != strconv.Itoa(len(m.Body)) || (!response && host != authority) || (response && seen["host"]) {
		return fail()
	}
	if _, x = httpHeaders010(m); x != nil {
		return fail()
	}
	return m, nil
}
func encodeHTTPBytes010(m HTTPMessage010, target string) ([]byte, error) {
	if _, x := httpHeaders010(m); x != nil {
		return nil, x
	}
	authority, x := httpEndpoint010(target)
	if x != nil {
		return nil, x
	}
	var line string
	if m.Status == 0 {
		if m.Method != "POST" || m.Target != target || m.Authority != authority {
			return nil, errCompletion010
		}
		line = "POST " + strings.TrimPrefix(target, "https://"+authority) + " HTTP/1.1"
	} else {
		if m.Method != "" || m.Target != "" || m.Authority != "" {
			return nil, errCompletion010
		}
		line = "HTTP/1.1 " + strconv.Itoa(m.Status) + " SAGE"
	}
	var out bytes.Buffer
	out.WriteString(line + "\r\n")
	if m.Status == 0 {
		out.WriteString("Host: " + authority + "\r\n")
	}
	for _, p := range m.Headers {
		k := strings.ToLower(p[0])
		if k == "host" || k == "content-length" || k == "connection" {
			return nil, errCompletion010
		}
		out.WriteString(p[0] + ": " + p[1] + "\r\n")
	}
	out.WriteString("Content-Length: " + strconv.Itoa(len(m.Body)) + "\r\nConnection: close\r\n\r\n")
	out.Write(m.Body)
	if _, x = parseHTTPBytes010(out.Bytes(), target, m.Status != 0); x != nil {
		return nil, x
	}
	return out.Bytes(), nil
}
