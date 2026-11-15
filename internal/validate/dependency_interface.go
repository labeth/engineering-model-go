// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// remedyDependencyInterface is the remedy of architecture.dependency_without_interface.
const remedyDependencyInterface = "name the interface the dependency goes through: declare it in architecture interfaces with its owner (a functional unit, or a referenced element for an external one) and point the relationship at that interface (IF-*, or alias::IF-* of a composed dependency's publication) instead of at the element itself"

// dependencyTypes are the relationship types that make one element depend on
// another (ADR-EMG-085): the architecture communication and use types. The
// structural and allocation types (contains, deployed_to, allocated_to,
// satisfies, verified_by, guarded_by, mitigated_by, bounded_by) and targets,
// which an attack vector makes, say where something is or what it is held to,
// not what it needs, and are never asked.
var dependencyTypes = map[string]bool{
	"depends_on": true, "calls": true, "publishes": true, "subscribes": true, "reads": true,
	"writes": true, "streams": true, "implements": true, "interacts_with": true,
}

// dependencyEndKinds are the kinds of element a dependency is between. A
// dependency end of another kind (a data object a unit reads or writes, a
// deployment target) is data access or placement, not a dependency between
// two architectural elements, and is not asked. A functional group is listed
// although mapping pairs never admit it as an end: a dependency between groups
// is the dependencies of the units inside them.
var dependencyEndKinds = map[string]bool{
	"functional_unit": true, "functional_group": true, "actor": true, "referenced_element": true,
	"interface": true, "hardware_item": true,
}

// interfaceDiagnostics runs the two interface rules of ADR-EMG-085 and
// ADR-EMG-086: a dependency names an interface and an interface carries data.
// TRLC-LINKS: REQ-EMG-207, REQ-EMG-208
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, EVT-MODEL-CHECK-REQUESTED, FLOW-MODEL-CHECK
func interfaceDiagnostics(a model.AuthoredArchitecture, kindByID map[string]string) []Diagnostic {
	return append(dependencyInterfaceDiagnostics(a, kindByID), interfaceDataDiagnostics(a, kindByID)...)
}

// dependencyInterfaceDiagnostics reports each architectural dependency that
// names no interface, as warnings that the tier policy grades (a warning up to
// mvp, an error at product and operational). The rule, per relationship of
// behavior.relationships (authoredArchitecture.mappings):
//
//   - the type is a dependency type (dependencyTypes) and both ends are
//     architectural elements (dependencyEndKinds); an end that is unknown is a
//     broken reference reported elsewhere;
//   - a relationship from an element to itself is not asked;
//   - the dependency has an interface when one of its ends is an interface
//     (a software interface): a local software interface must have an owner
//     (interface.owner, else the functional unit that contains it), a composed
//     alias::IF-* interface is a published one and counts as owned;
//   - a hardware item to hardware item interaction has an interface when a
//     hardware interface connects the two items;
//   - every other dependency (unit to unit, unit or actor to a unit, unit to a
//     referenced element, composed alias::ID unit) names no interface.
//
// Nothing depends on names or prose: the result is a function of the kinds,
// ends and owners of the model, in model order.
// TRLC-LINKS: REQ-EMG-207
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, EVT-MODEL-CHECK-REQUESTED, FLOW-MODEL-CHECK
func dependencyInterfaceDiagnostics(a model.AuthoredArchitecture, kindByID map[string]string) []Diagnostic {
	facts := interfaceFacts{kinds: kindByID, owned: ownedInterfaces(a), connected: connectedHardware(a)}
	var diags []Diagnostic
	for i, m := range a.Mappings {
		if d, ok := facts.dependencyDiagnostic(i, m); ok {
			diags = append(diags, d)
		}
	}
	return diags
}

// interfaceFacts is what the dependency rule reads from the model: the kind
// of each ID, the local interfaces that have an owner and the pairs of
// hardware items that a hardware interface connects.
type interfaceFacts struct {
	kinds     map[string]string
	owned     map[string]bool
	connected map[string]bool
}

// dependencyDiagnostic returns the diagnostic of the relationship at index i
// when it is a dependency between two architectural elements that names no
// interface.
// TRLC-LINKS: REQ-EMG-207
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (f interfaceFacts) dependencyDiagnostic(i int, m model.Mapping) (Diagnostic, bool) {
	typ, from, to := strings.TrimSpace(m.Type), strings.TrimSpace(m.From), strings.TrimSpace(m.To)
	if !dependencyTypes[typ] || from == to {
		return Diagnostic{}, false
	}
	if !dependencyEndKinds[f.kinds[from]] || !dependencyEndKinds[f.kinds[to]] {
		return Diagnostic{}, false
	}
	reason := f.gap(from, to)
	if reason == "" {
		return Diagnostic{}, false
	}
	path := fmt.Sprintf("authoredArchitecture.mappings[%d]", i)
	return Diagnostic{
		Code: "architecture.dependency_without_interface", Severity: SeverityWarning, Path: path, Remedy: remedyDependencyInterface,
		Message: fmt.Sprintf("dependency %q from %s to %s (relationship %s) %s; name the interface the dependency goes through", typ, from, to, path, reason),
	}, true
}

// gap says why a dependency has no interface, or "" when it has one.
// TRLC-LINKS: REQ-EMG-207
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (f interfaceFacts) gap(from, to string) string {
	for _, end := range []string{from, to} {
		if f.kinds[end] == "interface" {
			return f.interfaceEndGap(end)
		}
	}
	if f.kinds[from] == "hardware_item" && f.kinds[to] == "hardware_item" && f.connected[pairKey(from, to)] {
		return ""
	}
	return "names no interface"
}

// interfaceEndGap says why an interface at the end of a dependency does not
// count: it is a local interface that no unit owns.
// TRLC-LINKS: REQ-EMG-207
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (f interfaceFacts) interfaceEndGap(end string) string {
	if f.owned[end] || strings.Contains(end, "::") {
		return ""
	}
	return fmt.Sprintf("goes through interface %s, which has no owner", end)
}

// ownedInterfaces returns the local software interfaces that have an owner: a
// non-empty interface.owner, or a functional unit that contains the interface
// in a contains relationship.
// TRLC-LINKS: REQ-EMG-207
func ownedInterfaces(a model.AuthoredArchitecture) map[string]bool {
	owned := map[string]bool{}
	for _, id := range slices.Concat(interfacesWithOwnerField(a), containedIDs(a)) {
		owned[id] = true
	}
	return owned
}

// interfacesWithOwnerField lists the interfaces whose owner field is set.
// TRLC-LINKS: REQ-EMG-207
func interfacesWithOwnerField(a model.AuthoredArchitecture) []string {
	var ids []string
	for _, x := range a.Interfaces {
		if strings.TrimSpace(x.Owner) != "" {
			ids = append(ids, strings.TrimSpace(x.ID))
		}
	}
	return ids
}

// containedIDs lists the targets of the contains relationships.
// TRLC-LINKS: REQ-EMG-207
func containedIDs(a model.AuthoredArchitecture) []string {
	var ids []string
	for _, m := range a.Mappings {
		if strings.TrimSpace(m.Type) == "contains" {
			ids = append(ids, strings.TrimSpace(m.To))
		}
	}
	return ids
}

// connectedHardware returns the pairs of hardware items that a hardware
// interface connects, in either direction.
// TRLC-LINKS: REQ-EMG-207
func connectedHardware(a model.AuthoredArchitecture) map[string]bool {
	pairs := map[string]bool{}
	for _, x := range a.HardwareInterfaces {
		pairs[pairKey(strings.TrimSpace(x.From), strings.TrimSpace(x.To))] = true
	}
	return pairs
}

// pairKey names an unordered pair of IDs.
// TRLC-LINKS: REQ-EMG-207
func pairKey(a, b string) string {
	if b < a {
		a, b = b, a
	}
	return a + "\x00" + b
}
