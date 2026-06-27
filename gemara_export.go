// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// Gemara export: render the engineering model into OpenSSF Gemara documents
// (https://gemara.openssf.org) using the official Go SDK types from
// github.com/gemaraproj/go-gemara. Output is serialized with the same YAML
// codec the SDK uses (goccy/go-yaml), so every artifact round-trips through the
// SDK and validates against the published CUE schemas.
//
// Layer coverage produced here:
//   - L1 Vector Catalog        <- AttackVectors
//   - L2 Capability Catalog    <- FunctionalUnits (+ a system capability)
//   - L2 Threat Catalog        <- ThreatScenarios
//   - L2 Control Catalog       <- Controls + ControlVerifications (assessment-requirements)
//   - L3 Risk Catalog          <- Risks (severity derived from likelihood x impact)
//
// The L5 Evaluation Log is produced in gemara_evaluation.go.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	gemara "github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"

	"github.com/labeth/engineering-model-go/model"
)

// GemaraVersion is the Gemara specification version stamped into every artifact's metadata.
const GemaraVersion = "1.1.0"

// Stable mapping-reference ids used to cross-link the generated catalogs.
const (
	gemaraRefControlCatalog    = "ENGMOD-CONTROL-CATALOG"
	gemaraRefThreatCatalog     = "ENGMOD-THREAT-CATALOG"
	gemaraRefVectorCatalog     = "ENGMOD-VECTOR-CATALOG"
	gemaraRefCapabilityCatalog = "ENGMOD-CAPABILITY-CATALOG"
	gemaraRefRiskCatalog       = "ENGMOD-RISK-CATALOG"
)

// systemCapabilityID is a synthetic catch-all capability representing the whole
// system, used so every threat can satisfy the required (non-empty) capabilities list.
const systemCapabilityID = "CAP-SYSTEM"

// GemaraExportOptions configures the deterministic metadata stamped into artifacts.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraExportOptions struct {
	AuthorID   string // metadata.author.id (default "engmod")
	AuthorName string // metadata.author.name (default "Engineering Model")
	Version    string // metadata.version (optional)
	Date       string // metadata.date, ISO 8601 (optional; omitted when empty for reproducibility)
}

// GemaraExportResult holds the typed Gemara documents and their YAML serializations.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraExportResult struct {
	ControlCatalog    gemara.ControlCatalog
	ThreatCatalog     gemara.ThreatCatalog
	RiskCatalog       gemara.RiskCatalog
	VectorCatalog     gemara.VectorCatalog
	CapabilityCatalog gemara.CapabilityCatalog

	// Extended artifacts (emitted only when the model has supporting data).
	PrincipleCatalog     gemara.PrincipleCatalog
	GuidanceCatalog      gemara.GuidanceCatalog
	Policy               gemara.Policy
	Lexicon              gemara.Lexicon
	ControlThreatMapping gemara.MappingDocument
	AuditLog             gemara.AuditLog
	EnforcementLog       gemara.EnforcementLog
	HasGuidance          bool
	HasPolicy            bool
	HasMapping           bool
	HasAudit             bool
	HasEnforcement       bool

	// YAML maps an artifact short-name (e.g. "control-catalog") to its serialized document.
	YAML map[string]string
}

// GenerateGemaraFromFile loads the model bundle and renders the Gemara catalogs.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraFromFile(architecturePath string, options GemaraExportOptions) (GemaraExportResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return GemaraExportResult{}, err
	}
	return GenerateGemara(bundle, options)
}

// GenerateGemara renders the engineering model bundle into Gemara L1-L3 documents.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemara(bundle model.Bundle, options GemaraExportOptions) (GemaraExportResult, error) {
	cfg := newGemaraConfig(bundle, options)

	res := GemaraExportResult{
		VectorCatalog:     buildVectorCatalog(bundle, cfg),
		CapabilityCatalog: buildCapabilityCatalog(bundle, cfg),
		ControlCatalog:    buildControlCatalog(bundle, cfg),
		ThreatCatalog:     buildThreatCatalog(bundle, cfg),
		RiskCatalog:       buildRiskCatalog(bundle, cfg),
	}

	res.YAML = map[string]string{}
	docs := []struct {
		name string
		doc  any
	}{
		{"vector-catalog", res.VectorCatalog},
		{"capability-catalog", res.CapabilityCatalog},
		{"control-catalog", res.ControlCatalog},
		{"threat-catalog", res.ThreatCatalog},
		{"risk-catalog", res.RiskCatalog},
	}

	// L1 Principle + Guidance and L3 Policy.
	res.PrincipleCatalog = buildPrincipleCatalog(bundle, cfg)
	docs = append(docs, struct {
		name string
		doc  any
	}{"principle-catalog", res.PrincipleCatalog})
	if gc, ok := buildGuidanceCatalog(bundle, cfg); ok {
		res.GuidanceCatalog, res.HasGuidance = gc, true
		docs = append(docs, struct {
			name string
			doc  any
		}{"guidance-catalog", gc})
	}
	if pol, ok := buildPolicy(bundle, cfg, options.Date); ok {
		res.Policy, res.HasPolicy = pol, true
		docs = append(docs, struct {
			name string
			doc  any
		}{"policy", pol})
	}

	// Extended, data-derived artifacts.
	if lex, ok := buildLexicon(bundle, cfg); ok {
		res.Lexicon = lex
		docs = append(docs, struct {
			name string
			doc  any
		}{"lexicon", lex})
	}
	if md, ok := buildControlThreatMapping(bundle, cfg); ok {
		res.ControlThreatMapping, res.HasMapping = md, true
		docs = append(docs, struct {
			name string
			doc  any
		}{"control-threat-mapping", md})
	}
	if al, ok := buildAuditLog(bundle, cfg, options.Date); ok {
		res.AuditLog, res.HasAudit = al, true
		docs = append(docs, struct {
			name string
			doc  any
		}{"audit-log", al})
	}
	if el, ok := buildEnforcementLog(bundle, cfg, options.Date); ok {
		res.EnforcementLog, res.HasEnforcement = el, true
		docs = append(docs, struct {
			name string
			doc  any
		}{"enforcement-log", el})
	}

	for _, d := range docs {
		out, err := marshalGemara(d.doc)
		if err != nil {
			return GemaraExportResult{}, fmt.Errorf("marshal gemara %s: %w", d.name, err)
		}
		res.YAML[d.name] = out
	}
	return res, nil
}

// marshalGemara serializes a Gemara document with the same codec the SDK uses.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func marshalGemara(v any) (string, error) {
	b, err := goyaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// gemaraConfig carries shared, model-derived context for all builders.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type gemaraConfig struct {
	options    GemaraExportOptions
	modelID    string
	modelTitle string
	author     gemara.Actor
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func newGemaraConfig(bundle model.Bundle, options GemaraExportOptions) gemaraConfig {
	if strings.TrimSpace(options.AuthorID) == "" {
		options.AuthorID = "engmod"
	}
	if strings.TrimSpace(options.AuthorName) == "" {
		options.AuthorName = "Engineering Model"
	}
	id := strings.TrimSpace(bundle.Architecture.Model.ID)
	if id == "" {
		id = "engmod-model"
	}
	title := strings.TrimSpace(bundle.Architecture.Model.Title)
	if title == "" {
		title = id
	}
	return gemaraConfig{
		options:    options,
		modelID:    slug(id),
		modelTitle: title,
		author: gemara.Actor{
			Id:   slug(options.AuthorID),
			Name: options.AuthorName,
			Type: gemara.Human,
		},
	}
}

// newMetadata builds a valid #Metadata block for a catalog/log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c gemaraConfig) newMetadata(id, description string, t gemara.ArtifactType) gemara.Metadata {
	m := gemara.Metadata{
		Id:            id,
		Type:          t,
		GemaraVersion: GemaraVersion,
		Description:   fallback(description, id),
		Author:        c.author,
		Version:       strings.TrimSpace(c.options.Version),
	}
	if d := normalizeDatetime(c.options.Date); d != "" {
		m.Date = gemara.Datetime(d)
	}
	return m
}

// normalizeDatetime coerces a date or datetime string into RFC3339 (the format
// Gemara's #Datetime requires). Returns "" when the input is empty or unparseable.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func normalizeDatetime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return s
	}
	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return ""
}

// -------------------- L1: Vector Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildVectorCatalog(bundle model.Bundle, cfg gemaraConfig) gemara.VectorCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemara.VectorCatalog{
		Title:    cfg.modelTitle + " - Attack Vectors",
		Metadata: cfg.newMetadata(gemaraRefVectorCatalog, "Attack vectors relevant to "+cfg.modelTitle, gemara.VectorCatalogArtifact),
	}
	if len(a.AttackVectors) == 0 {
		return cat
	}
	group := gemara.Group{Id: "attack-vector", Title: "Attack Vector", Description: "Methods and pathways through which a threat may be realized."}
	cat.Groups = []gemara.Group{group}

	// Applicability groups from deployment environments, plus an all-environments catch-all.
	appGroups := newGroupSet()
	for _, dt := range a.DeploymentTargets {
		if env := strings.TrimSpace(dt.Environment); env != "" {
			appGroups.add(slug(env), strings.Title(env), "Vectors applicable in the "+env+" environment.")
		}
	}
	const allEnv = "all-environments"
	appGroups.add(allEnv, "All Environments", "Applies across every environment.")
	cat.Metadata.ApplicabilityGroups = appGroups.list()

	for _, av := range a.AttackVectors {
		cat.Vectors = append(cat.Vectors, gemara.Vector{
			Id:            av.ID,
			Title:         fallback(av.Name, av.ID),
			Description:   fallback(av.Description, fallback(av.Name, av.ID)),
			Group:         group.Id,
			Applicability: []string{allEnv},
		})
	}
	return cat
}

// -------------------- L2: Capability Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildCapabilityCatalog(bundle model.Bundle, cfg gemaraConfig) gemara.CapabilityCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemara.CapabilityCatalog{
		Title:    cfg.modelTitle + " - Capabilities",
		Metadata: cfg.newMetadata(gemaraRefCapabilityCatalog, "System capabilities of "+cfg.modelTitle, gemara.CapabilityCatalogArtifact),
	}

	// Groups from functional groups, plus a system group for the catch-all capability.
	groups := newGroupSet()
	for _, fg := range a.FunctionalGroups {
		groups.add(fg.ID, fallback(fg.Name, fg.ID), fallback(fg.Description, fallback(fg.Name, fg.ID)))
	}
	const sysGroup = "system"
	groups.add(sysGroup, "System", "Whole-system capability scope.")

	// Catch-all system capability so every threat can reference at least one capability.
	cat.Capabilities = append(cat.Capabilities, gemara.Capability{
		Id:          systemCapabilityID,
		Title:       cfg.modelTitle,
		Description: "The system as a whole: " + cfg.modelTitle + ".",
		Group:       sysGroup,
	})

	for _, fu := range a.FunctionalUnits {
		g := fu.Group
		if !groups.has(g) {
			g = sysGroup
		}
		cat.Capabilities = append(cat.Capabilities, gemara.Capability{
			Id:          fu.ID,
			Title:       fallback(fu.Name, fu.ID),
			Description: fallback(fu.Prose, fallback(fu.Name, fu.ID)),
			Group:       g,
		})
	}
	cat.Groups = groups.list()
	return cat
}

// -------------------- L2: Threat Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildThreatCatalog(bundle model.Bundle, cfg gemaraConfig) gemara.ThreatCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemara.ThreatCatalog{
		Title:    cfg.modelTitle + " - Threats",
		Metadata: cfg.newMetadata(gemaraRefThreatCatalog, "Threat scenarios for "+cfg.modelTitle, gemara.ThreatCatalogArtifact),
	}
	if len(a.ThreatScenarios) == 0 {
		return cat
	}

	// Valid capability ids (for capability references) and vector ids (for vector references).
	capIDs := map[string]bool{systemCapabilityID: true}
	for _, fu := range a.FunctionalUnits {
		capIDs[fu.ID] = true
	}
	vecIDs := map[string]bool{}
	for _, av := range a.AttackVectors {
		vecIDs[av.ID] = true
	}

	groups := newGroupSet()
	usedCapRef, usedVecRef := false, false

	for _, ts := range a.ThreatScenarios {
		gid := threatGroupID(ts)
		groups.add(gid, threatGroupTitle(ts), "Threat classification: "+threatGroupTitle(ts)+".")

		threat := gemara.Threat{
			Id:          ts.ID,
			Title:       fallback(ts.Title, ts.ID),
			Description: fallback(ts.Summary, fallback(ts.Title, ts.ID)),
			Group:       gid,
		}

		// capabilities (required, non-empty): map appliesTo functional units, else the system capability.
		var capEntries []gemara.ArtifactMapping
		for _, at := range ts.AppliesTo {
			if capIDs[at] {
				capEntries = append(capEntries, gemara.ArtifactMapping{ReferenceId: at})
			}
		}
		if len(capEntries) == 0 {
			capEntries = []gemara.ArtifactMapping{{ReferenceId: systemCapabilityID}}
		}
		threat.Capabilities = []gemara.MultiEntryMapping{{ReferenceId: gemaraRefCapabilityCatalog, Entries: capEntries}}
		usedCapRef = true

		// vectors (optional): map the attack-vector reference when it resolves.
		if ts.AttackVectorRef != "" && vecIDs[ts.AttackVectorRef] {
			threat.Vectors = []gemara.MultiEntryMapping{{
				ReferenceId: gemaraRefVectorCatalog,
				Entries:     []gemara.ArtifactMapping{{ReferenceId: ts.AttackVectorRef}},
			}}
			usedVecRef = true
		}

		// actors (optional): the relevant threat actors for this scenario.
		threat.Actors = threatActorsFor(ts)

		cat.Threats = append(cat.Threats, threat)
	}

	cat.Groups = groups.list()
	cat.Metadata.MappingReferences = mappingRefs(
		mappingRef(usedCapRef, gemaraRefCapabilityCatalog, "Capability Catalog"),
		mappingRef(usedVecRef, gemaraRefVectorCatalog, "Vector Catalog"),
	)
	return cat
}

// -------------------- L2: Control Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildControlCatalog(bundle model.Bundle, cfg gemaraConfig) gemara.ControlCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemara.ControlCatalog{
		Title:    cfg.modelTitle + " - Controls",
		Metadata: cfg.newMetadata(gemaraRefControlCatalog, "Security controls for "+cfg.modelTitle, gemara.ControlCatalogArtifact),
	}
	if len(a.Controls) == 0 {
		return cat
	}

	// Index control verifications by control id (source of assessment-requirements).
	verifsByControl := map[string][]model.ControlVerification{}
	for _, cv := range a.ControlVerifications {
		verifsByControl[cv.ControlRef] = append(verifsByControl[cv.ControlRef], cv)
	}
	// Index threat mitigations by control id (source of threat links).
	threatsByControl := map[string][]string{}
	for _, tm := range a.ThreatMitigations {
		if tm.ControlRef != "" && tm.ThreatScenarioRef != "" {
			threatsByControl[tm.ControlRef] = appendUnique(threatsByControl[tm.ControlRef], tm.ThreatScenarioRef)
		}
	}
	for _, ts := range a.ThreatScenarios {
		for _, ctrl := range ts.RelatedControls {
			threatsByControl[ctrl] = appendUnique(threatsByControl[ctrl], ts.ID)
		}
	}
	// Index control -> appliesTo functional units via compliance mappings.
	appliesByControl := map[string][]string{}
	for _, m := range bundle.Architecture.Compliance.Mappings {
		appliesByControl[m.ModelControlRef] = appendUniqueSlice(appliesByControl[m.ModelControlRef], m.AppliesTo)
	}

	// Applicability groups: functional units plus an all-systems catch-all.
	appGroups := newGroupSet()
	for _, fu := range a.FunctionalUnits {
		appGroups.add(fu.ID, fallback(fu.Name, fu.ID), fallback(fu.Prose, fallback(fu.Name, fu.ID)))
	}
	const allApplicability = "all-systems"
	appGroups.add(allApplicability, "All Systems", "Applies across the whole system.")
	cat.Metadata.ApplicabilityGroups = appGroups.list()

	groups := newGroupSet()
	usedThreatRef := false
	threatIDs := map[string]bool{}
	for _, ts := range a.ThreatScenarios {
		threatIDs[ts.ID] = true
	}

	for _, ctrl := range a.Controls {
		gid := controlGroupID(ctrl)
		groups.add(gid, controlGroupTitle(ctrl), "Control category: "+controlGroupTitle(ctrl)+".")

		// Resolve this control's applicability ids (valid applicability-group ids only).
		var applicability []string
		for _, at := range appliesByControl[ctrl.ID] {
			if appGroups.has(at) {
				applicability = appendUnique(applicability, at)
			}
		}
		if len(applicability) == 0 {
			applicability = []string{allApplicability}
		}

		c := gemara.Control{
			Id:        ctrl.ID,
			Title:     fallback(ctrl.Name, ctrl.ID),
			Objective: fallback(ctrl.Description, fallback(ctrl.Name, ctrl.ID)),
			Group:     gid,
			State:     gemara.LifecycleActive,
		}
		c.AssessmentRequirements = buildAssessmentRequirements(ctrl, verifsByControl[ctrl.ID], applicability)

		// guidelines (optional): link the control to its L1 guidance family.
		c.Guidelines = []gemara.MultiEntryMapping{{
			ReferenceId: gemaraRefGuidanceCatalog,
			Entries:     []gemara.ArtifactMapping{{ReferenceId: guidelineIDForCategory(ctrl.Category)}},
		}}

		// threats (optional): link mitigated threats to the threat catalog.
		var threatEntries []gemara.ArtifactMapping
		for _, tID := range threatsByControl[ctrl.ID] {
			if threatIDs[tID] {
				threatEntries = append(threatEntries, gemara.ArtifactMapping{ReferenceId: tID})
			}
		}
		if len(threatEntries) > 0 {
			c.Threats = []gemara.MultiEntryMapping{{ReferenceId: gemaraRefThreatCatalog, Entries: threatEntries}}
			usedThreatRef = true
		}

		cat.Controls = append(cat.Controls, c)
	}

	cat.Groups = groups.list()
	cat.Metadata.MappingReferences = mappingRefs(
		mappingRef(usedThreatRef, gemaraRefThreatCatalog, "Threat Catalog"),
		mappingRef(true, gemaraRefGuidanceCatalog, "Guidance Catalog"),
	)
	return cat
}

// buildAssessmentRequirements synthesizes verifiable requirements for a control.
// Each control verification becomes one assessment requirement; a control with no
// verifications gets a single default requirement derived from its description.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildAssessmentRequirements(ctrl model.Control, verifs []model.ControlVerification, applicability []string) []gemara.AssessmentRequirement {
	var reqs []gemara.AssessmentRequirement
	name := fallback(ctrl.Name, ctrl.ID)
	for _, cv := range verifs {
		method := fallback(cv.Method, "verification")
		text := fmt.Sprintf("The control %q MUST be verified by %s.", name, method)
		var rec string
		if len(cv.Findings) > 0 {
			rec = strings.Join(cv.Findings, "; ")
		}
		reqs = append(reqs, gemara.AssessmentRequirement{
			Id:             cv.ID,
			Text:           text,
			Applicability:  applicability,
			Recommendation: rec,
			State:          gemara.LifecycleActive,
		})
	}
	if len(reqs) == 0 {
		obj := fallback(ctrl.Description, name)
		reqs = append(reqs, gemara.AssessmentRequirement{
			Id:            ctrl.ID + "-AR-1",
			Text:          fmt.Sprintf("The system MUST implement the control %q: %s", name, obj),
			Applicability: applicability,
			State:         gemara.LifecycleActive,
		})
	}
	return reqs
}

// -------------------- L3: Risk Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildRiskCatalog(bundle model.Bundle, cfg gemaraConfig) gemara.RiskCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemara.RiskCatalog{
		Title:    cfg.modelTitle + " - Risks",
		Metadata: cfg.newMetadata(gemaraRefRiskCatalog, "Risk register for "+cfg.modelTitle, gemara.RiskCatalogArtifact),
	}
	if len(a.Risks) == 0 {
		return cat
	}

	const defaultCategory = "operational-risk"
	cat.Groups = []gemara.RiskCategory{{
		Id:          defaultCategory,
		Title:       "Operational Risk",
		Description: "Risks to the operation, security, and compliance of the system.",
		Appetite:    gemara.RiskAppetiteLow,
		MaxSeverity: gemara.SeverityHigh,
	}}

	threatIDs := map[string]bool{}
	for _, ts := range a.ThreatScenarios {
		threatIDs[ts.ID] = true
	}
	usedThreatRef := false
	rankByID := riskRankMap(a.Risks)

	for _, r := range a.Risks {
		risk := gemara.Risk{
			Id:          r.ID,
			Title:       fallback(r.Title, r.ID),
			Description: fallback(r.Statement, fallback(r.Title, r.ID)),
			Group:       defaultCategory,
			Severity:    deriveSeverity(r.Likelihood, r.Impact),
			Rank:        int64(rankByID[r.ID]),
			Impact:      strings.TrimSpace(r.Rationale),
		}
		if owner := strings.TrimSpace(r.Owner); owner != "" {
			contact := []gemara.Contact{{Name: owner}}
			risk.Owner = gemara.RACI{Responsible: contact, Accountable: contact, Informed: contact}
		}
		var threatEntries []gemara.ArtifactMapping
		for _, tID := range r.ThreatScenarios {
			if threatIDs[tID] {
				threatEntries = append(threatEntries, gemara.ArtifactMapping{ReferenceId: tID})
			}
		}
		if len(threatEntries) > 0 {
			risk.Threats = []gemara.MultiEntryMapping{{ReferenceId: gemaraRefThreatCatalog, Entries: threatEntries}}
			usedThreatRef = true
		}
		cat.Risks = append(cat.Risks, risk)
	}

	cat.Metadata.MappingReferences = mappingRefs(
		mappingRef(usedThreatRef, gemaraRefThreatCatalog, "Threat Catalog"),
	)
	return cat
}

// deriveSeverity collapses qualitative likelihood x impact into a Gemara severity.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func deriveSeverity(likelihood, impact string) gemara.Severity {
	l := levelScore(likelihood)
	i := levelScore(impact)
	switch {
	case l >= 3 && i >= 3:
		return gemara.SeverityCritical
	case l*i >= 6:
		return gemara.SeverityHigh
	case l*i >= 3:
		return gemara.SeverityMedium
	default:
		return gemara.SeverityLow
	}
}

// riskRankMap assigns each risk a unique rank (1 = highest), ordered by derived
// severity then id. Ranks must be unique among ranked risks per the schema.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func riskRankMap(risks []model.Risk) map[string]int {
	ordered := make([]model.Risk, len(risks))
	copy(ordered, risks)
	sevWeight := func(r model.Risk) int {
		switch deriveSeverity(r.Likelihood, r.Impact) {
		case gemara.SeverityCritical:
			return 0
		case gemara.SeverityHigh:
			return 1
		case gemara.SeverityMedium:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if wi, wj := sevWeight(ordered[i]), sevWeight(ordered[j]); wi != wj {
			return wi < wj
		}
		return ordered[i].ID < ordered[j].ID
	})
	out := map[string]int{}
	for i, r := range ordered {
		out[r.ID] = i + 1
	}
	return out
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func levelScore(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical", "very high", "veryhigh":
		return 4
	case "high":
		return 3
	case "medium", "moderate", "med":
		return 2
	case "low", "very low", "verylow", "minimal":
		return 1
	default:
		return 2 // unknown -> medium
	}
}

// -------------------- shared helpers --------------------

// groupSet collects unique groups in insertion order.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type groupSet struct {
	seen   map[string]bool
	groups []gemara.Group
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func newGroupSet() *groupSet { return &groupSet{seen: map[string]bool{}} }

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (g *groupSet) add(id, title, desc string) {
	// ids are treated as opaque: callers pass either authored model ids (e.g.
	// FG-PAYMENTS) or pre-canonicalized derived ids (e.g. stride-tampering), and
	// references must use the same form, so no slugging happens here.
	if id == "" || g.seen[id] {
		return
	}
	g.seen[id] = true
	g.groups = append(g.groups, gemara.Group{Id: id, Title: fallback(title, id), Description: fallback(desc, fallback(title, id))})
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (g *groupSet) has(id string) bool { return g.seen[id] }

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (g *groupSet) list() []gemara.Group {
	return g.groups
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatGroupID(ts model.ThreatScenario) string {
	if s := strings.TrimSpace(ts.Stride); s != "" {
		return "stride-" + slug(s)
	}
	if c := strings.TrimSpace(ts.Category); c != "" {
		return slug(c)
	}
	return "threat"
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatGroupTitle(ts model.ThreatScenario) string {
	if s := strings.TrimSpace(ts.Stride); s != "" {
		return s
	}
	if c := strings.TrimSpace(ts.Category); c != "" {
		return c
	}
	return "Threat"
}

// threatActorsFor returns the relevant threat actors for a scenario, derived from
// its STRIDE/category classification.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatActorsFor(ts model.ThreatScenario) []gemara.Actor {
	actors := []gemara.Actor{
		{Id: "external-attacker", Name: "External Attacker", Type: gemara.Human, Description: "An unauthenticated adversary outside the trust boundary."},
	}
	tag := strings.ToLower(ts.Stride + " " + ts.Category + " " + ts.Title)
	if strings.Contains(tag, "eleva") || strings.Contains(tag, "tamper") || strings.Contains(tag, "repudiat") || strings.Contains(tag, "insider") {
		actors = append(actors, gemara.Actor{Id: "malicious-insider", Name: "Malicious Insider", Type: gemara.Human, Description: "An authorized party abusing legitimate access."})
	}
	if strings.Contains(tag, "supply") || strings.Contains(tag, "depend") || strings.Contains(tag, "image") || strings.Contains(tag, "package") {
		actors = append(actors, gemara.Actor{Id: "compromised-dependency", Name: "Compromised Dependency", Type: gemara.Software, Description: "A malicious or compromised third-party component."})
	}
	return actors
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func controlGroupID(c model.Control) string {
	if cat := strings.TrimSpace(c.Category); cat != "" {
		return slug(cat)
	}
	return "general"
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func controlGroupTitle(c model.Control) string {
	if cat := strings.TrimSpace(c.Category); cat != "" {
		return cat
	}
	return "General"
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mappingRef(used bool, id, title string) *gemara.MappingReference {
	if !used {
		return nil
	}
	return &gemara.MappingReference{Id: id, Title: title, Version: GemaraVersion}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mappingRefs(refs ...*gemara.MappingReference) []gemara.MappingReference {
	var out []gemara.MappingReference
	for _, r := range refs {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}

// slug normalizes an identifier into a lowercase, dash-separated token.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func slug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func fallback(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// sortedKeys returns map keys in deterministic order (used by serialization helpers).
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
