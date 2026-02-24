package container

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
)

// fakeRCON is a minimal RCON server for testing.
type fakeRCON struct {
	listener net.Listener
	mu       sync.Mutex
	commands []string
	password string
}

func newFakeRCON(t *testing.T, password string) *fakeRCON {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	f := &fakeRCON{listener: ln, password: password}
	go f.serve(t)
	return f
}

func (f *fakeRCON) addr() string { return f.listener.Addr().String() }

func (f *fakeRCON) close() { _ = f.listener.Close() }

func (f *fakeRCON) serve(t *testing.T) {
	t.Helper()
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}
		go f.handleConn(t, conn)
	}
}

func (f *fakeRCON) handleConn(t *testing.T, conn net.Conn) {
	t.Helper()
	defer func() { _ = conn.Close() }()

	for {
		id, pktType, body, err := readTestPacket(conn)
		if err != nil {
			return
		}

		switch pktType {
		case packetTypeAuth:
			respID := id
			if body != f.password {
				respID = -1
			}
			if err := writeTestPacket(conn, respID, packetTypeAuthResponse, ""); err != nil {
				return
			}
		case packetTypeCommand:
			f.mu.Lock()
			f.commands = append(f.commands, body)
			f.mu.Unlock()
			if err := writeTestPacket(conn, id, packetTypeResponse, "OK"); err != nil {
				return
			}
		}
	}
}

func (f *fakeRCON) getCommands() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.commands))
	copy(out, f.commands)
	return out
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

func writeTestPacket(w io.Writer, id, pktType int32, body string) error {
	bodyBytes := []byte(body)
	size := int32(4 + 4 + len(bodyBytes) + 2)
	buf := make([]byte, 4+size)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(id))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(pktType))
	copy(buf[12:], bodyBytes)
	buf[12+len(bodyBytes)] = 0
	buf[13+len(bodyBytes)] = 0
	_, err := w.Write(buf)
	return err
}

func TestRCONClient_ConnectAndCommand(t *testing.T) {
	srv := newFakeRCON(t, "secret")
	defer srv.close()

	client := NewRCONClient(srv.addr(), "secret")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	resp, err := client.Command(ctx, "say hello")
	if err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if resp != "OK" {
		t.Errorf("Command() response = %q, want %q", resp, "OK")
	}

	cmds := srv.getCommands()
	if len(cmds) != 1 || cmds[0] != "say hello" {
		t.Errorf("server received %v, want [say hello]", cmds)
	}
}

func TestRCONClient_AuthFailure(t *testing.T) {
	srv := newFakeRCON(t, "secret")
	defer srv.close()

	client := NewRCONClient(srv.addr(), "wrong-password")
	err := client.Connect(context.Background())
	if err == nil {
		_ = client.Close()
		t.Fatal("Connect() should fail with wrong password")
	}
}

func TestRCONClient_DialFailure(t *testing.T) {
	client := NewRCONClient("127.0.0.1:0", "pass")
	err := client.Connect(context.Background())
	if err == nil {
		_ = client.Close()
		t.Fatal("Connect() should fail for unreachable address")
	}
}

func TestRCONClient_CommandWithoutConnect(t *testing.T) {
	client := NewRCONClient("127.0.0.1:0", "pass")
	_, err := client.Command(context.Background(), "list")
	if err == nil {
		t.Fatal("Command() should fail when not connected")
	}
}

func TestRCONClient_CloseIdempotent(t *testing.T) {
	srv := newFakeRCON(t, "secret")
	defer srv.close()

	client := NewRCONClient(srv.addr(), "secret")
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestRCONClient_MultipleCommands(t *testing.T) {
	srv := newFakeRCON(t, "pass")
	defer srv.close()

	client := NewRCONClient(srv.addr(), "pass")
	ctx := context.Background()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	for i := 0; i < 5; i++ {
		if _, err := client.Command(ctx, "list"); err != nil {
			t.Fatalf("Command() #%d error = %v", i, err)
		}
	}

	cmds := srv.getCommands()
	if len(cmds) != 5 {
		t.Errorf("server received %d commands, want 5", len(cmds))
	}
}
