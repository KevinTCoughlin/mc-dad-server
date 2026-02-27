package container

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// rconTestServer is a minimal TCP server that speaks the Source RCON protocol
// for unit-testing the RCONClient.
type rconTestServer struct {
	ln       net.Listener
	password string
	// handler is called for each command packet; return the response body.
	handler func(cmd string) string
}

func newRCONTestServer(t *testing.T, password string, handler func(string) string) *rconTestServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &rconTestServer{ln: ln, password: password, handler: handler}
	return s
}

func (s *rconTestServer) Addr() string { return s.ln.Addr().String() }

func (s *rconTestServer) Close() { _ = s.ln.Close() }

// Serve accepts one connection and handles it synchronously.
func (s *rconTestServer) Serve(t *testing.T) {
	t.Helper()
	conn, err := s.ln.Accept()
	if err != nil {
		return // listener closed
	}
	defer func() { _ = conn.Close() }()

	for {
		id, pktType, body, err := readTestPacket(conn)
		if err != nil {
			return // connection closed or error
		}

		switch pktType {
		case packetTypeAuth:
			if body == s.password {
				writeTestPacket(t, conn, id, packetTypeAuthResponse, "")
			} else {
				writeTestPacket(t, conn, -1, packetTypeAuthResponse, "")
			}
		case packetTypeCommand:
			resp := ""
			if s.handler != nil {
				resp = s.handler(body)
			}
			writeTestPacket(t, conn, id, packetTypeResponse, resp)
		}
	}
}

func writeTestPacket(t *testing.T, w io.Writer, id, pktType int32, body string) {
	t.Helper()
	bodyBytes := []byte(body)
	size := int32(4 + 4 + len(bodyBytes) + 2)
	buf := make([]byte, 4+size)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(id))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(pktType))
	copy(buf[12:], bodyBytes)
	buf[12+len(bodyBytes)] = 0
	buf[13+len(bodyBytes)] = 0
	if _, err := w.Write(buf); err != nil {
		t.Logf("writeTestPacket: %v", err)
	}
}

func readTestPacket(r io.Reader) (id, pktType int32, body string, err error) {
	var sizeBuf [4]byte
	if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
		return 0, 0, "", err
	}
	size := int32(binary.LittleEndian.Uint32(sizeBuf[:]))
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, 0, "", err
	}
	id = int32(binary.LittleEndian.Uint32(payload[0:4]))
	pktType = int32(binary.LittleEndian.Uint32(payload[4:8]))
	bodyLen := size - 10
	if bodyLen > 0 {
		body = string(payload[8 : 8+bodyLen])
	}
	return id, pktType, body, nil
}

func TestRCONClient_ConnectAndCommand(t *testing.T) {
	srv := newRCONTestServer(t, "secret", func(cmd string) string {
		return "executed: " + cmd
	})
	defer srv.Close()
	go srv.Serve(t)

	client := NewRCONClient(srv.Addr(), "secret")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Command(ctx, "say hello")
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if resp != "executed: say hello" {
		t.Errorf("Command() = %q, want %q", resp, "executed: say hello")
	}
}

func TestRCONClient_AuthFailure(t *testing.T) {
	srv := newRCONTestServer(t, "correct", nil)
	defer srv.Close()
	go srv.Serve(t)

	client := NewRCONClient(srv.Addr(), "wrong")
	ctx := context.Background()

	err := client.Connect(ctx)
	if err == nil {
		_ = client.Close()
		t.Fatal("Connect() expected auth failure, got nil")
	}
	if got := err.Error(); got != "rcon authentication failed" {
		t.Errorf("Connect() error = %q, want 'rcon authentication failed'", got)
	}
}

func TestRCONClient_ConnectionRefused(t *testing.T) {
	// Use a port that nothing is listening on.
	client := NewRCONClient("127.0.0.1:1", "pass")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		_ = client.Close()
		t.Fatal("Connect() expected dial error, got nil")
	}
}

func TestRCONClient_CommandNotConnected(t *testing.T) {
	client := NewRCONClient("127.0.0.1:1", "pass")
	_, err := client.Command(context.Background(), "list")
	if err == nil {
		t.Fatal("Command() expected error when not connected, got nil")
	}
}

func TestRCONClient_CloseIdempotent(t *testing.T) {
	// Close on a never-connected client should not error.
	client := NewRCONClient("127.0.0.1:1", "pass")
	if err := client.Close(); err != nil {
		t.Fatalf("Close() on unconnected client error = %v", err)
	}
}

func TestRCONClient_MultipleCommands(t *testing.T) {
	srv := newRCONTestServer(t, "pass", func(cmd string) string {
		return "ok:" + cmd
	})
	defer srv.Close()
	go srv.Serve(t)

	client := NewRCONClient(srv.Addr(), "pass")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	for i, cmd := range []string{"list", "say hi", "stop"} {
		resp, err := client.Command(ctx, cmd)
		if err != nil {
			t.Fatalf("Command(%d) error = %v", i, err)
		}
		want := "ok:" + cmd
		if resp != want {
			t.Errorf("Command(%d) = %q, want %q", i, resp, want)
		}
	}
}

func TestRCONClient_ConcurrentCommands(t *testing.T) {
	srv := newRCONTestServer(t, "pass", func(cmd string) string {
		return "resp:" + cmd
	})
	defer srv.Close()
	go srv.Serve(t)

	client := NewRCONClient(srv.Addr(), "pass")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = client.Command(ctx, "cmd")
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: Command() error = %v", i, err)
		}
	}
}

func TestRCONClient_EmptyBody(t *testing.T) {
	srv := newRCONTestServer(t, "pass", func(_ string) string {
		return ""
	})
	defer srv.Close()
	go srv.Serve(t)

	client := NewRCONClient(srv.Addr(), "pass")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Command(ctx, "list")
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if resp != "" {
		t.Errorf("Command() = %q, want empty", resp)
	}
}

func TestRCONClient_ServerClosesConnection(t *testing.T) {
	// Server accepts, authenticates, then immediately closes the connection.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Read auth packet and reply with success, then close.
		id, _, _, err := readTestPacket(conn)
		if err != nil {
			_ = conn.Close()
			return
		}
		writeTestPacket(t, conn, id, packetTypeAuthResponse, "")
		// Close immediately so the next Command fails.
		_ = conn.Close()
	}()

	client := NewRCONClient(ln.Addr().String(), "pass")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	_, err = client.Command(ctx, "list")
	if err == nil {
		t.Fatal("Command() expected error after server closed connection, got nil")
	}
	_ = client.Close()
}

func TestRCONClient_DialCancelledContext(t *testing.T) {
	// A cancelled context should prevent dialing.
	client := NewRCONClient("127.0.0.1:1", "pass")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Connect(ctx)
	if err == nil {
		_ = client.Close()
		t.Fatal("Connect() expected error with cancelled context, got nil")
	}
}
