// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestReadMessageRejectsOversizedPayload(t *testing.T) {
	var input bytes.Buffer
	_, _ = fmt.Fprintf(&input, "Content-Length: %d\r\n\r\n", maxMessageBytes+1)
	reader := bufio.NewReader(bytes.NewReader(input.Bytes()))
	if _, err := readMessage(reader); err == nil {
		t.Fatalf("expected oversized payload error")
	}
}

func TestReadWriteMessageRoundTrip(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	var out bytes.Buffer
	writer := bufio.NewWriter(&out)
	if err := writeMessage(writer, body); err != nil {
		t.Fatalf("write message: %v", err)
	}
	reader := bufio.NewReader(bytes.NewReader(out.Bytes()))
	got, err := readMessage(reader)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("body mismatch: got %s want %s", string(got), string(body))
	}
}
