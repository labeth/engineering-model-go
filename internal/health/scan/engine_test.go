// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/health/engine/treesitter"
)

// realEngine is the engine Scan uses by default.
func realEngine() health.Engine { return treesitter.New() }
