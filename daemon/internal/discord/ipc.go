package discord

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

const (
	OpHandshake uint32 = 0
	OpFrame     uint32 = 1
	OpClose     uint32 = 2
	OpPing      uint32 = 3
	OpPong      uint32 = 4
)

// WriteFrame encodes and writes an opcode and payload to w.
func WriteFrame(w io.Writer, opcode uint32, payload []byte) error {
	buf := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(buf[0:4], opcode)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(payload)))
	copy(buf[8:], payload)
	_, err := w.Write(buf)
	return err
}

// ReadFrame reads an 8-byte header and subsequent payload from r.
func ReadFrame(r io.Reader) (uint32, []byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}
	opcode := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return opcode, payload, nil
}

// Dialer defines a function that establishes an IPC connection.
type Dialer func() (io.ReadWriteCloser, error)

// Client manages communication with the Discord IPC daemon.
type Client struct {
	clientID string
	dialer   Dialer
	conn     io.ReadWriteCloser
	mu       sync.Mutex
	nonce    int64
}

func defaultDialer() (io.ReadWriteCloser, error) {
	for i := 0; i < 10; i++ {
		pipePath := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		f, err := os.OpenFile(pipePath, os.O_RDWR, 0)
		if err == nil {
			return f, nil
		}
	}
	return nil, errors.New("discord named pipe not found")
}

// NewClient creates a new Discord IPC client.
func NewClient(clientID string) *Client {
	return NewClientWithDialer(clientID, defaultDialer)
}

// NewClientWithDialer creates a new Discord IPC client using a custom dialer.
func NewClientWithDialer(clientID string, dialer Dialer) *Client {
	return &Client{
		clientID: clientID,
		dialer:   dialer,
	}
}

// Connect dials the Discord IPC named pipe and performs the initial handshake.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	conn, err := c.dialer()
	if err != nil {
		return fmt.Errorf("dialing discord ipc: %w", err)
	}

	handshakePayload, _ := json.Marshal(map[string]interface{}{
		"v":         1,
		"client_id": c.clientID,
	})

	if err := WriteFrame(conn, OpHandshake, handshakePayload); err != nil {
		_ = conn.Close()
		return fmt.Errorf("sending handshake: %w", err)
	}

	op, _, err := ReadFrame(conn)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("reading handshake response: %w", err)
	}
	if op != OpFrame {
		_ = conn.Close()
		return fmt.Errorf("unexpected handshake response opcode: %d", op)
	}

	c.conn = conn
	return nil
}

// SetActivity updates the rich presence activity in Discord.
func (c *Client) SetActivity(activity *Activity) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		conn, err := c.dialer()
		if err != nil {
			return fmt.Errorf("not connected and failed to reconnect: %w", err)
		}
		handshakePayload, _ := json.Marshal(map[string]interface{}{
			"v":         1,
			"client_id": c.clientID,
		})
		if err := WriteFrame(conn, OpHandshake, handshakePayload); err != nil {
			_ = conn.Close()
			return fmt.Errorf("sending handshake on reconnect: %w", err)
		}
		op, _, err := ReadFrame(conn)
		if err != nil || op != OpFrame {
			_ = conn.Close()
			return fmt.Errorf("handshake failed on reconnect: %w", err)
		}
		c.conn = conn
	}

	nonce := atomic.AddInt64(&c.nonce, 1)

	var actVal interface{}
	if activity != nil {
		actVal = activity
	}

	payload, err := json.Marshal(map[string]interface{}{
		"cmd": "SET_ACTIVITY",
		"args": map[string]interface{}{
			"pid":      os.Getpid(),
			"activity": actVal,
		},
		"nonce": fmt.Sprintf("%d", nonce),
	})
	if err != nil {
		return fmt.Errorf("marshaling set_activity: %w", err)
	}

	if err := WriteFrame(c.conn, OpFrame, payload); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("writing set_activity frame: %w", err)
	}

	_, _, err = ReadFrame(c.conn)
	if err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("reading set_activity response: %w", err)
	}

	return nil
}

// ClearActivity clears the current Discord presence.
func (c *Client) ClearActivity() error {
	return c.SetActivity(nil)
}

// Close disconnects and releases IPC resources.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}


// IsConnected reports whether an active connection to Discord IPC exists.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}
