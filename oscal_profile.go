// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package engmodel

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

// ErrNoOSCALProfiles indicates that the model has no compliance profile to
// publish as an aggregate OSCAL profile.
var ErrNoOSCALProfiles = errors.New("no OSCAL compliance profiles are authored")

// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type OSCALProfileOptions struct {
	Title        string
	Version      string
	LastModified string
}

type oscalAggregateProfileDocument struct {
	Profile oscalAggregateProfile `json:"profile"`
}

type oscalAggregateProfile struct {
	UUID     string                 `json:"uuid"`
	Metadata oscalMetadata          `json:"metadata"`
	Imports  []oscalAggregateImport `json:"imports"`
}

type oscalAggregateImport struct {
	Href string `json:"href"`
}

// GenerateOSCALProfileFromFile loads a model and emits one profile importing
// every authored compliance profile.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALProfileFromFile(path string, options OSCALProfileOptions) (string, error) {
	bundle, err := model.LoadBundle(path)
	if err != nil {
		return "", err
	}
	bundle, err = enrichBundleFromComposition(bundle, "architecture", "assurance", "compliance")
	if err != nil {
		return "", err
	}
	return GenerateOSCALProfile(bundle, options)
}

// GenerateOSCALProfile emits a deterministic OSCAL 1.1.2 aggregate profile so
// multi-framework SSPs import every framework they implement.
// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateOSCALProfile(bundle model.Bundle, options OSCALProfileOptions) (string, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return "", err
	}
	bundle = canonical.Documents()

	hrefs := make([]string, 0, len(bundle.Architecture.Compliance.Profiles))
	for _, profile := range bundle.Architecture.Compliance.Profiles {
		href := strings.TrimSpace(profile.Href)
		if href == "" {
			href = strings.TrimSpace(profile.CatalogHref)
		}
		if href != "" {
			hrefs = append(hrefs, href)
		}
	}
	hrefs = cleanStrings(hrefs)
	sort.Strings(hrefs)
	if len(hrefs) == 0 {
		return "", ErrNoOSCALProfiles
	}

	lastModified, err := resolveOSCALTimestamp(bundle, options.LastModified)
	if err != nil {
		return "", err
	}
	title := strings.TrimSpace(options.Title)
	if title == "" {
		title = strings.TrimSpace(bundle.Architecture.Model.Title) + " Compliance Profile"
	}
	version := strings.TrimSpace(options.Version)
	if version == "" {
		version = "0.1.0"
	}
	imports := make([]oscalAggregateImport, 0, len(hrefs))
	for _, href := range hrefs {
		imports = append(imports, oscalAggregateImport{Href: href})
	}
	doc := oscalAggregateProfileDocument{Profile: oscalAggregateProfile{
		UUID: deterministicUUID("oscal-profile|" + bundle.Architecture.Model.ID + "|" + strings.Join(hrefs, "|")),
		Metadata: oscalMetadata{
			Title:        title,
			LastModified: lastModified,
			Version:      version,
			OSCALVersion: "1.1.2",
		},
		Imports: imports,
	}}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
