// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"context"
	"os"

	"github.com/labeth/engineering-model-go/internal/engws"
)

// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func main() {
	os.Exit(engws.Main(context.Background(), os.Args[1:], os.Stdout, os.Stderr, engws.NewCUERegistry()))
}
