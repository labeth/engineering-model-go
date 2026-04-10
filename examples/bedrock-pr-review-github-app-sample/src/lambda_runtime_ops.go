// ENGMODEL-OWNER-UNIT: FU-LAMBDA-RUNTIME-OPERATIONS
// ENGMODEL-CODE-DESCRIPTION: wires Lambda runtime entrypoints and shared execution dependencies
package sample

import "log"

type LambdaRuntimeOps struct{}

func (o *LambdaRuntimeOps) ApplyRelease(functionName, imageTag string) {
	log.Printf("lambda-runtime-ops: deploy function=%s image=%s", functionName, imageTag)
}
