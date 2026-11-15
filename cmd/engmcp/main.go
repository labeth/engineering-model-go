// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/cliflags"
	"github.com/labeth/engineering-model-go/internal/mcp"
)

// ENGMODEL-LINKS: IF-CLI-ENGMCP, FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
const maxMessageBytes = 8 * 1024 * 1024

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGMCP, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, FU-MCP-SERVER, EVT-MCP-TOOL-CALL-RECEIVED, FLOW-MCP-TOOL-CALL
func main() {
	modelEnv := cliflags.Register(nil)
	flag.Parse()
	if flag.NArg() != 0 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: engmcp [--workspace off|<engmod.work.yml>] [--lock check|write]")
		os.Exit(2)
	}
	modelEnv.MustApply()
	mcp.ApplyReadOnlyResolutionDefaults()
	s := mcp.NewServer()
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for serveOne(s, r, w) {
	}
}

// serveOne reads one message, handles it and writes the response. It
// reports whether to go on: false at the end of input or after a read or
// write failure; a handler error is logged and the next message is read.
// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGMCP, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, FU-MCP-SERVER, EVT-MCP-TOOL-CALL-RECEIVED, FLOW-MCP-TOOL-CALL
func serveOne(s *mcp.Server, r *bufio.Reader, w *bufio.Writer) bool {
	msg, err := readMessage(r)
	if err != nil {
		if err != io.EOF {
			_, _ = fmt.Fprintf(os.Stderr, "read mcp message: %v\n", err)
		}
		return false
	}
	resp, err := s.Handle(msg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "handle mcp message: %v\n", err)
		return true
	}
	if len(resp) == 0 {
		return true
	}
	if err := writeMessage(w, resp); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write mcp message: %v\n", err)
		return false
	}
	return true
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
// ENGMODEL-LINKS: IF-CLI-ENGMCP, FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func readMessage(r *bufio.Reader) ([]byte, error) {
	headers, err := readHeaders(r)
	if err != nil {
		return nil, err
	}
	n, err := contentLength(headers)
	if err != nil {
		return nil, err
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
// ENGMODEL-LINKS: IF-CLI-ENGMCP, FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func readHeaders(r *bufio.Reader) (map[string]string, error) {
	headers := map[string]string{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return headers, nil
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
		}
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
// ENGMODEL-LINKS: IF-CLI-ENGMCP, FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func contentLength(headers map[string]string) (int, error) {
	cl := headers["content-length"]
	if cl == "" {
		return 0, fmt.Errorf("missing Content-Length header")
	}
	n, err := strconv.Atoi(cl)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid Content-Length %q", cl)
	}
	if n > maxMessageBytes {
		return 0, fmt.Errorf("Content-Length exceeds max allowed size (%d)", maxMessageBytes)
	}
	return n, nil
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
// ENGMODEL-LINKS: IF-CLI-ENGMCP, FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func writeMessage(w *bufio.Writer, body []byte) error {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Content-Length: %d\r\n\r\n", len(body))
	_, _ = b.Write(body)
	if _, err := w.Write(b.Bytes()); err != nil {
		return err
	}
	return w.Flush()
}
