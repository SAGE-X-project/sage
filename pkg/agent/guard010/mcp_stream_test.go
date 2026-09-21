package guard010

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// Scripted malformed frames stay in an offline unit fixture; runtime tests use
// benign messages only and do not provide an attack or bypass program.
type streamFixture struct {
	net.Conn
	input        *bytes.Reader
	output       bytes.Buffer
	readBytes    int
	chunk        int
	failWrite    bool
	failDeadline bool
	closes       atomic.Int64
}

func (f *streamFixture) Read(p []byte) (int, error) {
	if f.chunk > 0 && len(p) > f.chunk {
		p = p[:f.chunk]
	}
	n, e := f.input.Read(p)
	f.readBytes += n
	return n, e
}
func (f *streamFixture) Write(p []byte) (int, error) {
	if f.failWrite {
		return 0, io.ErrClosedPipe
	}
	if f.chunk > 0 && len(p) > f.chunk {
		p = p[:f.chunk]
	}
	return f.output.Write(p)
}
func (f *streamFixture) SetReadDeadline(time.Time) error {
	if f.failDeadline {
		return errors.New("deadline")
	}
	return nil
}
func (f *streamFixture) SetWriteDeadline(time.Time) error { return f.SetReadDeadline(time.Time{}) }
func (f *streamFixture) Close() error                     { f.closes.Add(1); return nil }
func streamFrame(b []byte) []byte {
	f := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(f, uint32(len(b)))
	copy(f[4:], b)
	return f
}
func TestMCPStreamBoundedFraming(t *testing.T) {
	for _, size := range []int{1, mcpWireLimit} {
		b := bytes.Repeat([]byte("x"), size)
		f := &streamFixture{input: bytes.NewReader(append(streamFrame(b), streamFrame([]byte("next"))...)), chunk: 3}
		s, e := newMCPStream(f, time.Second)
		if e != nil {
			t.Fatal(e)
		}
		got, e := s.Receive(context.Background())
		if e != nil || !bytes.Equal(got, b) || f.readBytes != size+4 {
			t.Fatal("exact frame", e)
		}
		if e = s.Send(context.Background(), b); e != nil || !bytes.Equal(f.output.Bytes(), streamFrame(b)) {
			t.Fatal("short writes", e)
		}
		next, e := s.Receive(context.Background())
		if e != nil || string(next) != "next" {
			t.Fatal("next frame", e)
		}
		_ = s.Close()
		_ = s.Close()
		if f.closes.Load() != 1 {
			t.Fatal("close repeated")
		}
	}
}
func TestMCPStreamRejectsInvalidFrames(t *testing.T) {
	oversized := make([]byte, 4)
	binary.BigEndian.PutUint32(oversized, mcpWireLimit+1)
	for _, raw := range [][]byte{{}, {0, 0}, {0, 0, 0, 0}, oversized, {0, 0, 0, 3, 'x'}} {
		f := &streamFixture{input: bytes.NewReader(raw)}
		s, _ := newMCPStream(f, time.Second)
		if b, e := s.Receive(context.Background()); e == nil || b != nil || f.closes.Load() != 1 {
			t.Fatal("invalid input retained")
		}
		if e := s.Send(context.Background(), []byte("later")); e == nil || f.output.Len() != 0 {
			t.Fatal("stream reused")
		}
	}
	for _, size := range []int{0, mcpWireLimit + 1} {
		f := &streamFixture{}
		s, _ := newMCPStream(f, time.Second)
		if e := s.Send(context.Background(), make([]byte, size)); e == nil || f.output.Len() != 0 || f.closes.Load() != 1 {
			t.Fatal("invalid output")
		}
	}
}
func TestMCPStreamProviderFailure(t *testing.T) {
	for _, f := range []*streamFixture{{failWrite: true}, {failDeadline: true}} {
		s, _ := newMCPStream(f, time.Second)
		if e := s.Send(context.Background(), []byte("hello")); e == nil || f.closes.Load() != 1 {
			t.Fatal("provider failure")
		}
	}
	f := &streamFixture{}
	s, _ := newMCPStream(f, time.Second)
	//nolint:staticcheck // Deliberately verify rejection of a missing host context.
	if e := s.Send(nil, []byte("hello")); e == nil || f.output.Len() != 0 {
		t.Fatal("nil context")
	}
	if _, e := newMCPStream(nil, time.Second); e == nil {
		t.Fatal("nil connection")
	}
	for _, bound := range []time.Duration{0, -1, 31 * time.Second} {
		if _, e := newMCPStream(f, bound); e == nil {
			t.Fatal("invalid bound")
		}
	}
}
func TestMCPStreamRuntimeCancellation(t *testing.T) {
	for _, sending := range []bool{false, true} {
		for _, cancelContext := range []bool{false, true} {
			x, y := net.Pipe()
			s, _ := newMCPStream(x, 40*time.Millisecond)
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				if sending {
					done <- s.Send(ctx, []byte("hello"))
				} else {
					_, e := s.Receive(ctx)
					done <- e
				}
			}()
			if cancelContext {
				cancel()
			}
			select {
			case e := <-done:
				if e == nil || !s.closed.Load() {
					t.Fatal("blocked I/O succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("I/O did not settle")
			}
			cancel()
			_ = s.Close()
			_ = y.Close()
		}
	}
}
func TestMCPStreamRejectsOverlappingOperations(t *testing.T) {
	for _, sending := range []bool{false, true} {
		x, y := net.Pipe()
		s, _ := newMCPStream(x, time.Second)
		defer y.Close()
		done := make(chan error, 1)
		if sending {
			go func() { done <- s.Send(context.Background(), []byte("one")) }()
			awaitHost(t, s.sending.Load)
		} else {
			go func() { _, e := s.Receive(context.Background()); done <- e }()
			awaitHost(t, s.receiving.Load)
		}
		var e error
		if sending {
			e = s.Send(context.Background(), []byte("two"))
		} else {
			_, e = s.Receive(context.Background())
		}
		if e == nil {
			t.Fatal("overlap accepted")
		}
		select {
		case e = <-done:
			if e == nil {
				t.Fatal("original survived overlap")
			}
		case <-time.After(time.Second):
			t.Fatal("original blocked")
		}
	}
}
func TestMCPStreamLoopbackRuntime(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	server := make(chan net.Conn, 1)
	failure := make(chan error, 1)
	go func() {
		c, e := listener.Accept()
		if e != nil {
			failure <- e
			return
		}
		server <- c
	}()
	x, e := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	var y net.Conn
	select {
	case y = <-server:
	case e = <-failure:
		t.Fatal(e)
	case <-time.After(time.Second):
		t.Fatal("accept")
	}
	a, _ := newMCPStream(x, time.Second)
	b, _ := newMCPStream(y, time.Second)
	defer a.Close()
	defer b.Close()
	done := make(chan error, 1)
	go func() {
		wire, e := b.Receive(context.Background())
		if e == nil {
			e = b.Send(context.Background(), wire)
		}
		done <- e
	}()
	if e = a.Send(context.Background(), []byte("benign round trip")); e != nil {
		t.Fatal(e)
	}
	got, e := a.Receive(context.Background())
	if e != nil || string(got) != "benign round trip" {
		t.Fatal("round trip", e)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}

type panicStreamFixture struct {
	*streamFixture
	operation string
}

func (f *panicStreamFixture) Write(p []byte) (int, error) {
	if f.operation == "write" {
		_, _ = f.output.Write(p[:1])
		panic("write")
	}
	return f.streamFixture.Write(p)
}
func (f *panicStreamFixture) Read(p []byte) (int, error) {
	if f.operation == "read" {
		_, _ = f.streamFixture.Read(p[:1])
		panic("read")
	}
	return f.streamFixture.Read(p)
}
func (f *panicStreamFixture) SetReadDeadline(d time.Time) error {
	if f.operation == "read deadline" {
		panic("deadline")
	}
	return f.streamFixture.SetReadDeadline(d)
}
func (f *panicStreamFixture) SetWriteDeadline(d time.Time) error {
	if f.operation == "write deadline" {
		panic("deadline")
	}
	return f.streamFixture.SetWriteDeadline(d)
}
func TestMCPStreamProviderPanic(t *testing.T) {
	for _, operation := range []string{"write", "read", "write deadline", "read deadline"} {
		f := &panicStreamFixture{streamFixture: &streamFixture{input: bytes.NewReader(streamFrame([]byte("benign")))}, operation: operation}
		s, _ := newMCPStream(f, time.Second)
		var e error
		if operation == "write" || operation == "write deadline" {
			e = s.Send(context.Background(), []byte("benign"))
		} else {
			var b []byte
			b, e = s.Receive(context.Background())
			if b != nil {
				t.Fatal("panic published bytes")
			}
		}
		if e == nil || !s.closed.Load() || f.closes.Load() != 1 {
			t.Fatal("panic left stream live", operation)
		}
		if e = s.Send(context.Background(), []byte("later")); e == nil {
			t.Fatal("panic stream reused")
		}
	}
}
