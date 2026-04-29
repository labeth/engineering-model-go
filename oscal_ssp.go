// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type OSCALSSPOptions struct {
	ProfileHref       string
	SystemName        string
	SystemDescription string
}

type OSCALSSPResult struct {
	JSON        string
	Document    OSCALSSPDocument
	Diagnostics []validate.Diagnostic
}

type OSCALSSPDocument struct {
	SystemSecurityPlan oscalSystemSecurityPlan `json:"system-security-plan"`
}

type oscalSystemSecurityPlan struct {
	UUID                  string                     `json:"uuid"`
	Metadata              oscalMetadata              `json:"metadata"`
	ImportProfile         oscalImportProfile         `json:"import-profile"`
	SystemCharacteristics oscalSystemCharacteristics `json:"system-characteristics"`
	SystemImplementation  oscalSystemImplementation  `json:"system-implementation"`
	ControlImplementation oscalControlImplementation `json:"control-implementation"`
}

type oscalMetadata struct {
	Title        string `json:"title"`
	LastModified string `json:"last-modified"`
	Version      string `json:"version"`
	OSCALVersion string `json:"oscal-version"`
}

type oscalImportProfile struct {
	Href string `json:"href"`
}

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

type oscalSystemID struct {
	IdentifierType string `json:"identifier-type"`
	ID             string `json:"id"`
}

type oscalSystemInformation struct {
	InformationTypes []oscalInformationType `json:"information-types"`
}

type oscalInformationType struct {
	UUID                  string                 `json:"uuid"`
	Title                 string                 `json:"title"`
	Description           string                 `json:"description"`
	ConfidentialityImpact oscalInformationImpact `json:"confidentiality-impact"`
	IntegrityImpact       oscalInformationImpact `json:"integrity-impact"`
	AvailabilityImpact    oscalInformationImpact `json:"availability-impact"`
}

type oscalInformationImpact struct {
	Base string `json:"base"`
}

type oscalSecurityImpactLevel struct {
	SecurityObjectiveConfidentiality string `json:"security-objective-confidentiality"`
	SecurityObjectiveIntegrity       string `json:"security-objective-integrity"`
	SecurityObjectiveAvailability    string `json:"security-objective-availability"`
}

type oscalOperationalStatus struct {
	State   string `json:"state"`
	Remarks string `json:"remarks,omitempty"`
}

type oscalAuthorizationBoundary struct {
	Description string `json:"description"`
}

type oscalSystemImplementation struct {
	Users      []oscalUser      `json:"users"`
	Components []oscalComponent `json:"components,omitempty"`
}

type oscalUser struct {
	UUID                 string                     `json:"uuid"`
	Title                string                     `json:"title"`
	Description          string                     `json:"description"`
	RoleIDs              []string                   `json:"role-ids,omitempty"`
	Props                []oscalProperty            `json:"props,omitempty"`
	ShortName            string                     `json:"short-name,omitempty"`
	AuthorizedPrivileges []oscalAuthorizedPrivilege `json:"authorized-privileges,omitempty"`
}

type oscalAuthorizedPrivilege struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type oscalComponent struct {
	UUID        string                 `json:"uuid"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Status      oscalOperationalStatus `json:"status"`
	Props       []oscalProperty        `json:"props,omitempty"`
}

type oscalControlImplementation struct {
	Description             string                        `json:"description"`
	ImplementedRequirements []oscalImplementedRequirement `json:"implemented-requirements"`
}

type oscalImplementedRequirement struct {
	UUID         string             `json:"uuid"`
	ControlID    string             `json:"control-id"`
	ByComponents []oscalByComponent `json:"by-components,omitempty"`
}

type oscalByComponent struct {
	UUID          string          `json:"uuid"`
	ComponentUUID string          `json:"component-uuid"`
	Description   string          `json:"description,omitempty"`
	Props         []oscalProperty `json:"props,omitempty"`
	Remarks       string          `json:"remarks,omitempty"`
}

type oscalProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TRLC-LINKS: REQ-EMG-013
func GenerateOSCALSSPFromFile(architecturePath string, options OSCALSSPOptions) (OSCALSSPResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return OSCALSSPResult{}, err
	}
	return GenerateOSCALSSP(bundle, options)
}

func GenerateOSCALSSP(bundle model.Bundle, options OSCALSSPOptions) (OSCALSSPResult, error) {
	diags := validate.Bundle(bundle)
	if validate.HasErrors(diags) {
		return OSCALSSPResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	profile := strings.TrimSpace(options.ProfileHref)
	if profile == "" {
		profile = "https://raw.githubusercontent.com/usnistgov/OSCAL/main/content/nist.gov/SP800-53/rev5/json/NIST_SP-800-53_rev5_MODERATE-baseline_profile.json"
	}
	systemName := strings.TrimSpace(options.SystemName)
	if systemName == "" {
		systemName = nonEmpty(strings.TrimSpace(bundle.Architecture.Model.Title), strings.TrimSpace(bundle.Architecture.Model.ID))
	}
	systemDesc := strings.TrimSpace(options.SystemDescription)
	if systemDesc == "" {
		systemDesc = strings.TrimSpace(bundle.Architecture.Model.Introduction)
	}

	labelByID := buildSSPLabelIndex(bundle.Architecture.AuthoredArchitecture)
	kindByID := buildSSPKindIndex(bundle.Architecture.AuthoredArchitecture)
	componentByID := map[string]oscalComponent{}
	requirementsByControl := map[string][]oscalByComponent{}

	for _, alloc := range bundle.Architecture.AuthoredArchitecture.ControlAllocations {
		narrative := strings.TrimSpace(alloc.Narrative)
		if narrative == "" {
			narrative = fmt.Sprintf("Implementation allocation for control %s.", strings.TrimSpace(alloc.ControlRef))
		}
		evidenceParts := []string{}
		for _, ev := range alloc.Evidence {
			if p := strings.TrimSpace(ev.Path); p != "" {
				evidenceParts = append(evidenceParts, p)
			}
		}
		evidenceNote := ""
		if len(evidenceParts) > 0 {
			evidenceNote = "Evidence: " + strings.Join(evidenceParts, ", ")
		}

		for _, targetID := range alloc.AppliesTo {
			targetID = strings.TrimSpace(targetID)
			if targetID == "" {
				continue
			}
			if _, ok := componentByID[targetID]; !ok {
				componentByID[targetID] = oscalComponent{
					UUID:        deterministicUUID("component|" + bundle.Architecture.Model.ID + "|" + targetID),
					Type:        componentTypeForKind(kindByID[targetID]),
					Title:       nonEmpty(labelByID[targetID], targetID),
					Description: fmt.Sprintf("Architecture component mapped from %s.", targetID),
					Status:      oscalOperationalStatus{State: "operational"},
					Props:       []oscalProperty{{Name: "architecture-id", Value: targetID}, {Name: "architecture-kind", Value: nonEmpty(kindByID[targetID], "unknown")}},
				}
			}
		}

		for _, cid := range alloc.OSCALControlIDs {
			cid = normalizeOSCALControlID(strings.TrimSpace(cid))
			if cid == "" {
				continue
			}
			for _, targetID := range alloc.AppliesTo {
				targetID = strings.TrimSpace(targetID)
				if targetID == "" {
					continue
				}
				comp := componentByID[targetID]
				props := []oscalProperty{}
				if st := strings.TrimSpace(alloc.Status); st != "" {
					props = append(props, oscalProperty{Name: "implementation-status", Value: st})
				}
				if it := strings.TrimSpace(alloc.ImplementationType); it != "" {
					props = append(props, oscalProperty{Name: "implementation-type", Value: it})
				}
				requirementsByControl[cid] = append(requirementsByControl[cid], oscalByComponent{
					UUID:          deterministicUUID("by-component|" + bundle.Architecture.Model.ID + "|" + cid + "|" + targetID + "|" + strings.TrimSpace(alloc.ID)),
					ComponentUUID: comp.UUID,
					Description:   narrative,
					Props:         props,
					Remarks:       evidenceNote,
				})
			}
		}
	}

	componentIDs := make([]string, 0, len(componentByID))
	for id := range componentByID {
		componentIDs = append(componentIDs, id)
	}
	sort.Strings(componentIDs)
	components := make([]oscalComponent, 0, len(componentIDs))
	for _, id := range componentIDs {
		components = append(components, componentByID[id])
	}

	controlIDs := make([]string, 0, len(requirementsByControl))
	for cid := range requirementsByControl {
		controlIDs = append(controlIDs, cid)
	}
	sort.Strings(controlIDs)
	implemented := make([]oscalImplementedRequirement, 0, len(controlIDs))
	for _, cid := range controlIDs {
		entries := requirementsByControl[cid]
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].ComponentUUID < entries[j].ComponentUUID
		})
		implemented = append(implemented, oscalImplementedRequirement{
			UUID:         deterministicUUID("implemented-requirement|" + bundle.Architecture.Model.ID + "|" + cid),
			ControlID:    cid,
			ByComponents: entries,
		})
	}

	users := []oscalUser{}
	for _, actor := range bundle.Architecture.AuthoredArchitecture.Actors {
		id := strings.TrimSpace(actor.ID)
		if id == "" {
			continue
		}
		users = append(users, oscalUser{
			UUID:        deterministicUUID("user|" + bundle.Architecture.Model.ID + "|" + id),
			Title:       nonEmpty(strings.TrimSpace(actor.Name), id),
			Description: nonEmpty(strings.TrimSpace(actor.Description), "Architecture actor mapped as SSP user."),
			ShortName:   strings.ToLower(strings.ReplaceAll(nonEmpty(strings.TrimSpace(actor.Name), id), " ", "-")),
		})
	}
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].Title < users[j].Title
	})
	if len(users) == 0 {
		users = append(users, oscalUser{
			UUID:        deterministicUUID("user|" + bundle.Architecture.Model.ID + "|default"),
			Title:       "Architecture Owner",
			Description: "Default SSP user derived from architecture metadata.",
			ShortName:   "architecture-owner",
		})
	}

	doc := OSCALSSPDocument{SystemSecurityPlan: oscalSystemSecurityPlan{
		UUID: deterministicUUID("ssp|" + bundle.Architecture.Model.ID),
		Metadata: oscalMetadata{
			Title:        nonEmpty(systemName, "Engineering Model SSP"),
			LastModified: time.Now().UTC().Format(time.RFC3339),
			Version:      "0.1.0",
			OSCALVersion: "1.1.2",
		},
		ImportProfile: oscalImportProfile{Href: profile},
		SystemCharacteristics: oscalSystemCharacteristics{
			SystemIDs:                []oscalSystemID{{IdentifierType: "https://engineering-model-go/system-id", ID: nonEmpty(strings.TrimSpace(bundle.Architecture.Model.ID), "engineering-model-system")}},
			SystemName:               systemName,
			Description:              systemDesc,
			SecuritySensitivityLevel: "moderate",
			SystemInformation: oscalSystemInformation{InformationTypes: []oscalInformationType{{
				UUID:                  deterministicUUID("information-type|" + bundle.Architecture.Model.ID),
				Title:                 "Operational Data",
				Description:           "Application and operational data managed by the engineered system.",
				ConfidentialityImpact: oscalInformationImpact{Base: "fips-199-moderate"},
				IntegrityImpact:       oscalInformationImpact{Base: "fips-199-moderate"},
				AvailabilityImpact:    oscalInformationImpact{Base: "fips-199-moderate"},
			}}},
			SecurityImpactLevel:   oscalSecurityImpactLevel{SecurityObjectiveConfidentiality: "moderate", SecurityObjectiveIntegrity: "moderate", SecurityObjectiveAvailability: "moderate"},
			Status:                oscalOperationalStatus{State: "operational"},
			AuthorizationBoundary: oscalAuthorizationBoundary{Description: "Authorization boundary aligns with authored architecture scope and allocated components."},
		},
		SystemImplementation: oscalSystemImplementation{Users: users, Components: components},
		ControlImplementation: oscalControlImplementation{
			Description:             "Control implementation allocations derived from authored controlAllocations.",
			ImplementedRequirements: implemented,
		},
	}}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return OSCALSSPResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return OSCALSSPResult{JSON: string(b), Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

func deterministicUUID(seed string) string {
	h := sha1.Sum([]byte(seed))
	b := h[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", x[0:8], x[8:12], x[12:16], x[16:20], x[20:32])
}

func buildSSPLabelIndex(a model.AuthoredArchitecture) map[string]string {
	out := map[string]string{}
	for _, x := range a.FunctionalGroups {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.FunctionalUnits {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.Actors {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.AttackVectors {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.ReferencedElements {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.Interfaces {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.DataObjects {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.DeploymentTargets {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.Controls {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.TrustBoundaries {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.States {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	for _, x := range a.Events {
		out[x.ID] = nonEmpty(strings.TrimSpace(x.Name), x.ID)
	}
	return out
}

func buildSSPKindIndex(a model.AuthoredArchitecture) map[string]string {
	out := map[string]string{}
	for _, x := range a.FunctionalGroups {
		out[x.ID] = "functional_group"
	}
	for _, x := range a.FunctionalUnits {
		out[x.ID] = "functional_unit"
	}
	for _, x := range a.Actors {
		out[x.ID] = "actor"
	}
	for _, x := range a.AttackVectors {
		out[x.ID] = "attack_vector"
	}
	for _, x := range a.ReferencedElements {
		out[x.ID] = "referenced_element"
	}
	for _, x := range a.Interfaces {
		out[x.ID] = "interface"
	}
	for _, x := range a.DataObjects {
		out[x.ID] = "data_object"
	}
	for _, x := range a.DeploymentTargets {
		out[x.ID] = "deployment_target"
	}
	for _, x := range a.Controls {
		out[x.ID] = "control"
	}
	for _, x := range a.TrustBoundaries {
		out[x.ID] = "trust_boundary"
	}
	for _, x := range a.States {
		out[x.ID] = "state"
	}
	for _, x := range a.Events {
		out[x.ID] = "event"
	}
	return out
}

func componentTypeForKind(kind string) string {
	k := strings.TrimSpace(kind)
	if k == "" {
		return "service"
	}
	return k
}

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
