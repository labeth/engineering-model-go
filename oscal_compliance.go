// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DO-ARCHITECTURE-MODEL
type oscalComplianceContext struct {
	Profiles map[string]oscalComplianceProfile
	Mappings []oscalComplianceMapping
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalComplianceProfile struct {
	ID          string
	Href        string
	CatalogHref string
	Controls    map[string]oscalCatalogControl
	Loaded      bool
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalComplianceMapping struct {
	ID                   string
	ProfileRef           string
	ControlIDs           []string
	ModelControlRef      string
	AppliesTo            []string
	ImplementationType   string
	ImplementationStatus string
	Narrative            string
	Rationale            string
	Evidence             []model.ControlEvidence
	InheritedFrom        []string
	ResponsibleRoles     []string
	Source               string
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalCatalogControl struct {
	ID    string
	Title string
}

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type oscalResolveOptions struct {
	ProfileHref string
	CatalogHref string
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DO-ARCHITECTURE-MODEL
func resolveOSCALCompliance(bundle model.Bundle, options oscalResolveOptions) (oscalComplianceContext, []validate.Diagnostic) {
	ctx := oscalComplianceContext{Profiles: map[string]oscalComplianceProfile{}}
	diags := []validate.Diagnostic{}

	for _, p := range bundle.Architecture.Compliance.Profiles {
		profile := oscalComplianceProfile{
			ID:          strings.TrimSpace(p.ID),
			Href:        strings.TrimSpace(p.Href),
			CatalogHref: strings.TrimSpace(p.CatalogHref),
			Controls:    map[string]oscalCatalogControl{},
		}
		loadComplianceProfile(bundle, &profile, &diags)
		if profile.ID != "" {
			ctx.Profiles[profile.ID] = profile
		}
	}

	if len(ctx.Profiles) == 0 && (strings.TrimSpace(options.ProfileHref) != "" || strings.TrimSpace(options.CatalogHref) != "") {
		profile := oscalComplianceProfile{
			ID:          "PROFILE-CLI",
			Href:        strings.TrimSpace(options.ProfileHref),
			CatalogHref: strings.TrimSpace(options.CatalogHref),
			Controls:    map[string]oscalCatalogControl{},
		}
		loadComplianceProfile(bundle, &profile, &diags)
		ctx.Profiles[profile.ID] = profile
	}

	for _, m := range bundle.Architecture.Compliance.Mappings {
		ctx.Mappings = append(ctx.Mappings, oscalComplianceMapping{
			ID:                   strings.TrimSpace(m.ID),
			ProfileRef:           strings.TrimSpace(m.ProfileRef),
			ControlIDs:           normalizeControlIDs(m.ControlIDs),
			ModelControlRef:      strings.TrimSpace(m.ModelControlRef),
			AppliesTo:            cleanStrings(m.AppliesTo),
			ImplementationType:   strings.TrimSpace(m.ImplementationType),
			ImplementationStatus: firstNonEmptyString(strings.TrimSpace(m.ImplementationStatus), strings.TrimSpace(m.Status)),
			Narrative:            strings.TrimSpace(m.Narrative),
			Rationale:            strings.TrimSpace(m.Rationale),
			Evidence:             m.Evidence,
			InheritedFrom:        cleanStrings(m.InheritedFrom),
			ResponsibleRoles:     cleanStrings(m.ResponsibleRoles),
			Source:               "compliance.mapping",
		})
	}

	for _, m := range ctx.Mappings {
		profile := ctx.Profiles[m.ProfileRef]
		if !profile.Loaded || len(profile.Controls) == 0 {
			continue
		}
		for _, cid := range m.ControlIDs {
			if _, ok := profile.Controls[cid]; !ok {
				diags = append(diags, validate.Diagnostic{
					Code:     "oscal.control_not_in_profile",
					Severity: validate.SeverityError,
					Message:  fmt.Sprintf("mapped OSCAL control %q is not selected by compliance profile %q", cid, m.ProfileRef),
					Path:     "compliance.mappings." + m.ID,
				})
			}
		}
	}

	sort.SliceStable(ctx.Mappings, func(i, j int) bool { return ctx.Mappings[i].ID < ctx.Mappings[j].ID })
	return ctx, validate.SortDiagnostics(diags)
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func loadComplianceProfile(bundle model.Bundle, profile *oscalComplianceProfile, diags *[]validate.Diagnostic) {
	if profile == nil {
		return
	}
	if strings.TrimSpace(profile.Href) != "" && !isRemoteHref(profile.Href) {
		selected, err := loadOSCALProfileControls(bundle, profile.Href, profile.CatalogHref)
		if err != nil {
			*diags = append(*diags, validate.Diagnostic{Code: "oscal.profile_load_failed", Severity: validate.SeverityWarning, Message: err.Error(), Path: profile.Href})
			return
		}
		profile.Controls = selected
		profile.Loaded = true
		return
	}
	if strings.TrimSpace(profile.CatalogHref) != "" {
		controls, err := loadOSCALCatalogControls(bundle, profile.CatalogHref, "")
		if err != nil {
			*diags = append(*diags, validate.Diagnostic{Code: "oscal.catalog_load_failed", Severity: validate.SeverityWarning, Message: err.Error(), Path: profile.CatalogHref})
		} else {
			profile.Controls = controls
			profile.Loaded = true
		}
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func loadOSCALProfileControls(bundle model.Bundle, profileHref, defaultCatalogHref string) (map[string]oscalCatalogControl, error) {
	profilePath, ok := resolveOSCALPath(bundle, profileHref, "")
	if !ok {
		return nil, fmt.Errorf("profile href %q is not a local file", profileHref)
	}
	var doc map[string]any
	if err := readJSONFile(profilePath, &doc); err != nil {
		return nil, err
	}
	profileObj, _ := doc["profile"].(map[string]any)
	imports, _ := profileObj["imports"].([]any)
	out := map[string]oscalCatalogControl{}
	for _, rawImport := range imports {
		imp, _ := rawImport.(map[string]any)
		href := strings.TrimSpace(toStringAny(imp["href"]))
		if href == "" {
			href = strings.TrimSpace(defaultCatalogHref)
		}
		catalog, err := loadOSCALCatalogControls(bundle, href, filepath.Dir(profilePath))
		if err != nil {
			return nil, err
		}
		selected := includeControlIDs(imp)
		excluded := excludeControlIDs(imp)
		if len(selected) == 0 {
			out = mergeCatalogControls(out, catalog)
		} else {
			for _, id := range selected {
				if c, ok := catalog[id]; ok {
					out[id] = c
				}
			}
		}
		for _, id := range excluded {
			delete(out, id)
		}
	}
	if len(imports) == 0 && strings.TrimSpace(defaultCatalogHref) != "" {
		return loadOSCALCatalogControls(bundle, defaultCatalogHref, filepath.Dir(profilePath))
	}
	return out, nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func loadOSCALCatalogControls(bundle model.Bundle, href, baseDir string) (map[string]oscalCatalogControl, error) {
	path, ok := resolveOSCALPath(bundle, href, baseDir)
	if !ok {
		return nil, fmt.Errorf("catalog href %q is not a local file", href)
	}
	var doc map[string]any
	if err := readJSONFile(path, &doc); err != nil {
		return nil, err
	}
	catalog, _ := doc["catalog"].(map[string]any)
	out := map[string]oscalCatalogControl{}
	collectCatalogControls(catalog, out)
	return out, nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func collectCatalogControls(node map[string]any, out map[string]oscalCatalogControl) {
	for _, key := range []string{"groups", "controls"} {
		items, _ := node[key].([]any)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			id := normalizeOSCALControlID(toStringAny(item["id"]))
			if id != "" {
				out[id] = oscalCatalogControl{ID: id, Title: strings.TrimSpace(toStringAny(item["title"]))}
			}
			collectCatalogControls(item, out)
		}
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func includeControlIDs(importObj map[string]any) []string {
	return profileControlIDs(importObj["include-controls"])
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func excludeControlIDs(importObj map[string]any) []string {
	return profileControlIDs(importObj["exclude-controls"])
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func profileControlIDs(raw any) []string {
	out := []string{}
	switch x := raw.(type) {
	case []any:
		for _, item := range x {
			obj, _ := item.(map[string]any)
			out = append(out, stringArrayAny(obj["with-ids"])...)
			out = append(out, stringArrayAny(obj["matching"])...)
		}
	case map[string]any:
		out = append(out, stringArrayAny(x["with-ids"])...)
		out = append(out, stringArrayAny(x["matching"])...)
	}
	return normalizeControlIDs(out)
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func readJSONFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func resolveOSCALPath(bundle model.Bundle, href, baseDir string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" || isRemoteHref(href) {
		return "", false
	}
	if filepath.IsAbs(href) {
		return filepath.Clean(href), true
	}
	if baseDir != "" {
		return filepath.Clean(filepath.Join(baseDir, href)), true
	}
	if bundle.ArchitecturePath != "" {
		return filepath.Clean(filepath.Join(filepath.Dir(bundle.ArchitecturePath), href)), true
	}
	return filepath.Clean(href), true
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func isRemoteHref(href string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(href)), "http://") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(href)), "https://")
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mergeCatalogControls(left, right map[string]oscalCatalogControl) map[string]oscalCatalogControl {
	if left == nil {
		left = map[string]oscalCatalogControl{}
	}
	for k, v := range right {
		left[k] = v
	}
	return left
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func normalizeControlIDs(in []string) []string {
	out := []string{}
	for _, x := range in {
		if id := normalizeOSCALControlID(x); id != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return uniqueStringList(out)
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func cleanStrings(in []string) []string {
	out := []string{}
	for _, x := range in {
		if s := strings.TrimSpace(x); s != "" {
			out = append(out, s)
		}
	}
	return uniqueStringList(out)
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func uniqueStringList(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range in {
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func toStringAny(v any) string {
	s, _ := v.(string)
	return s
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func stringArrayAny(v any) []string {
	items, _ := v.([]any)
	out := []string{}
	for _, item := range items {
		if s := strings.TrimSpace(toStringAny(item)); s != "" {
			out = append(out, s)
		}
	}
	return out
}
