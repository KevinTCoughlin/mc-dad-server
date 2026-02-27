package container

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Source RCON protocol packet types.
const (
	packetTypeAuth         int32 = 3
	packetTypeAuthResponse int32 = 2
	packetTypeCommand      int32 = 2
	packetTypeResponse     int32 = 0
)

// RCONClient implements the Source RCON protocol for communicating with a
// Minecraft server. It is safe for concurrent use after Connect returns.
type RCONClient struct {
	addr     string
	password string
	conn     net.Conn
	mu       sync.Mutex
	reqID    atomic.Int32
}

// NewRCONClient creates a new RCON client for the given address and password.
func NewRCONClient(addr, password string) *RCONClient {
	return &RCONClient{
		addr:     addr,
		password: password,
	}
}

// Connect dials the RCON server and authenticates.
func (r *RCONClient) Connect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", r.addr)
	if err != nil {
		return fmt.Errorf("rcon dial: %w", err)
	}
	r.conn = conn

	// Apply a deadline for the auth handshake so reads don't block
	// indefinitely if the server accepts the connection but never responds.
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	}
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	// Authenticate.
	id := r.nextID()
	if err := r.writePacket(id, packetTypeAuth, r.password); err != nil {
		_ = conn.Close()
		r.conn = nil
		return fmt.Errorf("rcon auth write: %w", err)
	}

	respID, respType, _, err := r.readPacket()
	if err != nil {
		_ = conn.Close()
		r.conn = nil
		return fmt.Errorf("rcon auth read: %w", err)
	}

	// Minecraft sends an auth response with the request ID on success,
	// or -1 on failure.
	if respType == packetTypeAuthResponse && respID == -1 {
		_ = conn.Close()
		r.conn = nil
		return fmt.Errorf("rcon authentication failed")
	}

	// Some servers send an empty response before the auth response.
	if respType != packetTypeAuthResponse {
		respID, respType, _, err = r.readPacket()
		if err != nil {
			_ = conn.Close()
			r.conn = nil
			return fmt.Errorf("rcon auth read (2nd): %w", err)
		}
		if respType == packetTypeAuthResponse && respID == -1 {
			_ = conn.Close()
			r.conn = nil
			return fmt.Errorf("rcon authentication failed")
		}
	}

	return nil
}

// Command sends a command and returns the response body.
func (r *RCONClient) Command(ctx context.Context, cmd string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil {
		return "", fmt.Errorf("rcon not connected")
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = r.conn.SetDeadline(deadline)
		defer func() { _ = r.conn.SetDeadline(time.Time{}) }()
	}

	id := r.nextID()
	if err := r.writePacket(id, packetTypeCommand, cmd); err != nil {
		return "", fmt.Errorf("rcon command write: %w", err)
	}

	respID, _, body, err := r.readPacket()
	if err != nil {
		return "", fmt.Errorf("rcon command read: %w", err)
	}
	if respID != id {
		return body, fmt.Errorf("rcon response ID mismatch: got %d, want %d", respID, id)
	}

	return body, nil
}

// Close closes the underlying connection.
func (r *RCONClient) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil {
		return nil
	}

	err := r.conn.Close()
	r.conn = nil
	return err
}

func (r *RCONClient) nextID() int32 {
	return r.reqID.Add(1)
}

func (r *RCONClient) writePacket(id, pktType int32, body string) error {
	bodyBytes := []byte(body)
	// Packet layout: 4 (size) + 4 (id) + 4 (type) + body + 2 (null terminators)
	size := int32(4 + 4 + len(bodyBytes) + 2)
	buf := make([]byte, 4+size)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(id))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(pktType))
	copy(buf[12:], bodyBytes)
	buf[12+len(bodyBytes)] = 0
	buf[13+len(bodyBytes)] = 0

	_, err := r.conn.Write(buf)
	return err
}

func (r *RCONClient) readPacket() (id, pktType int32, body string, err error) {
	// Read the 4-byte size prefix.
	var sizeBuf [4]byte
	if _, err := io.ReadFull(r.conn, sizeBuf[:]); err != nil {
		return 0, 0, "", err
	}
	size := int32(binary.LittleEndian.Uint32(sizeBuf[:]))
	if size < 10 || size > 4096+10 {
		return 0, 0, "", fmt.Errorf("rcon packet size out of range: %d", size)
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r.conn, payload); err != nil {
		return 0, 0, "", err
	}

	id = int32(binary.LittleEndian.Uint32(payload[0:4]))
	pktType = int32(binary.LittleEndian.Uint32(payload[4:8]))
	// Body is everything between the type field and the two null terminators.
	bodyLen := size - 10
	if bodyLen > 0 {
		body = string(payload[8 : 8+bodyLen])
	}

	return id, pktType, body, nil
}
