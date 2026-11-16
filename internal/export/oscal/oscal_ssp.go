// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package oscal

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/export/exportutil"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type SSPOptions struct {
	ProfileHref       string
	ImportProfileHref string
	CatalogHref       string
	SystemName        string
	SystemDescription string
	LastModified      string
}

var ErrNoCompliance = errors.New("OSCAL export requires at least one authored compliance mapping")

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type SSPResult struct {
	JSON        string
	Document    SSPDocument
	Diagnostics []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type SSPDocument struct {
	SystemSecurityPlan oscalSystemSecurityPlan `json:"system-security-plan"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalSystemSecurityPlan struct {
	UUID                  string                     `json:"uuid"`
	Metadata              oscalMetadata              `json:"metadata"`
	ImportProfile         oscalImportProfile         `json:"import-profile"`
	SystemCharacteristics oscalSystemCharacteristics `json:"system-characteristics"`
	SystemImplementation  oscalSystemImplementation  `json:"system-implementation"`
	ControlImplementation oscalControlImplementation `json:"control-implementation"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalMetadata struct {
	Title        string `json:"title"`
	LastModified string `json:"last-modified"`
	Version      string `json:"version"`
	OSCALVersion string `json:"oscal-version"`
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-050
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func resolveOSCALTimestamp(bundle model.Bundle, value string) (string, error) {
	return exportutil.ResolveGeneratedTimestamp(bundle, value, "OSCAL last-modified")
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalImportProfile struct {
	Href string `json:"href"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalSystemCharacteristics struct {
	SystemIDs                []oscalSystemID            `json:"system-ids"`
	SystemName               string                     `json:"system-name"`
	Description              string                     `json:"description,omitempty"`
	SecuritySensitivityLevel string                     `json:"security-sensitivity-level"`
	SystemInformation        oscalSystemInformation     `json:"system-information"`
	SecurityImpactLevel      oscalSecurityImpactLevel   `json:"security-impact-level"`
	Status                   oscalOperationalStatus     `json:"status"`
	AuthorizationBoundary    oscalAuthorizationBoundary `json:"authorization-boundary"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalSystemID struct {
	IdentifierType string `json:"identifier-type"`
	ID             string `json:"id"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalSystemInformation struct {
	InformationTypes []oscalInformationType `json:"information-types"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalInformationType struct {
	UUID                  string                 `json:"uuid"`
	Title                 string                 `json:"title"`
	Description           string                 `json:"description"`
	ConfidentialityImpact oscalInformationImpact `json:"confidentiality-impact"`
	IntegrityImpact       oscalInformationImpact `json:"integrity-impact"`
	AvailabilityImpact    oscalInformationImpact `json:"availability-impact"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalInformationImpact struct {
	Base string `json:"base"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER
type oscalSecurityImpactLevel struct {
	SecurityObjectiveConfidentiality string `json:"security-objective-confidentiality"`
	SecurityObjectiveIntegrity       string `json:"security-objective-integrity"`
	SecurityObjectiveAvailability    string `json:"security-objective-availability"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalOperationalStatus struct {
	State   string `json:"state"`
	Remarks string `json:"remarks,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type oscalAuthorizationBoundary struct {
	Description string `json:"description"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalSystemImplementation struct {
	Users      []oscalUser      `json:"users"`
	Components []oscalComponent `json:"components,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalUser struct {
	UUID                 string                     `json:"uuid"`
	Title                string                     `json:"title"`
	Description          string                     `json:"description"`
	RoleIDs              []string                   `json:"role-ids,omitempty"`
	Props                []oscalProperty            `json:"props,omitempty"`
	ShortName            string                     `json:"short-name,omitempty"`
	AuthorizedPrivileges []oscalAuthorizedPrivilege `json:"authorized-privileges,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalAuthorizedPrivilege struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-THREAT-EXPORTER, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
type oscalComponent struct {
	UUID        string                 `json:"uuid"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Status      oscalOperationalStatus `json:"status"`
	Props       []oscalProperty        `json:"props,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalControlImplementation struct {
	Description             string                        `json:"description"`
	ImplementedRequirements []oscalImplementedRequirement `json:"implemented-requirements"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalImplementedRequirement struct {
	UUID         string             `json:"uuid"`
	ControlID    string             `json:"control-id"`
	ByComponents []oscalByComponent `json:"by-components,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalByComponent struct {
	UUID          string          `json:"uuid"`
	ComponentUUID string          `json:"component-uuid"`
	Description   string          `json:"description,omitempty"`
	Props         []oscalProperty `json:"props,omitempty"`
	Remarks       string          `json:"remarks,omitempty"`
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalProperty struct {
	Name  string `json:"name"`
	NS    string `json:"ns,omitempty"`
	Value string `json:"value"`
}

const engineeringModelOSCALNamespace = "https://github.com/labeth/engineering-model-go/ns/oscal"

// TRLC-LINKS: REQ-EMG-013
func engineeringModelOSCALProperty(name, value string) oscalProperty {
	return oscalProperty{Name: name, NS: engineeringModelOSCALNamespace, Value: value}
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func GenerateSSPFromFile(architecturePath string, options SSPOptions) (SSPResult, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return SSPResult{}, err
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return SSPResult{}, err
	}
	return GenerateSSP(bundle, options)
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-013, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, DO-OSCAL-DOCUMENTS
func GenerateSSP(bundle model.Bundle, options SSPOptions) (SSPResult, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return SSPResult{}, err
	}
	bundle = canonical.Documents()
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return sspValidationFailure(diags)
	}

	compliance, complianceDiags := resolveOSCALCompliance(bundle, oscalResolveOptions{ProfileHref: options.ProfileHref, CatalogHref: options.CatalogHref})
	diags = append(diags, complianceDiags...)
	if validate.HasErrors(diags) {
		return sspValidationFailure(diags)
	}
	if len(compliance.Mappings) == 0 {
		return SSPResult{Diagnostics: validate.SortDiagnostics(diags)}, ErrNoCompliance
	}
	lastModified, err := resolveOSCALTimestamp(bundle, options.LastModified)
	if err != nil {
		return SSPResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	header := sspDocumentHeader{
		Profile:      sspProfileHref(bundle, options),
		SystemName:   sspSystemName(bundle, options),
		Description:  sspSystemDescription(bundle, options),
		LastModified: lastModified,
	}
	doc := SSPDocument{SystemSecurityPlan: buildSystemSecurityPlan(bundle, header, allocateSSPComponents(bundle, compliance))}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return SSPResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return SSPResult{JSON: string(b), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspValidationFailure(diags []validate.Diagnostic) (SSPResult, error) {
	return SSPResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type sspDocumentHeader struct {
	Profile      string
	SystemName   string
	Description  string
	LastModified string
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspProfileHref(bundle model.Bundle, options SSPOptions) string {
	profile := strings.TrimSpace(options.ImportProfileHref)
	if profile == "" {
		profile = strings.TrimSpace(options.ProfileHref)
	}
	if profile == "" && len(bundle.Architecture.Compliance.Profiles) > 0 {
		profile = strings.TrimSpace(bundle.Architecture.Compliance.Profiles[0].Href)
	}
	if profile == "" {
		profile = "https://raw.githubusercontent.com/usnistgov/OSCAL/main/content/nist.gov/SP800-53/rev5/json/NIST_SP-800-53_rev5_MODERATE-baseline_profile.json"
	}
	return profile
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspSystemName(bundle model.Bundle, options SSPOptions) string {
	name := strings.TrimSpace(options.SystemName)
	if name == "" {
		name = textutil.NonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID))
	}
	return name
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspSystemDescription(bundle model.Bundle, options SSPOptions) string {
	desc := strings.TrimSpace(options.SystemDescription)
	if desc == "" {
		desc = strings.TrimSpace(bundle.Architecture.Model.Introduction)
	}
	return desc
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type sspAllocation struct {
	Components  []oscalComponent
	Implemented []oscalImplementedRequirement
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func allocateSSPComponents(bundle model.Bundle, compliance oscalComplianceContext) sspAllocation {
	arch := bundle.Architecture.AuthoredArchitecture
	entries := sspEntries(arch)
	allocator := sspAllocator{
		modelID:               bundle.Architecture.Model.ID,
		labelByID:             sspLabelIndex(entries),
		kindByID:              sspKindIndex(entries),
		componentByID:         map[string]oscalComponent{},
		requirementsByControl: map[string][]oscalByComponent{},
	}
	for _, alloc := range compliance.Mappings {
		allocator.addComponents(alloc)
		allocator.addRequirements(alloc)
	}
	return sspAllocation{Components: allocator.sortedComponents(), Implemented: allocator.implementedRequirements()}
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type sspAllocator struct {
	modelID               string
	labelByID             map[string]string
	kindByID              map[string]string
	componentByID         map[string]oscalComponent
	requirementsByControl map[string][]oscalByComponent
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (a *sspAllocator) addComponents(alloc oscalComplianceMapping) {
	for _, targetID := range trimmedNonEmpty(alloc.AppliesTo) {
		if _, ok := a.componentByID[targetID]; !ok {
			a.componentByID[targetID] = a.newComponent(targetID)
		}
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (a *sspAllocator) newComponent(targetID string) oscalComponent {
	return oscalComponent{
		UUID:        textutil.DeterministicUUID("component|" + a.modelID + "|" + targetID),
		Type:        componentTypeForKind(a.kindByID[targetID]),
		Title:       textutil.NonEmpty(a.labelByID[targetID], targetID),
		Description: fmt.Sprintf("Architecture component mapped from %s.", targetID),
		Status:      oscalOperationalStatus{State: "operational"},
		Props: []oscalProperty{
			engineeringModelOSCALProperty("architecture-id", targetID),
			engineeringModelOSCALProperty("architecture-kind", textutil.NonEmpty(a.kindByID[targetID], "unknown")),
		},
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (a *sspAllocator) addRequirements(alloc oscalComplianceMapping) {
	narrative := strings.TrimSpace(alloc.Narrative)
	if narrative == "" {
		narrative = fmt.Sprintf("Implementation allocation for control %s.", strings.TrimSpace(alloc.ModelControlRef))
	}
	evidenceNote := sspEvidenceNote(alloc)
	props := sspAllocationProps(alloc)
	for _, cid := range alloc.ControlIDs {
		cid = normalizeOSCALControlID(strings.TrimSpace(cid))
		if cid == "" {
			continue
		}
		for _, targetID := range trimmedNonEmpty(alloc.AppliesTo) {
			a.requirementsByControl[cid] = append(a.requirementsByControl[cid], oscalByComponent{
				UUID:          textutil.DeterministicUUID("by-component|" + a.modelID + "|" + cid + "|" + targetID + "|" + strings.TrimSpace(alloc.ID)),
				ComponentUUID: a.componentByID[targetID].UUID,
				Description:   narrative,
				Props:         append([]oscalProperty{}, props...),
				Remarks:       evidenceNote,
			})
		}
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func trimmedNonEmpty(in []string) []string {
	var out []string
	for _, x := range in {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspEvidenceNote(alloc oscalComplianceMapping) string {
	evidenceParts := []string{}
	for _, ev := range alloc.Evidence {
		if p := strings.TrimSpace(ev.Path); p != "" {
			evidenceParts = append(evidenceParts, p)
		}
	}
	if len(evidenceParts) == 0 {
		return ""
	}
	return "Evidence: " + strings.Join(evidenceParts, ", ")
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspAllocationProps(alloc oscalComplianceMapping) []oscalProperty {
	props := []oscalProperty{}
	for _, p := range []struct{ name, value string }{
		{"implementation-status", alloc.ImplementationStatus},
		{"implementation-type", alloc.ImplementationType},
		{"model-control-ref", alloc.ModelControlRef},
		{"profile-ref", alloc.ProfileRef},
		{"mapping-rationale", alloc.Rationale},
	} {
		if v := strings.TrimSpace(p.value); v != "" {
			props = append(props, engineeringModelOSCALProperty(p.name, v))
		}
	}
	return props
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (a *sspAllocator) sortedComponents() []oscalComponent {
	componentIDs := make([]string, 0, len(a.componentByID))
	for id := range a.componentByID {
		componentIDs = append(componentIDs, id)
	}
	sort.Strings(componentIDs)
	components := make([]oscalComponent, 0, len(componentIDs))
	for _, id := range componentIDs {
		components = append(components, a.componentByID[id])
	}
	return components
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (a *sspAllocator) implementedRequirements() []oscalImplementedRequirement {
	controlIDs := make([]string, 0, len(a.requirementsByControl))
	for cid := range a.requirementsByControl {
		controlIDs = append(controlIDs, cid)
	}
	sort.Strings(controlIDs)
	implemented := make([]oscalImplementedRequirement, 0, len(controlIDs))
	for _, cid := range controlIDs {
		entries := a.requirementsByControl[cid]
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].ComponentUUID < entries[j].ComponentUUID
		})
		implemented = append(implemented, oscalImplementedRequirement{
			UUID:         textutil.DeterministicUUID("implemented-requirement|" + a.modelID + "|" + cid),
			ControlID:    cid,
			ByComponents: entries,
		})
	}
	return implemented
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspUsers(bundle model.Bundle) []oscalUser {
	modelID := bundle.Architecture.Model.ID
	users := []oscalUser{}
	for _, actor := range bundle.Architecture.AuthoredArchitecture.Actors {
		id := strings.TrimSpace(actor.ID)
		if id == "" {
			continue
		}
		users = append(users, oscalUser{
			UUID:        textutil.DeterministicUUID("user|" + modelID + "|" + id),
			Title:       textutil.NonEmpty(strings.TrimSpace(actor.Name), id),
			Description: textutil.NonEmpty(strings.TrimSpace(actor.Description), "Architecture actor mapped as SSP user."),
			ShortName:   strings.ToLower(strings.ReplaceAll(textutil.NonEmpty(strings.TrimSpace(actor.Name), id), " ", "-")),
		})
	}
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].Title < users[j].Title
	})
	if len(users) == 0 {
		users = append(users, oscalUser{
			UUID:        textutil.DeterministicUUID("user|" + modelID + "|default"),
			Title:       "Architecture Owner",
			Description: "Default SSP user derived from architecture metadata.",
			ShortName:   "architecture-owner",
		})
	}
	return users
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildSystemSecurityPlan(bundle model.Bundle, header sspDocumentHeader, alloc sspAllocation) oscalSystemSecurityPlan {
	modelID := bundle.Architecture.Model.ID
	return oscalSystemSecurityPlan{
		UUID: textutil.DeterministicUUID("ssp|" + modelID),
		Metadata: oscalMetadata{
			Title:        textutil.NonEmpty(header.SystemName, "Engineering Model SSP"),
			LastModified: header.LastModified,
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportProfile:         oscalImportProfile{Href: header.Profile},
		SystemCharacteristics: sspSystemCharacteristics(bundle, header),
		SystemImplementation:  oscalSystemImplementation{Users: sspUsers(bundle), Components: alloc.Components},
		ControlImplementation: oscalControlImplementation{
			Description:             "Control implementation allocations derived from authored compliance mappings.",
			ImplementedRequirements: alloc.Implemented,
		},
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspSystemCharacteristics(bundle model.Bundle, header sspDocumentHeader) oscalSystemCharacteristics {
	modelID := bundle.Architecture.Model.ID
	return oscalSystemCharacteristics{
		SystemIDs:                []oscalSystemID{{IdentifierType: "https://engineering-model-go/system-id", ID: textutil.NonEmpty(strings.TrimSpace(modelID), "engineering-model-system")}},
		SystemName:               header.SystemName,
		Description:              header.Description,
		SecuritySensitivityLevel: "moderate",
		SystemInformation: oscalSystemInformation{InformationTypes: []oscalInformationType{{
			UUID:                  textutil.DeterministicUUID("information-type|" + modelID),
			Title:                 "Operational Data",
			Description:           "Application and operational data managed by the engineered system.",
			ConfidentialityImpact: oscalInformationImpact{Base: "fips-199-moderate"},
			IntegrityImpact:       oscalInformationImpact{Base: "fips-199-moderate"},
			AvailabilityImpact:    oscalInformationImpact{Base: "fips-199-moderate"},
		}}},
		SecurityImpactLevel:   oscalSecurityImpactLevel{SecurityObjectiveConfidentiality: "moderate", SecurityObjectiveIntegrity: "moderate", SecurityObjectiveAvailability: "moderate"},
		Status:                oscalOperationalStatus{State: "operational"},
		AuthorizationBoundary: oscalAuthorizationBoundary{Description: "Authorization boundary aligns with authored architecture scope and allocated components."},
	}
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type sspEntry struct {
	ID    string
	Label string
	Kind  string
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspEntriesOf[T any](kind string, items []T, idName func(T) (string, string)) []sspEntry {
	out := make([]sspEntry, 0, len(items))
	for _, x := range items {
		id, name := idName(x)
		out = append(out, sspEntry{ID: id, Label: textutil.NonEmpty(strings.TrimSpace(name), id), Kind: kind})
	}
	return out
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspEntries(a model.AuthoredArchitecture) []sspEntry {
	groups := [][]sspEntry{
		sspEntriesOf("functional_group", a.FunctionalGroups, func(x model.FunctionalGroup) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("functional_unit", a.FunctionalUnits, func(x model.FunctionalUnit) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("actor", a.Actors, func(x model.Actor) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("attack_vector", a.AttackVectors, func(x model.AttackVector) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("referenced_element", a.ReferencedElements, func(x model.ReferencedElement) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("interface", a.Interfaces, func(x model.Interface) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("data_object", a.DataObjects, func(x model.DataObject) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("deployment_target", a.DeploymentTargets, func(x model.DeploymentTarget) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("control", a.Controls, func(x model.Control) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("trust_boundary", a.TrustBoundaries, func(x model.TrustBoundary) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("machine", a.Machines, func(x model.Machine) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("state", a.MachineStates(), func(x model.MachineState) (string, string) { return x.ID, x.Name }),
		sspEntriesOf("event", a.Events, func(x model.Event) (string, string) { return x.ID, x.Name }),
	}
	var out []sspEntry
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspLabelIndex(entries []sspEntry) map[string]string {
	out := map[string]string{}
	for _, e := range entries {
		out[e.ID] = e.Label
	}
	return out
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sspKindIndex(entries []sspEntry) map[string]string {
	out := map[string]string{}
	for _, e := range entries {
		out[e.ID] = e.Kind
	}
	return out
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func componentTypeForKind(kind string) string {
	k := strings.TrimSpace(kind)
	if k == "" {
		return "service"
	}
	return k
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func normalizeOSCALControlID(raw string) string {
	x := strings.ToLower(strings.TrimSpace(raw))
	if x == "" {
		return ""
	}
	for {
		open := strings.Index(x, "(")
		close := strings.Index(x, ")")
		if open < 0 || close < 0 || close <= open {
			break
		}
		inside := strings.TrimSpace(x[open+1 : close])
		x = x[:open] + "." + inside + x[close+1:]
	}
	x = strings.ReplaceAll(x, " ", "")
	return x
}
