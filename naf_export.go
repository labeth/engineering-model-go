// ENGMODEL-OWNER-UNIT: FU-NAF-EXPORTER
package engmodel

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
	"github.com/labeth/engineering-model-go/view"
)

// NAFExportResult is a deterministic NAF v4.1 architecture document and its diagnostics.
type NAFExportResult struct {
	Document    string
	Diagnostics []validate.Diagnostic
}

// GenerateNAFV41FromFile loads canonical YAML and generates the selected NAF products.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-042, REQ-EMG-043
// ENGMODEL-LINKS: FU-NAF-EXPORTER, IF-CLI-ENGNAF, DO-NAF-V4-ARCHITECTURE, REF-NAF-V4-1-SPECIFICATION
func GenerateNAFV41FromFile(architecturePath string) (NAFExportResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return NAFExportResult{}, err
	}
	return GenerateNAFV41(bundle)
}

// GenerateNAFV41 validates and projects existing Engmod views into NAF products.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-042, REQ-EMG-043
func GenerateNAFV41(bundle model.Bundle) (NAFExportResult, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return NAFExportResult{}, err
	}
	bundle = canonical.Documents()
	diagnostics := validate.Bundle(bundle)
	if validate.HasErrors(diagnostics) {
		return NAFExportResult{Diagnostics: diagnostics}, fmt.Errorf("NAF validation failed")
	}
	if !bundle.Architecture.NAF.Enabled() {
		diagnostic := validate.Diagnostic{
			Code: "naf.profile_missing", Severity: validate.SeverityError,
			Message: "architecture does not contain a NAF profile", Path: "naf",
		}
		return NAFExportResult{Diagnostics: []validate.Diagnostic{diagnostic}}, fmt.Errorf("NAF profile is required")
	}

	products := append([]model.NAFProduct(nil), bundle.Architecture.NAF.Products...)
	sort.Slice(products, func(i, j int) bool {
		if products[i].Viewpoint != products[j].Viewpoint {
			return products[i].Viewpoint < products[j].Viewpoint
		}
		return products[i].ID < products[j].ID
	})
	projected := make(map[string]view.ProjectedView, len(products))
	for _, product := range products {
		productView, viewDiagnostics := view.Build(bundle, product.ViewRef)
		diagnostics = append(diagnostics, viewDiagnostics...)
		projected[product.ID] = productView
	}
	diagnostics = validate.SortDiagnostics(diagnostics)
	if validate.HasErrors(diagnostics) {
		return NAFExportResult{Diagnostics: diagnostics}, fmt.Errorf("NAF view projection failed")
	}

	return NAFExportResult{
		Document:    renderNAFV41(bundle, products, projected),
		Diagnostics: diagnostics,
	}, nil
}

// TRLC-LINKS: REQ-EMG-042
func renderNAFV41(bundle model.Bundle, products []model.NAFProduct, projected map[string]view.ProjectedView) string {
	profile := bundle.Architecture.NAF
	var out bytes.Buffer
	fmt.Fprintf(&out, "= %s\n", asciidocCell(profile.ArchitectureDescription))
	fmt.Fprintln(&out, ":naf-framework: NATO Architecture Framework")
	fmt.Fprintf(&out, ":naf-version: %s\n", profile.Version)
	fmt.Fprintf(&out, ":engmod-model-id: %s\n\n", bundle.Architecture.Model.ID)
	fmt.Fprintln(&out, "This architecture description applies NAF 4.1 profile metadata to canonical Engineering Model views.")
	fmt.Fprintln(&out, "Architecture elements and relationships remain defined once in the canonical SysML/KerML-aligned semantic model.")

	stakeholders := append([]model.NAFStakeholder(nil), profile.Stakeholders...)
	sort.Slice(stakeholders, func(i, j int) bool { return stakeholders[i].ActorRef < stakeholders[j].ActorRef })
	actorNames := make(map[string]string)
	for _, actor := range bundle.Architecture.AuthoredArchitecture.Actors {
		actorNames[actor.ID] = actor.Name
	}
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "== Stakeholders")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "[cols=\"1,2,3\",options=\"header\"]")
	fmt.Fprintln(&out, "|===")
	fmt.Fprintln(&out, "|Actor ID |Stakeholder |Role")
	for _, stakeholder := range stakeholders {
		fmt.Fprintf(&out, "|`%s` |%s |%s\n", asciidocCell(stakeholder.ActorRef), asciidocCell(actorNames[stakeholder.ActorRef]), asciidocCell(stakeholder.Role))
	}
	fmt.Fprintln(&out, "|===")

	concerns := append([]model.NAFConcern(nil), profile.Concerns...)
	sort.Slice(concerns, func(i, j int) bool { return concerns[i].ID < concerns[j].ID })
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "== Concerns")
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, "[cols=\"1,2,4,3\",options=\"header\"]")
	fmt.Fprintln(&out, "|===")
	fmt.Fprintln(&out, "|Concern ID |Name |Description |Stakeholders")
	for _, concern := range concerns {
		refs := append([]string(nil), concern.StakeholderRefs...)
		sort.Strings(refs)
		fmt.Fprintf(&out, "|`%s` |%s |%s |%s\n", asciidocCell(concern.ID), asciidocCell(concern.Name), asciidocCell(concern.Description), asciidocCell(strings.Join(refs, ", ")))
	}
	fmt.Fprintln(&out, "|===")

	fmt.Fprintln(&out, "\n== Architecture Products")
	for _, product := range products {
		viewpointTitle, _ := model.NAFV41ViewpointTitle(product.Viewpoint)
		fmt.Fprintf(&out, "\n=== %s - %s\n\n", product.Viewpoint, asciidocCell(viewpointTitle))
		fmt.Fprintf(&out, "*Product:* `%s` +\n", asciidocCell(product.ID))
		fmt.Fprintf(&out, "*Title:* %s +\n", asciidocCell(product.Title))
		fmt.Fprintf(&out, "*Canonical view:* `%s` +\n", asciidocCell(product.ViewRef))
		refs := append([]string(nil), product.ConcernRefs...)
		sort.Strings(refs)
		fmt.Fprintf(&out, "*Concerns:* %s\n", asciidocCell(strings.Join(refs, ", ")))

		productView := projected[product.ID]
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "==== Elements")
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "[cols=\"2,2,4\",options=\"header\"]")
		fmt.Fprintln(&out, "|===")
		fmt.Fprintln(&out, "|Canonical ID |Kind |Label")
		for _, node := range productView.Nodes {
			fmt.Fprintf(&out, "|`%s` |%s |%s\n", asciidocCell(node.ID), asciidocCell(node.Kind), asciidocCell(node.Label))
		}
		fmt.Fprintln(&out, "|===")

		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "==== Relationships")
		fmt.Fprintln(&out)
		if len(productView.Edges) == 0 {
			fmt.Fprintln(&out, "No relationships are selected by this architecture product.")
			continue
		}
		fmt.Fprintln(&out, "[cols=\"2,2,2,4\",options=\"header\"]")
		fmt.Fprintln(&out, "|===")
		fmt.Fprintln(&out, "|From |Type |To |Description")
		for _, edge := range productView.Edges {
			fmt.Fprintf(&out, "|`%s` |%s |`%s` |%s\n", asciidocCell(edge.From), asciidocCell(edge.Type), asciidocCell(edge.To), asciidocCell(edge.Label))
		}
		fmt.Fprintln(&out, "|===")
	}
	return out.String()
}

// TRLC-LINKS: REQ-EMG-042
func asciidocCell(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "|", `\|`)
	return strings.ReplaceAll(value, "\n", " ")
}
