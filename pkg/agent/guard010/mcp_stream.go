package guard010

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const mcpWireLimit = 32768

// mcpStream is a private, deployment-selected framing adapter, not MCP stdio or
// a negotiated protocol extension. A four-byte unsigned big-endian length precedes
// each exact SAGE envelope. No framing byte enters the signed envelope.
//
// Construction transfers exclusive connection ownership on success. The trusted
// host must supply a connection whose deadlines and Close interrupt pending I/O
// within finite bounds (for example a native TCP or Unix connection). Callers must
// not retain connection aliases. At most one Send and one Receive may be active;
// there is no input pump, read-ahead buffer or goroutine per received frame.
type mcpStream struct {
	conn               net.Conn
	bound              time.Duration
	sending, receiving atomic.Bool
	closed             atomic.Bool
	once               sync.Once
}

func newMCPStream(conn net.Conn, bound time.Duration) (*mcpStream, error) {
	if conn == nil || bound <= 0 || bound > 30*time.Second {
		return nil, ErrInvalid
	}
	return &mcpStream{conn: conn, bound: bound}, nil
}

func (s *mcpStream) Close() error {
	s.closed.Store(true)
	s.once.Do(func() { _ = s.conn.Close() })
	return nil
}

// begin applies one absolute deadline to the whole frame, including its header.
// Cancellation closes both directions: partial or uncertain frames cannot resume
// on this stream. The cancellation callback also wakes I/O without a ctx deadline.
func (s *mcpStream) begin(ctx context.Context, active *atomic.Bool, setDeadline func(time.Time) error) (func() error, error) {
	if ctx == nil || s.closed.Load() || !active.CompareAndSwap(false, true) {
		_ = s.Close()
		return nil, ErrInvalid
	}
	if ctx.Err() != nil {
		_ = s.Close()
		active.Store(false)
		return nil, ErrInvalid
	}
	deadline := time.Now().Add(s.bound)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if setDeadline(deadline) != nil {
		_ = s.Close()
		active.Store(false)
		return nil, ErrInvalid
	}
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	return func() error {
		defer active.Store(false)
		stopped := stop()
		if !stopped || ctx.Err() != nil || s.closed.Load() || !time.Now().Before(deadline) {
			_ = s.Close()
			return ErrInvalid
		}
		return nil
	}, nil
}

func (s *mcpStream) Send(ctx context.Context, wire []byte) (err error) {
	defer func() {
		if recover() != nil {
			_ = s.Close()
			s.sending.Store(false)
			err = ErrInvalid
		}
	}()
	if len(wire) == 0 || len(wire) > mcpWireLimit {
		_ = s.Close()
		return ErrInvalid
	}
	finish, err := s.begin(ctx, &s.sending, s.conn.SetWriteDeadline)
	if err != nil {
		return err
	}
	defer func() {
		if recover() != nil {
			err = ErrInvalid
		}
		if err != nil {
			_ = s.Close()
		}
		if finish() != nil || err != nil {
			_ = s.Close()
			err = ErrInvalid
		}
	}()
	frame := make([]byte, 4+len(wire))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(wire)))
	copy(frame[4:], wire)
	for len(frame) != 0 {
		n, e := s.conn.Write(frame)
		if e != nil || n <= 0 || n > len(frame) {
			return ErrInvalid
		}
		frame = frame[n:]
	}
	return nil
}

func (s *mcpStream) Receive(ctx context.Context) (wire []byte, err error) {
	defer func() {
		if recover() != nil {
			_ = s.Close()
			s.receiving.Store(false)
			wire, err = nil, ErrInvalid
		}
	}()
	finish, err := s.begin(ctx, &s.receiving, s.conn.SetReadDeadline)
	if err != nil {
		return nil, err
	}
	defer func() {
		if recover() != nil {
			err = ErrInvalid
		}
		if err != nil {
			_ = s.Close()
		}
		if finish() != nil || err != nil {
			_ = s.Close()
			wire, err = nil, ErrInvalid
		}
	}()
	var header [4]byte
	if _, err = io.ReadFull(s.conn, header[:]); err != nil {
		return nil, ErrInvalid
	}
	n := binary.BigEndian.Uint32(header[:])
	if n == 0 || n > mcpWireLimit {
		return nil, ErrInvalid
	}
	// Validate before allocation; read exactly this frame and never the next one.
	wire = make([]byte, int(n))
	if _, err = io.ReadFull(s.conn, wire); err != nil {
		return nil, ErrInvalid
	}
	return wire, nil
}
