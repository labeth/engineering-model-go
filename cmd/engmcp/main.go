// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/mcp"
)

const maxMessageBytes = 8 * 1024 * 1024

func main() {
	s := mcp.NewServer()
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for {
		msg, err := readMessage(r)
		if err != nil {
			if err == io.EOF {
				return
			}
			_, _ = fmt.Fprintf(os.Stderr, "read mcp message: %v\n", err)
			return
		}
		resp, err := s.Handle(msg)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "handle mcp message: %v\n", err)
			continue
		}
		if len(resp) == 0 {
			continue
		}
		if err := writeMessage(w, resp); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "write mcp message: %v\n", err)
			return
		}
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func readMessage(r *bufio.Reader) ([]byte, error) {
	headers := map[string]string{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		}
	}
	cl := headers["content-length"]
	if cl == "" {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	n, err := strconv.Atoi(cl)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("invalid Content-Length %q", cl)
	}
	if n > maxMessageBytes {
		return nil, fmt.Errorf("Content-Length exceeds max allowed size (%d)", maxMessageBytes)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeMessage(w *bufio.Writer, body []byte) error {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Content-Length: %d\r\n\r\n", len(body))
	_, _ = b.Write(body)
	if _, err := w.Write(b.Bytes()); err != nil {
		return err
	}
	return w.Flush()
}
