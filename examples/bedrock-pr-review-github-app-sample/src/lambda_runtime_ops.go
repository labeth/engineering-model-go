// ENGMODEL-OWNER-UNIT: FU-LAMBDA-RUNTIME-OPERATIONS
// ENGMODEL-CODE-DESCRIPTION: wires Lambda runtime entrypoints and shared execution dependencies
package sample

import "log"

type LambdaRuntimeOps struct{}

// TRLC-LINKS: REQ-PRR-001, REQ-PRR-003, REQ-PRR-005, REQ-PRR-006, REQ-PRR-007, REQ-PRR-008
func (o *LambdaRuntimeOps) ApplyRelease(functionName, imageTag string) {
	log.Printf("lambda-runtime-ops: deploy function=%s image=%s", functionName, imageTag)
}
