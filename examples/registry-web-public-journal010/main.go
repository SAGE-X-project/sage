// Local single-request HTTPS publisher backed by a Registry write journal.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type configuration struct {
	ServerCert  string `json:"server_cert"`
	ServerKey   string `json:"server_key"`
	JournalPath string `json:"journal_path"`
	DID         string `json:"did"`
	Source      string `json:"source"`
	Now         int64  `json:"now"`
}

type readOnlyAuthority struct{}

func (readOnlyAuthority) AuthenticatedActor(context.Context) (string, error) {
	return "", registry010.ErrRejected
}
func (readOnlyAuthority) Delegated(context.Context, string, string, string, string, string) (bool, error) {
	return false, registry010.ErrRejected
}

func main() {
	var cfg configuration
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 32_769))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&cfg) != nil || decoder.Decode(new(any)) != io.EOF {
		os.Exit(2)
	}
	certDER, certErr := base64.RawURLEncoding.Strict().DecodeString(cfg.ServerCert)
	keyDER, keyErr := base64.RawURLEncoding.Strict().DecodeString(cfg.ServerKey)
	key, parseErr := x509.ParsePKCS8PrivateKey(keyDER)
	if certErr != nil || keyErr != nil || parseErr != nil || cfg.JournalPath == "" {
		os.Exit(2)
	}
	journal, err := registry010.OpenWebRegistryWriteJournal010(
		cfg.JournalPath, cfg.DID, cfg.Source, readOnlyAuthority{}, false)
	if err != nil {
		fmt.Println(`{"verdict":"RECORD_UNREACHABLE"}`)
		return
	}
	defer func() { _ = journal.Close() }()
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{certDER}, PrivateKey: key}},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		os.Exit(2)
	}
	defer func() { _ = listener.Close() }()
	fmt.Printf("PORT %d\n", listener.Addr().(*net.TCPAddr).Port)
	connection, err := listener.Accept()
	if err != nil {
		os.Exit(2)
	}
	defer func() { _ = connection.Close() }()
	_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
	parts := strings.Split(strings.TrimPrefix(cfg.DID, "did:sage:web:"), ":")
	if len(parts) != 2 {
		fmt.Println(`{"verdict":"RECORD_UNREACHABLE"}`)
		return
	}
	request := "GET /.well-known/sage/agents/" + parts[1] + " HTTP/1.1\r\nHost: " + parts[0] +
		"\r\nAccept: application/json\r\nCache-Control: no-cache, no-store\r\nConnection: close\r\n\r\n"
	actual := make([]byte, len(request))
	if _, err := io.ReadFull(connection, actual); err != nil || string(actual) != request {
		fmt.Println(`{"verdict":"RECORD_UNREACHABLE"}`)
		return
	}
	body, err := journal.PublicEnvelope010(cfg.DID, cfg.Source, cfg.Now)
	if err != nil {
		fmt.Println(`{"verdict":"RECORD_UNREACHABLE"}`)
		return
	}
	_, err = fmt.Fprintf(connection, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nContent-Length: %d\r\n\r\n", len(body))
	if err == nil {
		var written int
		written, err = connection.Write(body)
		if written != len(body) {
			err = io.ErrShortWrite
		}
	}
	if err != nil {
		fmt.Println(`{"verdict":"RECORD_UNREACHABLE"}`)
		return
	}
	fmt.Println(`{"verdict":"PUBLISHED"}`)
}
