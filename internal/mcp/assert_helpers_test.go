// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"strings"
	"testing"
)

// checker reports assertion failures about one named subject.
type checker struct {
	t    *testing.T
	what string
}

// check returns a checker that names what in every failure message.
func check(t *testing.T, what string) checker { return checker{t: t, what: what} }

// keysPresent fails unless every key is present in the map.
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c checker) keysPresent(m map[string]any, keys ...string) {
	c.t.Helper()
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			c.t.Fatalf("%s missing %q: %+v", c.what, key, m)
		}
	}
}

// keysAbsent fails if any key is present in the map.
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c checker) keysAbsent(m map[string]any, keys ...string) {
	c.t.Helper()
	for _, key := range keys {
		if _, ok := m[key]; ok {
			c.t.Fatalf("%s must omit %q: %+v", c.what, key, m)
		}
	}
}

// textHas fails unless the text contains every fragment.
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c checker) textHas(text string, fragments ...string) {
	c.t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			c.t.Fatalf("%s lacks %q: %s", c.what, fragment, text)
		}
	}
}

// textLacks fails if the text contains any fragment.
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c checker) textLacks(text string, fragments ...string) {
	c.t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(text, fragment) {
			c.t.Fatalf("%s must not contain %q: %s", c.what, fragment, text)
		}
	}
}

// fieldsEqual fails unless every wanted field of the map is equal.
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c checker) fieldsEqual(m, want map[string]any) {
	c.t.Helper()
	for key, value := range want {
		if m[key] != value {
			c.t.Fatalf("%s: %s = %v, want %v: %+v", c.what, key, m[key], value, m)
		}
	}
}

// fileIs fails unless the file holds exactly want.
// ENGMODEL-LINKS: FU-MCP-SERVER
func (c checker) fileIs(path string, want []byte) {
	c.t.Helper()
	if got, _ := os.ReadFile(path); string(got) != string(want) {
		c.t.Fatal(c.what)
	}
}
