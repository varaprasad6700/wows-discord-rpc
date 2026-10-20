package discord_test

import (
	"encoding/json"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/wows-discord-rpc/daemon/internal/discord"
)

func TestFrame_EncodeDecode(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	payload := []byte(`{"test": "data", "num": 123}`)

	go func() {
		err := discord.WriteFrame(clientConn, discord.OpHandshake, payload)
		if err != nil {
			t.Errorf("WriteFrame error: %v", err)
		}
	}()

	op, data, err := discord.ReadFrame(serverConn)
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}

	if op != discord.OpHandshake {
		t.Errorf("expected opcode %d, got %d", discord.OpHandshake, op)
	}

	if string(data) != string(payload) {
		t.Errorf("expected payload '%s', got '%s'", string(payload), string(data))
	}
}

func TestIPCClient_ConnectAndSetActivity(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	// Mock Discord server
	go func() {
		defer wg.Done()

		// 1. Expect Handshake
		op, data, err := discord.ReadFrame(serverConn)
		if err != nil {
			t.Errorf("mock server read handshake error: %v", err)
			return
		}
		if op != discord.OpHandshake {
			t.Errorf("mock server expected handshake opcode 0, got %d", op)
		}
		var handshake map[string]interface{}
		json.Unmarshal(data, &handshake)
		if handshake["client_id"] != "123456789" {
			t.Errorf("expected client_id 123456789, got %v", handshake["client_id"])
		}

		// Respond to handshake
		readyResp := []byte(`{"cmd":"DISPATCH","evt":"READY"}`)
		_ = discord.WriteFrame(serverConn, discord.OpFrame, readyResp)

		// 2. Expect SetActivity
		op, data, err = discord.ReadFrame(serverConn)
		if err != nil {
			t.Errorf("mock server read set_activity error: %v", err)
			return
		}
		if op != discord.OpFrame {
			t.Errorf("mock server expected frame opcode 1, got %d", op)
		}

		var cmd map[string]interface{}
		json.Unmarshal(data, &cmd)
		if cmd["cmd"] != "SET_ACTIVITY" {
			t.Errorf("expected cmd SET_ACTIVITY, got %v", cmd["cmd"])
		}

		args := cmd["args"].(map[string]interface{})
		act := args["activity"].(map[string]interface{})
		if act["details"] != "Test Details" {
			t.Errorf("expected details 'Test Details', got %v", act["details"])
		}

		// Respond to SetActivity
		ackResp := []byte(`{"cmd":"SET_ACTIVITY","data":{}}`)
		_ = discord.WriteFrame(serverConn, discord.OpFrame, ackResp)

		// 3. Expect ClearActivity
		op, data, err = discord.ReadFrame(serverConn)
		if err != nil {
			t.Errorf("mock server read clear_activity error: %v", err)
			return
		}
		json.Unmarshal(data, &cmd)
		args = cmd["args"].(map[string]interface{})
		if args["activity"] != nil {
			t.Errorf("expected nil activity for clear, got %v", args["activity"])
		}

		_ = discord.WriteFrame(serverConn, discord.OpFrame, ackResp)
	}()

	client := discord.NewClientWithDialer("123456789", func() (io.ReadWriteCloser, error) {
		return clientConn, nil
	})

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	activity := &discord.Activity{
		Details: "Test Details",
		State:   "Test State",
	}

	err = client.SetActivity(activity)
	if err != nil {
		t.Fatalf("SetActivity failed: %v", err)
	}

	err = client.ClearActivity()
	if err != nil {
		t.Fatalf("ClearActivity failed: %v", err)
	}

	client.Close()
	wg.Wait()
}
