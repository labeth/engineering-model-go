// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// nafValidator validates a NAF profile against the model's actors and views.
type nafValidator struct {
	diagSink
	profile model.NAFProfile
	actors  map[string]bool
	views   map[string]bool
	// stakeholders and concerns are the ids declared so far.
	stakeholders map[string]bool
	concerns     map[string]bool
}

// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func validateNAFProfile(architecture model.ArchitectureDocument, actors map[string]bool) []Diagnostic {
	profile := architecture.NAF
	if !profile.Enabled() {
		return nil
	}
	n := &nafValidator{profile: profile, actors: actors, views: map[string]bool{}}
	for _, architectureView := range architecture.Views {
		n.views[strings.TrimSpace(architectureView.ID)] = true
	}
	n.checkHeader()
	n.checkStakeholders()
	n.checkConcerns()
	n.checkProducts()
	return n.diags
}

// checkHeader requires the NAF framework, version 4.1 and an architecture description.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) checkHeader() {
	if strings.TrimSpace(n.profile.Framework) != "NAF" {
		n.fail("naf.unsupported_framework", "naf.framework", fmt.Sprintf("unsupported architecture framework %q", n.profile.Framework))
	}
	if strings.TrimSpace(n.profile.Version) != "4.1" {
		n.fail("naf.unsupported_version", "naf.version", fmt.Sprintf("unsupported NAF version %q", n.profile.Version))
	}
	if strings.TrimSpace(n.profile.ArchitectureDescription) == "" {
		n.fail("naf.missing_architecture_description", "naf.architectureDescription", "NAF architectureDescription is required")
	}
}

// checkStakeholders requires each stakeholder to be a distinct actor with a role;
// the declared actors are kept in stakeholders.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) checkStakeholders() {
	stakeholders := map[string]bool{}
	n.stakeholders = stakeholders
	for i, stakeholder := range n.profile.Stakeholders {
		path := indexedPath("naf.stakeholders", i)
		actorRef := strings.TrimSpace(stakeholder.ActorRef)
		if actorRef == "" || !n.actors[actorRef] {
			n.fail("naf.invalid_stakeholder_actor_ref", path+".actorRef", fmt.Sprintf("NAF stakeholder references unknown actor %q", actorRef))
		} else if stakeholders[actorRef] {
			n.fail("naf.duplicate_stakeholder", path+".actorRef", fmt.Sprintf("NAF stakeholder actor %q is declared more than once", actorRef))
		}
		stakeholders[actorRef] = true
		n.requireRole(path, stakeholder)
	}
	if len(n.profile.Stakeholders) == 0 {
		n.fail("naf.empty_stakeholders", "naf.stakeholders", "NAF profile must declare at least one stakeholder")
	}
}

// requireRole requires a stakeholder role.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) requireRole(path string, stakeholder model.NAFStakeholder) {
	if strings.TrimSpace(stakeholder.Role) == "" {
		n.fail("naf.missing_stakeholder_role", path+".role", "NAF stakeholder role is required")
	}
}

// knownRefs records the error for each trimmed reference the known set lacks;
// a blank reference is looked up like any other.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) knownRefs(path string, refs []string, spec refSpec) {
	for j, raw := range refs {
		ref := strings.TrimSpace(raw)
		if !spec.ok(ref) {
			n.fail(spec.code, indexedPath(path+"."+spec.field, j), fmt.Sprintf(spec.format, leadArgs(spec.lead, ref)...))
		}
	}
}

// checkConcerns requires each concern to have a unique id, a name and declared stakeholders;
// the concern ids are kept in concerns.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) checkConcerns() {
	concerns := map[string]bool{}
	n.concerns = concerns
	for i, concern := range n.profile.Concerns {
		path := indexedPath("naf.concerns", i)
		id := strings.TrimSpace(concern.ID)
		if id == "" {
			n.fail("naf.missing_concern_id", path+".id", "NAF concern id is required")
		} else if concerns[id] {
			n.fail("naf.duplicate_concern", path+".id", fmt.Sprintf("duplicate NAF concern %q", id))
		}
		concerns[id] = true
		n.checkConcern(path, id, concern)
	}
	if len(n.profile.Concerns) == 0 {
		n.fail("naf.empty_concerns", "naf.concerns", "NAF profile must declare at least one concern")
	}
}

// checkConcern requires a concern name and declared stakeholders.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) checkConcern(path, id string, concern model.NAFConcern) {
	if strings.TrimSpace(concern.Name) == "" {
		n.fail("naf.missing_concern_name", path+".name", "NAF concern name is required")
	}
	if len(concern.StakeholderRefs) == 0 {
		n.fail("naf.empty_concern_stakeholders", path+".stakeholderRefs", fmt.Sprintf("NAF concern %q must reference at least one stakeholder", id))
	}
	n.knownRefs(path, concern.StakeholderRefs, refSpec{ok: inSet(n.stakeholders), code: "naf.invalid_concern_stakeholder_ref", format: "NAF concern %q references undeclared stakeholder %q", field: "stakeholderRefs", lead: []any{id}})
}

// checkProducts requires each product to have a unique id, a known viewpoint and view and declared concerns.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) checkProducts() {
	products := map[string]bool{}
	for i, product := range n.profile.Products {
		path := indexedPath("naf.products", i)
		id := strings.TrimSpace(product.ID)
		if id == "" {
			n.fail("naf.missing_product_id", path+".id", "NAF product id is required")
		} else if products[id] {
			n.fail("naf.duplicate_product", path+".id", fmt.Sprintf("duplicate NAF product %q", id))
		}
		products[id] = true
		n.checkProduct(path, id, product)
	}
	if len(n.profile.Products) == 0 {
		n.fail("naf.empty_products", "naf.products", "NAF profile must declare at least one architecture product")
	}
}

// checkProduct requires a product's viewpoint, view and concerns to exist.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
func (n *nafValidator) checkProduct(path, id string, product model.NAFProduct) {
	if _, ok := model.NAFV41ViewpointTitle(product.Viewpoint); !ok {
		n.fail("naf.unknown_viewpoint", path+".viewpoint", fmt.Sprintf("unknown NAF v4.1 viewpoint %q", product.Viewpoint))
	}
	if viewRef := strings.TrimSpace(product.ViewRef); !n.views[viewRef] {
		n.fail("naf.invalid_view_ref", path+".viewRef", fmt.Sprintf("NAF product %q references unknown Engmod view %q", id, viewRef))
	}
	if len(product.ConcernRefs) == 0 {
		n.fail("naf.empty_product_concerns", path+".concernRefs", fmt.Sprintf("NAF product %q must address at least one concern", id))
	}
	n.knownRefs(path, product.ConcernRefs, refSpec{ok: inSet(n.concerns), code: "naf.invalid_product_concern_ref", format: "NAF product %q references unknown concern %q", field: "concernRefs", lead: []any{id}})
}
