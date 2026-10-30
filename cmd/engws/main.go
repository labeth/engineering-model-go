// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/labeth/engineering-model-go/internal/engws"
)

// main runs engws; an interrupt cancels the context, so engws stage --serve
// shuts its registry down cleanly.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := engws.Main(ctx, os.Args[1:], os.Stdout, os.Stderr, engws.NewCUERegistry())
	stop()
	os.Exit(code)
}
