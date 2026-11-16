// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// remedyInterfaceData is the remedy of architecture.interface_without_data.
const remedyInterfaceData = "name the data type sent over the interface: declare the data object (DO-*) in architecture dataObjects and list it in dataRefs, dataIn or dataOut of a flow step whose interfaceRef is the interface (run the ef-behavior skill (engineering-flow) to model the flow); an interface nothing is sent over is removed"

// interfaceDataDiagnostics reports each local software interface that has no
// data object sent over it, as warnings that the tier policy grades (a
// warning up to mvp, an error at product and operational). The rule
// (ADR-EMG-086), per entry of architecture interfaces:
//
//   - an interface carries data when at least one flow step whose interfaceRef
//     is the interface lists at least one data object in the step's dataRefs,
//     dataIn or dataOut; the flow's own dataRefs, relationships (publishes,
//     reads and the others name a data object but no interface), the schemaRef
//     path and names or prose do not count;
//   - an interface that no dependency, flow or relationship uses is asked all
//     the same: the rule is on the interface;
//   - a data object counts when it is a data_object of the model or, in a
//     bundle that is not enriched, an alias::ID of a composed dependency;
//   - a composed alias::IF-* interface is checked in the model that declares
//     it and is not asked here; a hardware interface joins two hardware items
//     and is not a software interface.
//
// Nothing depends on names or prose: the result is a function of the ends of
// the flow steps and the kinds of the model, in model order.
// TRLC-LINKS: REQ-EMG-208
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, EVT-MODEL-CHECK-REQUESTED, FLOW-MODEL-CHECK
func interfaceDataDiagnostics(a model.AuthoredArchitecture, kindByID map[string]string) []Diagnostic {
	carried := interfacesWithData(a, kindByID)
	var diags []Diagnostic
	for i, x := range a.Interfaces {
		id := strings.TrimSpace(x.ID)
		if id == "" || strings.Contains(id, "::") || carried[id] {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: "architecture.interface_without_data", Severity: SeverityWarning, Remedy: remedyInterfaceData,
			Path:    fmt.Sprintf("authoredArchitecture.interfaces[%d]", i),
			Message: fmt.Sprintf("interface %s has no data object sent over it: no flow step with interfaceRef %s lists a data object in dataRefs, dataIn or dataOut; name the data type sent over the interface", id, id),
		})
	}
	return diags
}

// interfacesWithData returns the interfaces that at least one flow step names
// in interfaceRef together with at least one data object.
// TRLC-LINKS: REQ-EMG-208
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func interfacesWithData(a model.AuthoredArchitecture, kindByID map[string]string) map[string]bool {
	carried := map[string]bool{}
	for _, s := range flowSteps(a) {
		if ref := strings.TrimSpace(s.InterfaceRef); ref != "" && stepNamesData(s, kindByID) {
			carried[ref] = true
		}
	}
	return carried
}

// flowSteps lists the steps of every flow in model order.
// TRLC-LINKS: REQ-EMG-208
func flowSteps(a model.AuthoredArchitecture) []model.FlowStep {
	var steps []model.FlowStep
	for _, f := range a.Flows {
		steps = append(steps, f.Steps...)
	}
	return steps
}

// stepNamesData reports whether a flow step lists a data object in dataRefs,
// dataIn or dataOut.
// TRLC-LINKS: REQ-EMG-208
func stepNamesData(s model.FlowStep, kindByID map[string]string) bool {
	for _, d := range slices.Concat(s.DataRefs, s.DataIn, s.DataOut) {
		if isDataObjectRef(strings.TrimSpace(d), kindByID) {
			return true
		}
	}
	return false
}

// isDataObjectRef reports whether an ID is a data object of the model, or an
// alias::ID that the bundle does not resolve (a composed data object of a
// bundle that is not enriched).
// TRLC-LINKS: REQ-EMG-208
func isDataObjectRef(id string, kindByID map[string]string) bool {
	kind, known := kindByID[id]
	return kind == "data_object" || (!known && strings.Contains(id, "::"))
}
