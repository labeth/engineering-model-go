// ENGMODEL-OWNER-UNIT: FU-LAMBDA-RUNTIME-OPERATIONS
package sample

import "log"

type LambdaRuntimeOps struct{}

func (o *LambdaRuntimeOps) ApplyRelease(functionName, imageTag string) {
	log.Printf("lambda-runtime-ops: deploy function=%s image=%s", functionName, imageTag)
}
