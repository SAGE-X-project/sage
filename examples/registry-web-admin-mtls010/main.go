// Local reference adapter for a single verified mTLS Registry write.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	ClientRoot  string `json:"client_root"`
	ClientPin   string `json:"client_pin"`
	Actor       string `json:"actor"`
	JournalPath string `json:"journal_path"`
	DID         string `json:"did"`
	Source      string `json:"source"`
	Now         int64  `json:"now"`
	Create      bool   `json:"create"`
}

type request struct {
	Candidate       string `json:"candidate"`
	Operation       string `json:"operation"`
	ExpectedVersion string `json:"expected_version"`
}

func decode(value string) ([]byte, error) {
	return base64.RawURLEncoding.Strict().DecodeString(value)
}

func main() {
	var cfg configuration
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 32768))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&cfg) != nil {
		os.Exit(2)
	}
	certDER, certErr := decode(cfg.ServerCert)
	keyDER, keyErr := decode(cfg.ServerKey)
	rootDER, rootErr := decode(cfg.ClientRoot)
	key, parseErr := x509.ParsePKCS8PrivateKey(keyDER)
	pin, pinErr := hex.DecodeString(cfg.ClientPin)
	if certErr != nil || keyErr != nil || rootErr != nil || parseErr != nil || pinErr != nil || len(pin) != 32 || cfg.Actor == "" || cfg.DID == "" || cfg.Source == "" || cfg.JournalPath == "" {
		os.Exit(2)
	}
	var fingerprint [32]byte
	copy(fingerprint[:], pin)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.Exit(2)
	}
	defer listener.Close()
	fmt.Printf("PORT %d\n", listener.Addr().(*net.TCPAddr).Port)
	connection, err := listener.Accept()
	if err != nil {
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := registry010.AcceptWebRegistryAdminMTLS010(ctx, connection,
		tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: key}, rootDER,
		map[[32]byte]string{fingerprint: cfg.Actor})
	if err != nil {
		fmt.Println(`{"verdict":"WRITE_REJECTED","committed":false}`)
		return
	}
	defer session.Close()
	if session.SetDeadline(time.Now().Add(5*time.Second)) != nil {
		fmt.Println(`{"verdict":"WRITE_REJECTED","committed":false}`)
		return
	}
	line, err := bufio.NewReaderSize(session, 150002).ReadSlice('\n')
	if err != nil || len(line) > 150000 || !strings.HasSuffix(string(line), "\n") {
		fmt.Println(`{"verdict":"WRITE_REJECTED","committed":false}`)
		return
	}
	var req request
	input := json.NewDecoder(strings.NewReader(string(line[:len(line)-1])))
	input.DisallowUnknownFields()
	if input.Decode(&req) != nil || input.Decode(new(any)) != io.EOF {
		fmt.Println(`{"verdict":"WRITE_REJECTED","committed":false}`)
		return
	}
	candidate, err := decode(req.Candidate)
	if err != nil || len(candidate) > 69632 {
		fmt.Println(`{"verdict":"WRITE_REJECTED","committed":false}`)
		return
	}
	journal, err := registry010.OpenWebRegistryWriteJournal010(cfg.JournalPath, cfg.DID, cfg.Source, session, cfg.Create)
	if err != nil {
		fmt.Println(`{"verdict":"RECORD_UNREACHABLE","committed":false}`)
		return
	}
	before := journal.Inspect()
	err = registry010.ApplyWebRegistryWrite010(ctx, journal, cfg.Source, cfg.DID,
		candidate, cfg.Now, req.ExpectedVersion, req.Operation)
	after := journal.Inspect()
	if journal.Close() != nil {
		err = registry010.ErrUnreachable
	}
	verdict := "TRANSITION_ACCEPT"
	if errors.Is(err, registry010.ErrRejected) {
		verdict = "WRITE_REJECTED"
	} else if errors.Is(err, registry010.ErrStale) {
		verdict = "RECORD_STALE"
	} else if err != nil {
		verdict = "RECORD_INVALID"
	}
	changed := len(after.History) != len(before.History)
	output, _ := json.Marshal(map[string]any{"verdict": verdict, "committed": changed})
	fmt.Println(string(output))
}
