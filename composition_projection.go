// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

// PublishedProjectionEntity is one explicitly published dependency entity,
// retaining its typed value, qualified identity, source bundle, and provenance.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type PublishedProjectionEntity struct {
	QualifiedID string
	SourceID    string
	Domain      string
	Facet       string
	Value       any
	Source      *model.Bundle
	Provenance  ImportedEntityProvenance
}

// PublishedProjection groups the entities selected from one dependency publication.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type PublishedProjection struct {
	Alias       string
	Publication string
	ModulePath  string
	Version     string
	SourceModel string
	Entities    []PublishedProjectionEntity
}

// CompositionProjection is the reusable, read-only projection boundary used by
// every output format. It never mutates an authored bundle.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type CompositionProjection struct {
	Publications []PublishedProjection
	Entities     []PublishedProjectionEntity
}

// BuildCompositionProjection materializes only direct, explicitly selected
// dependency publications. Transitive or unexported child data is not exposed.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func BuildCompositionProjection(result CompositionResult) CompositionProjection {
	var projection CompositionProjection
	if result.Root == nil {
		return projection
	}
	provenanceByID := map[string]ImportedEntityProvenance{}
	for _, provenance := range result.Provenance {
		provenanceByID[provenance.QualifiedID] = provenance
	}
	for _, child := range result.Root.Children {
		published := PublishedProjection{
			Alias: child.Dependency, Publication: child.Publication,
			ModulePath: child.ModulePath, Version: child.Version,
			SourceModel: child.Bundle.Manifest.Module.ModelID,
		}
		for sourceID, domain := range child.PublishedFacet {
			value, ok := projectionEntityByID(child.Bundle, domain, sourceID)
			if !ok {
				continue
			}
			qualifiedID, err := model.QualifyReference(child.Dependency, sourceID)
			if err != nil {
				continue
			}
			entity := PublishedProjectionEntity{
				QualifiedID: qualifiedID, SourceID: sourceID, Domain: domain, Facet: domain,
				Value: value, Source: &child.Bundle, Provenance: provenanceByID[qualifiedID],
			}
			published.Entities = append(published.Entities, entity)
		}
		sort.Slice(published.Entities, func(i, j int) bool {
			return published.Entities[i].QualifiedID < published.Entities[j].QualifiedID
		})
		projection.Publications = append(projection.Publications, published)
		projection.Entities = append(projection.Entities, published.Entities...)
	}
	sort.Slice(projection.Publications, func(i, j int) bool {
		if projection.Publications[i].Alias != projection.Publications[j].Alias {
			return projection.Publications[i].Alias < projection.Publications[j].Alias
		}
		return projection.Publications[i].Publication < projection.Publications[j].Publication
	})
	sort.Slice(projection.Entities, func(i, j int) bool {
		return projection.Entities[i].QualifiedID < projection.Entities[j].QualifiedID
	})
	return projection
}

// ResolveCompositionProjection resolves and validates composition for a bundle.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func ResolveCompositionProjection(bundle model.Bundle) (CompositionProjection, CompositionResult, error) {
	if !HasComposition(bundle) {
		return CompositionProjection{}, CompositionResult{}, nil
	}
	result, err := GenerateCompositionFromFile(bundle.ManifestPath)
	if err != nil {
		return CompositionProjection{}, result, err
	}
	if hasCompositionErrors(result.Diagnostics) {
		return CompositionProjection{}, result, fmt.Errorf("composition validation failed")
	}
	return BuildCompositionProjection(result), result, nil
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func enrichBundleFromComposition(bundle model.Bundle, domains ...string) (model.Bundle, error) {
	projection, _, err := ResolveCompositionProjection(bundle)
	if err != nil {
		return bundle, err
	}
	return projection.EnrichBundle(bundle, domains...), nil
}

// Domain returns published entities from one manifest publication facet.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func (projection CompositionProjection) Domain(domain string) []PublishedProjectionEntity {
	var entities []PublishedProjectionEntity
	for _, entity := range projection.Entities {
		if entity.Domain == domain {
			entities = append(entities, entity)
		}
	}
	return entities
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func projectionEntityByID(bundle model.Bundle, domain, id string) (any, bool) {
	var roots []any
	switch domain {
	case "architecture":
		roots = []any{
			bundle.Architecture.AuthoredArchitecture.FunctionalGroups,
			bundle.Architecture.AuthoredArchitecture.FunctionalUnits,
			bundle.Architecture.AuthoredArchitecture.Actors,
			bundle.Architecture.AuthoredArchitecture.ReferencedElements,
			bundle.Architecture.AuthoredArchitecture.Interfaces,
			bundle.Architecture.AuthoredArchitecture.DataObjects,
			bundle.Architecture.AuthoredArchitecture.DeploymentTargets,
			bundle.Architecture.AuthoredArchitecture.HardwareItems,
			bundle.Architecture.AuthoredArchitecture.HardwareInterfaces,
			bundle.Architecture.Contract,
		}
	case "behavior":
		roots = []any{bundle.Behavior.Behavior}
	case "assurance":
		roots = []any{bundle.Assurance.Assurance}
	case "compliance":
		roots = []any{bundle.Compliance.Compliance}
	case "requirements":
		roots = []any{bundle.Requirements.Requirements}
	case "views":
		roots = []any{bundle.Views.Views}
	}
	for _, root := range roots {
		if value, ok := findProjectionEntity(reflect.ValueOf(root), id); ok {
			return value, true
		}
	}
	return nil, false
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func findProjectionEntity(value reflect.Value, id string) (any, bool) {
	if !value.IsValid() {
		return nil, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, false
		}
		return findProjectionEntity(value.Elem(), id)
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if found, ok := findProjectionEntity(value.Index(i), id); ok {
				return found, true
			}
		}
	case reflect.Struct:
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			if strings.Split(field.Tag.Get("yaml"), ",")[0] == "id" &&
				value.Field(i).Kind() == reflect.String &&
				strings.TrimSpace(value.Field(i).String()) == id {
				return value.Interface(), true
			}
		}
		for i := 0; i < value.NumField(); i++ {
			if found, ok := findProjectionEntity(value.Field(i), id); ok {
				return found, true
			}
		}
	}
	return nil, false
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func (entity PublishedProjectionEntity) qualify(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.Contains(ref, "::") {
		return ref
	}
	if entity.Source != nil {
		for _, publication := range entity.Source.Manifest.Publications {
			if publication.ID != entity.Provenance.Publication {
				continue
			}
			for _, ids := range [][]string{
				publication.Architecture, publication.Behavior, publication.Assurance,
				publication.Compliance, publication.Requirements, publication.Views,
			} {
				for _, id := range ids {
					if id == ref {
						qualified, _ := model.QualifyReference(entity.Provenance.Alias, ref)
						return qualified
					}
				}
			}
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func (entity PublishedProjectionEntity) qualifyList(refs []string) []string {
	var qualified []string
	for _, ref := range refs {
		if value := entity.qualify(ref); value != "" {
			qualified = append(qualified, value)
		}
	}
	return qualified
}

// EnrichBundle returns a projection-only copy containing typed published
// entities. It is intended for native generators after the authored bundle has
// been validated; unsupported relationships are omitted rather than flattened.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func (projection CompositionProjection) EnrichBundle(bundle model.Bundle, domains ...string) model.Bundle {
	selected := map[string]bool{}
	for _, domain := range domains {
		selected[domain] = true
	}
	for _, entity := range projection.Entities {
		if !selected[entity.Domain] {
			continue
		}
		q := entity.QualifiedID
		switch value := entity.Value.(type) {
		case model.FunctionalGroup:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.FunctionalGroups = append(bundle.Architecture.AuthoredArchitecture.FunctionalGroups, value)
		case model.FunctionalUnit:
			value.ID, value.Group = q, entity.qualify(value.Group)
			bundle.Architecture.AuthoredArchitecture.FunctionalUnits = append(bundle.Architecture.AuthoredArchitecture.FunctionalUnits, value)
		case model.Actor:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.Actors = append(bundle.Architecture.AuthoredArchitecture.Actors, value)
		case model.ReferencedElement:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.ReferencedElements = append(bundle.Architecture.AuthoredArchitecture.ReferencedElements, value)
		case model.Interface:
			value.ID, value.Owner = q, entity.qualify(value.Owner)
			bundle.Architecture.AuthoredArchitecture.Interfaces = append(bundle.Architecture.AuthoredArchitecture.Interfaces, value)
		case model.DataObject:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.DataObjects = append(bundle.Architecture.AuthoredArchitecture.DataObjects, value)
		case model.DeploymentTarget:
			value.ID, value.TrustZone = q, entity.qualify(value.TrustZone)
			bundle.Architecture.AuthoredArchitecture.DeploymentTargets = append(bundle.Architecture.AuthoredArchitecture.DeploymentTargets, value)
		case model.HardwareItem:
			value.ID, value.Hosts = q, entity.qualifyList(value.Hosts)
			bundle.Architecture.AuthoredArchitecture.HardwareItems = append(bundle.Architecture.AuthoredArchitecture.HardwareItems, value)
		case model.HardwareInterface:
			value.ID, value.From, value.To = q, entity.qualify(value.From), entity.qualify(value.To)
			value.SoftwareInterfaceRef = entity.qualify(value.SoftwareInterfaceRef)
			if value.From != "" && value.To != "" {
				bundle.Architecture.AuthoredArchitecture.HardwareInterfaces = append(bundle.Architecture.AuthoredArchitecture.HardwareInterfaces, value)
			}
		case model.ContractEntry:
			value.ID, value.Ref = q, entity.qualify(value.Ref)
			if projectionContractRequires(entity) {
				bundle.Architecture.Contract.Requires = append(bundle.Architecture.Contract.Requires, value)
			} else {
				bundle.Architecture.Contract.Provides = append(bundle.Architecture.Contract.Provides, value)
			}
		case model.State:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.States = append(bundle.Architecture.AuthoredArchitecture.States, value)
			bundle.Behavior.Behavior.States = append(bundle.Behavior.Behavior.States, value)
		case model.Event:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.Events = append(bundle.Architecture.AuthoredArchitecture.Events, value)
			bundle.Behavior.Behavior.Events = append(bundle.Behavior.Behavior.Events, value)
		case model.Flow:
			value.ID = q
			value.SourceRef, value.DestinationRef = entity.qualify(value.SourceRef), entity.qualify(value.DestinationRef)
			value.DataRefs, value.Threats = entity.qualifyList(value.DataRefs), entity.qualifyList(value.Threats)
			for i := range value.Steps {
				value.Steps[i].Ref = entity.qualify(value.Steps[i].Ref)
				value.Steps[i].SourceRef = entity.qualify(value.Steps[i].SourceRef)
				value.Steps[i].DestinationRef = entity.qualify(value.Steps[i].DestinationRef)
				value.Steps[i].DataIn = entity.qualifyList(value.Steps[i].DataIn)
				value.Steps[i].DataOut = entity.qualifyList(value.Steps[i].DataOut)
				value.Steps[i].DataRefs = entity.qualifyList(value.Steps[i].DataRefs)
				value.Steps[i].InterfaceRef = entity.qualify(value.Steps[i].InterfaceRef)
				value.Steps[i].TrustBoundaryRef = entity.qualify(value.Steps[i].TrustBoundaryRef)
			}
			bundle.Architecture.AuthoredArchitecture.Flows = append(bundle.Architecture.AuthoredArchitecture.Flows, value)
			bundle.Behavior.Behavior.Flows = append(bundle.Behavior.Behavior.Flows, value)
		case model.Mapping:
			value.From, value.To = entity.qualify(value.From), entity.qualify(value.To)
			if value.From != "" && value.To != "" {
				bundle.Architecture.AuthoredArchitecture.Mappings = append(bundle.Architecture.AuthoredArchitecture.Mappings, value)
				bundle.Behavior.Behavior.Relationships = append(bundle.Behavior.Behavior.Relationships, value)
			}
		case model.AttackVector:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.AttackVectors = append(bundle.Architecture.AuthoredArchitecture.AttackVectors, value)
			bundle.Assurance.Assurance.AttackVectors = append(bundle.Assurance.Assurance.AttackVectors, value)
		case model.Control:
			value.ID = q
			bundle.Architecture.AuthoredArchitecture.Controls = append(bundle.Architecture.AuthoredArchitecture.Controls, value)
			bundle.Assurance.Assurance.Controls = append(bundle.Assurance.Assurance.Controls, value)
		case model.Risk:
			value.ID, value.Owner = q, entity.qualify(value.Owner)
			value.AppliesTo = entity.qualifyList(value.AppliesTo)
			value.RelatedControls = entity.qualifyList(value.RelatedControls)
			value.AttackVectors = entity.qualifyList(value.AttackVectors)
			value.ThreatScenarios = entity.qualifyList(value.ThreatScenarios)
			bundle.Architecture.AuthoredArchitecture.Risks = append(bundle.Architecture.AuthoredArchitecture.Risks, value)
			bundle.Assurance.Assurance.Risks = append(bundle.Assurance.Assurance.Risks, value)
		case model.POAMItem:
			value.ID, value.RiskRef = q, entity.qualify(value.RiskRef)
			value.ResponsibleRole = entity.qualify(value.ResponsibleRole)
			bundle.Architecture.AuthoredArchitecture.POAMItems = append(bundle.Architecture.AuthoredArchitecture.POAMItems, value)
			bundle.Assurance.Assurance.POAMItems = append(bundle.Assurance.Assurance.POAMItems, value)
		case model.TrustBoundary:
			value.ID, value.ParentRef = q, entity.qualify(value.ParentRef)
			value.Members = entity.qualifyList(value.Members)
			bundle.Architecture.AuthoredArchitecture.TrustBoundaries = append(bundle.Architecture.AuthoredArchitecture.TrustBoundaries, value)
			bundle.Assurance.Assurance.TrustBoundaries = append(bundle.Assurance.Assurance.TrustBoundaries, value)
		case model.ThreatScenario:
			value.ID = q
			value.Owner = entity.qualify(value.Owner)
			value.AttackVectorRef, value.RiskRef = entity.qualify(value.AttackVectorRef), entity.qualify(value.RiskRef)
			value.AppliesTo = entity.qualifyList(value.AppliesTo)
			value.FlowRefs = entity.qualifyList(value.FlowRefs)
			value.RelatedControls = entity.qualifyList(value.RelatedControls)
			value.AssumptionRefs = entity.qualifyList(value.AssumptionRefs)
			value.OutOfScopeRefs = entity.qualifyList(value.OutOfScopeRefs)
			value.MitigationRefs = entity.qualifyList(value.MitigationRefs)
			value.VerificationRefs = entity.qualifyList(value.VerificationRefs)
			bundle.Architecture.AuthoredArchitecture.ThreatScenarios = append(bundle.Architecture.AuthoredArchitecture.ThreatScenarios, value)
			bundle.Assurance.Assurance.ThreatScenarios = append(bundle.Assurance.Assurance.ThreatScenarios, value)
		case model.ThreatAssumption:
			value.ID, value.AppliesTo = q, entity.qualifyList(value.AppliesTo)
			value.Owner = entity.qualify(value.Owner)
			bundle.Architecture.AuthoredArchitecture.ThreatAssumptions = append(bundle.Architecture.AuthoredArchitecture.ThreatAssumptions, value)
			bundle.Assurance.Assurance.ThreatAssumptions = append(bundle.Assurance.Assurance.ThreatAssumptions, value)
		case model.ThreatOutOfScope:
			value.ID, value.AppliesTo = q, entity.qualifyList(value.AppliesTo)
			value.Owner = entity.qualify(value.Owner)
			bundle.Architecture.AuthoredArchitecture.ThreatOutOfScope = append(bundle.Architecture.AuthoredArchitecture.ThreatOutOfScope, value)
			bundle.Assurance.Assurance.ThreatOutOfScope = append(bundle.Assurance.Assurance.ThreatOutOfScope, value)
		case model.ThreatMitigation:
			value.ID = q
			value.Owner = entity.qualify(value.Owner)
			value.ThreatScenarioRef, value.ControlRef = entity.qualify(value.ThreatScenarioRef), entity.qualify(value.ControlRef)
			value.VerificationRefs = entity.qualifyList(value.VerificationRefs)
			bundle.Architecture.AuthoredArchitecture.ThreatMitigations = append(bundle.Architecture.AuthoredArchitecture.ThreatMitigations, value)
			bundle.Assurance.Assurance.ThreatMitigations = append(bundle.Assurance.Assurance.ThreatMitigations, value)
		case model.ControlVerification:
			value.ID, value.ControlRef = q, entity.qualify(value.ControlRef)
			value.Owner = entity.qualify(value.Owner)
			value.ThreatScenarioRefs = entity.qualifyList(value.ThreatScenarioRefs)
			value.RiskRefs = entity.qualifyList(value.RiskRefs)
			bundle.Architecture.AuthoredArchitecture.ControlVerifications = append(bundle.Architecture.AuthoredArchitecture.ControlVerifications, value)
			bundle.Assurance.Assurance.ControlVerifications = append(bundle.Assurance.Assurance.ControlVerifications, value)
		case model.ComplianceProfile:
			value.ID = q
			value.Href = projectionSourcePath(bundle, entity, value.Href)
			value.CatalogHref = projectionSourcePath(bundle, entity, value.CatalogHref)
			bundle.Architecture.Compliance.Profiles = append(bundle.Architecture.Compliance.Profiles, value)
			bundle.Compliance.Compliance.Profiles = append(bundle.Compliance.Compliance.Profiles, value)
		case model.ComplianceMapping:
			value.ID, value.ProfileRef = q, entity.qualify(value.ProfileRef)
			value.ModelControlRef = entity.qualify(value.ModelControlRef)
			value.AppliesTo = entity.qualifyList(value.AppliesTo)
			value.InheritedFrom = entity.qualifyList(value.InheritedFrom)
			value.ResponsibleRoles = entity.qualifyList(value.ResponsibleRoles)
			if value.ProfileRef != "" && value.ModelControlRef != "" {
				bundle.Architecture.Compliance.Mappings = append(bundle.Architecture.Compliance.Mappings, value)
				bundle.Compliance.Compliance.Mappings = append(bundle.Compliance.Compliance.Mappings, value)
			}
		case model.Requirement:
			value.ID, value.AppliesTo = q, entity.qualifyList(value.AppliesTo)
			bundle.Requirements.Requirements = append(bundle.Requirements.Requirements, value)
		case model.View:
			value.ID, value.Roots = q, entity.qualifyList(value.Roots)
			bundle.Architecture.Views = append(bundle.Architecture.Views, value)
			bundle.Views.Views = append(bundle.Views.Views, value)
		}
	}
	return bundle
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func projectionSourcePath(bundle model.Bundle, entity PublishedProjectionEntity, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "://") ||
		entity.Source == nil || strings.TrimSpace(entity.Source.ManifestPath) == "" {
		return path
	}
	base := filepath.Dir(bundle.ManifestPath)
	sourcePath := filepath.Join(filepath.Dir(entity.Source.ManifestPath), filepath.FromSlash(path))
	relative, err := filepath.Rel(base, sourcePath)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func projectionContractRequires(entity PublishedProjectionEntity) bool {
	if entity.Source == nil {
		return false
	}
	for _, entry := range entity.Source.Architecture.Contract.Requires {
		if strings.TrimSpace(entry.ID) == entity.SourceID {
			return true
		}
	}
	return false
}
