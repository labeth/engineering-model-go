// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

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

	gemaraproj "github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/export/exportutil"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// Version is the Gemara specification version stamped into every artifact's metadata.
const Version = "1.1.0"

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

// Options configures the deterministic metadata stamped into artifacts.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type Options struct {
	AuthorID   string // metadata.author.id (default "engmod")
	AuthorName string // metadata.author.name (default "Engineering Model")
	// AuthorType is metadata.author.type. The default author is this generator,
	// so the zero value renders Software; set Human when a person signs.
	AuthorType         gemaraproj.EntityType
	Version            string // metadata.version (optional)
	Date               string // metadata.date, ISO 8601 (optional; omitted when empty for reproducibility)
	AssessmentPlanHref string // OSCAL Assessment Results import-ap href
}

// Result holds the typed Gemara documents and their YAML serializations.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type Result struct {
	ControlCatalog    gemaraproj.ControlCatalog
	ThreatCatalog     gemaraproj.ThreatCatalog
	RiskCatalog       gemaraproj.RiskCatalog
	VectorCatalog     gemaraproj.VectorCatalog
	CapabilityCatalog gemaraproj.CapabilityCatalog

	// Extended artifacts (emitted only when the model has supporting data).
	PrincipleCatalog     gemaraproj.PrincipleCatalog
	GuidanceCatalog      gemaraproj.GuidanceCatalog
	Policy               gemaraproj.Policy
	Lexicon              gemaraproj.Lexicon
	ControlThreatMapping gemaraproj.MappingDocument
	AuditLog             gemaraproj.AuditLog
	EnforcementLog       gemaraproj.EnforcementLog
	HasGuidance          bool
	HasPolicy            bool
	HasMapping           bool
	HasAudit             bool
	HasEnforcement       bool

	// YAML maps an artifact short-name (e.g. "control-catalog") to its serialized document.
	YAML map[string]string

	// Diagnostics reports values Gemara requires that the model does not state
	// (rendered as "not set") so they are not mistaken for authored content.
	Diagnostics []validate.Diagnostic
}

// GenerateFromFile loads the model bundle and renders the Gemara catalogs.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateFromFile(architecturePath string, options Options) (Result, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "behavior", "assurance", "compliance")
	if err != nil {
		return Result{}, err
	}
	return Generate(bundle, options)
}

// Generate renders the engineering model bundle into Gemara L1-L3 documents.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE, DO-GEMARA-DOCUMENTS
func Generate(bundle model.Bundle, options Options) (Result, error) {
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return Result{}, err
	}
	bundle = canonical.Documents()
	options, err = resolveGemaraOptions(bundle, options)
	if err != nil {
		return Result{}, err
	}
	cfg := newGemaraConfig(bundle, options)
	var diags []validate.Diagnostic
	cfg.diags = &diags

	res := Result{
		VectorCatalog:     buildVectorCatalog(bundle, cfg),
		CapabilityCatalog: buildCapabilityCatalog(bundle, cfg),
		ControlCatalog:    buildControlCatalog(bundle, cfg),
		ThreatCatalog:     buildThreatCatalog(bundle, cfg),
		RiskCatalog:       buildRiskCatalog(bundle, cfg),
	}
	docs := []gemaraDoc{
		{"vector-catalog", res.VectorCatalog},
		{"capability-catalog", res.CapabilityCatalog},
		{"control-catalog", res.ControlCatalog},
		{"threat-catalog", res.ThreatCatalog},
		{"risk-catalog", res.RiskCatalog},
	}
	docs = append(docs, res.addPolicyLayer(bundle, cfg, options.Date)...)
	docs = append(docs, res.addDataDerived(bundle, cfg, options.Date)...)

	res.YAML, err = marshalGemaraDocs(docs)
	if err != nil {
		return Result{}, err
	}
	res.Diagnostics = validate.SortDiagnostics(diags)
	return res, nil
}

// gemaraDoc is one rendered artifact: its short name and its document.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type gemaraDoc struct {
	name string
	doc  any
}

// addPolicyLayer fills the L1 Principle and Guidance catalogs and the L3
// Policy and returns their documents.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (r *Result) addPolicyLayer(bundle model.Bundle, cfg gemaraConfig, date string) []gemaraDoc {
	r.PrincipleCatalog = buildPrincipleCatalog(bundle, cfg)
	docs := []gemaraDoc{{"principle-catalog", r.PrincipleCatalog}}
	if gc, ok := buildGuidanceCatalog(bundle, cfg); ok {
		r.GuidanceCatalog, r.HasGuidance = gc, true
		docs = append(docs, gemaraDoc{"guidance-catalog", gc})
	}
	if pol, ok := buildPolicy(bundle, cfg, date); ok {
		r.Policy, r.HasPolicy = pol, true
		docs = append(docs, gemaraDoc{"policy", pol})
	}
	return docs
}

// addDataDerived fills the extended artifacts that exist only when the model
// has supporting data and returns their documents.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (r *Result) addDataDerived(bundle model.Bundle, cfg gemaraConfig, date string) []gemaraDoc {
	var docs []gemaraDoc
	if lex, ok := buildLexicon(bundle, cfg); ok {
		r.Lexicon = lex
		docs = append(docs, gemaraDoc{"lexicon", lex})
	}
	if md, ok := buildControlThreatMapping(bundle, cfg); ok {
		r.ControlThreatMapping, r.HasMapping = md, true
		docs = append(docs, gemaraDoc{"control-threat-mapping", md})
	}
	if al, ok := buildAuditLog(bundle, cfg, date); ok {
		r.AuditLog, r.HasAudit = al, true
		docs = append(docs, gemaraDoc{"audit-log", al})
	}
	if el, ok := buildEnforcementLog(bundle, cfg, date); ok {
		r.EnforcementLog, r.HasEnforcement = el, true
		docs = append(docs, gemaraDoc{"enforcement-log", el})
	}
	return docs
}

// marshalGemaraDocs serializes every document under its short name.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func marshalGemaraDocs(docs []gemaraDoc) (map[string]string, error) {
	out := make(map[string]string, len(docs))
	for _, d := range docs {
		text, err := marshalGemara(d.doc)
		if err != nil {
			return nil, fmt.Errorf("marshal gemara %s: %w", d.name, err)
		}
		out[d.name] = text
	}
	return out, nil
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func resolveGemaraOptions(bundle model.Bundle, options Options) (Options, error) {
	if len(bundle.Architecture.AuthoredArchitecture.Controls) > 0 ||
		len(bundle.Architecture.AuthoredArchitecture.ControlVerifications) > 0 ||
		len(bundle.Architecture.AuthoredArchitecture.POAMItems) > 0 {
		resolved, err := exportutil.ResolveGeneratedTimestamp(bundle, options.Date, "Gemara")
		if err != nil {
			return options, err
		}
		options.Date = resolved
	} else if raw := strings.TrimSpace(options.Date); raw != "" {
		resolved, err := exportutil.ResolveGeneratedTimestamp(bundle, raw, "Gemara")
		if err != nil {
			return options, err
		}
		options.Date = resolved
	}
	return options, nil
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
	options    Options
	modelID    string
	modelTitle string
	author     gemaraproj.Actor
	// diags collects export diagnostics; nil discards them.
	diags *[]validate.Diagnostic
}

// addDiagnostic records an export diagnostic when a sink is attached.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c gemaraConfig) addDiagnostic(d validate.Diagnostic) {
	if c.diags != nil {
		*c.diags = append(*c.diags, d)
	}
}

// policyID is the metadata id of the rendered Policy; artifacts that cite the
// policy use it as their mapping-reference id so the reference resolves.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c gemaraConfig) policyID() string { return c.modelID + "-POLICY" }

// evaluationLogID is the metadata id of the rendered Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c gemaraConfig) evaluationLogID() string { return c.modelID + "-EVALUATION-LOG" }

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func newGemaraConfig(bundle model.Bundle, options Options) gemaraConfig {
	if strings.TrimSpace(options.AuthorID) == "" {
		options.AuthorID = "engmod"
	}
	if strings.TrimSpace(options.AuthorName) == "" {
		options.AuthorName = "Engineering Model"
	}
	if options.AuthorType == gemaraproj.InvalidEntityType {
		options.AuthorType = gemaraproj.Software
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
		author: gemaraproj.Actor{
			Id:   slug(options.AuthorID),
			Name: options.AuthorName,
			Type: options.AuthorType,
		},
	}
}

// newMetadata builds a valid #Metadata block for a catalog/log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (c gemaraConfig) newMetadata(id, description string, t gemaraproj.ArtifactType) gemaraproj.Metadata {
	m := gemaraproj.Metadata{
		Id:            id,
		Type:          t,
		GemaraVersion: Version,
		Description:   textutil.Fallback(description, id),
		Author:        c.author,
		Version:       strings.TrimSpace(c.options.Version),
	}
	if d := normalizeDatetime(c.options.Date); d != "" {
		m.Date = gemaraproj.Datetime(d)
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
func buildVectorCatalog(bundle model.Bundle, cfg gemaraConfig) gemaraproj.VectorCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemaraproj.VectorCatalog{
		Title:    cfg.modelTitle + " - Attack Vectors",
		Metadata: cfg.newMetadata(gemaraRefVectorCatalog, "Attack vectors relevant to "+cfg.modelTitle, gemaraproj.VectorCatalogArtifact),
	}
	if len(a.AttackVectors) == 0 {
		return cat
	}
	group := gemaraproj.Group{Id: "attack-vector", Title: "Attack Vector", Description: "Methods and pathways through which a threat may be realized."}
	cat.Groups = []gemaraproj.Group{group}

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
		cat.Vectors = append(cat.Vectors, gemaraproj.Vector{
			Id:            av.ID,
			Title:         textutil.Fallback(av.Name, av.ID),
			Description:   textutil.Fallback(av.Description, textutil.Fallback(av.Name, av.ID)),
			Group:         group.Id,
			Applicability: []string{allEnv},
		})
	}
	return cat
}

// -------------------- L2: Capability Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildCapabilityCatalog(bundle model.Bundle, cfg gemaraConfig) gemaraproj.CapabilityCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemaraproj.CapabilityCatalog{
		Title:    cfg.modelTitle + " - Capabilities",
		Metadata: cfg.newMetadata(gemaraRefCapabilityCatalog, "System capabilities of "+cfg.modelTitle, gemaraproj.CapabilityCatalogArtifact),
	}

	// Groups from functional groups, plus a system group for the catch-all capability.
	groups := newGroupSet()
	for _, fg := range a.FunctionalGroups {
		groups.add(fg.ID, textutil.Fallback(fg.Name, fg.ID), textutil.Fallback(fg.Description, textutil.Fallback(fg.Name, fg.ID)))
	}
	const sysGroup = "system"
	groups.add(sysGroup, "System", "Whole-system capability scope.")

	// Catch-all system capability so every threat can reference at least one capability.
	cat.Capabilities = append(cat.Capabilities, gemaraproj.Capability{
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
		cat.Capabilities = append(cat.Capabilities, gemaraproj.Capability{
			Id:          fu.ID,
			Title:       textutil.Fallback(fu.Name, fu.ID),
			Description: textutil.Fallback(fu.Prose, textutil.Fallback(fu.Name, fu.ID)),
			Group:       g,
		})
	}
	cat.Groups = groups.list()
	return cat
}

// -------------------- L2: Threat Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildThreatCatalog(bundle model.Bundle, cfg gemaraConfig) gemaraproj.ThreatCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemaraproj.ThreatCatalog{
		Title:    cfg.modelTitle + " - Threats",
		Metadata: cfg.newMetadata(gemaraRefThreatCatalog, "Threat scenarios for "+cfg.modelTitle, gemaraproj.ThreatCatalogArtifact),
	}
	if len(a.ThreatScenarios) == 0 {
		return cat
	}

	// Valid capability ids (for capability references) and vector ids (for vector references).
	idx := threatIndex{capIDs: map[string]bool{systemCapabilityID: true}, vecIDs: map[string]bool{}}
	for _, fu := range a.FunctionalUnits {
		idx.capIDs[fu.ID] = true
	}
	for _, av := range a.AttackVectors {
		idx.vecIDs[av.ID] = true
	}

	groups := newGroupSet()
	usedVecRef := false
	for _, ts := range a.ThreatScenarios {
		gid := threatGroupID(ts)
		groups.add(gid, threatGroupTitle(ts), "Threat classification: "+threatGroupTitle(ts)+".")
		threat, usedVec := idx.buildThreat(ts, gid)
		usedVecRef = usedVecRef || usedVec
		cat.Threats = append(cat.Threats, threat)
	}

	cat.Groups = groups.list()
	cat.Metadata.MappingReferences = mappingRefs(
		mappingRef(true, gemaraRefCapabilityCatalog, "Capability Catalog"),
		mappingRef(usedVecRef, gemaraRefVectorCatalog, "Vector Catalog"),
	)
	return cat
}

// threatIndex holds the ids a threat can reference.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type threatIndex struct {
	capIDs map[string]bool
	vecIDs map[string]bool
}

// buildThreat renders one threat scenario and reports whether it references the
// vector catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (idx threatIndex) buildThreat(ts model.ThreatScenario, gid string) (gemaraproj.Threat, bool) {
	threat := gemaraproj.Threat{
		Id:          ts.ID,
		Title:       textutil.Fallback(ts.Title, ts.ID),
		Description: textutil.Fallback(ts.Summary, textutil.Fallback(ts.Title, ts.ID)),
		Group:       gid,
	}

	// capabilities (required, non-empty): map appliesTo functional units, else the system capability.
	var capEntries []gemaraproj.ArtifactMapping
	for _, at := range ts.AppliesTo {
		if idx.capIDs[at] {
			capEntries = append(capEntries, gemaraproj.ArtifactMapping{ReferenceId: at})
		}
	}
	if len(capEntries) == 0 {
		capEntries = []gemaraproj.ArtifactMapping{{ReferenceId: systemCapabilityID}}
	}
	threat.Capabilities = []gemaraproj.MultiEntryMapping{{ReferenceId: gemaraRefCapabilityCatalog, Entries: capEntries}}

	// actors (optional): the relevant threat actors for this scenario.
	threat.Actors = threatActorsFor(ts)

	// vectors (optional): map the attack-vector reference when it resolves.
	if ts.AttackVectorRef == "" || !idx.vecIDs[ts.AttackVectorRef] {
		return threat, false
	}
	threat.Vectors = []gemaraproj.MultiEntryMapping{{
		ReferenceId: gemaraRefVectorCatalog,
		Entries:     []gemaraproj.ArtifactMapping{{ReferenceId: ts.AttackVectorRef}},
	}}
	return threat, true
}

// -------------------- L2: Control Catalog --------------------

// allSystemsApplicability is the catch-all applicability group of control artifacts.
const allSystemsApplicability = "all-systems"

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildControlCatalog(bundle model.Bundle, cfg gemaraConfig) gemaraproj.ControlCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemaraproj.ControlCatalog{
		Title:    cfg.modelTitle + " - Controls",
		Metadata: cfg.newMetadata(gemaraRefControlCatalog, "Security controls for "+cfg.modelTitle, gemaraproj.ControlCatalogArtifact),
	}
	if len(a.Controls) == 0 {
		return cat
	}

	idx := newControlIndex(bundle)
	cat.Metadata.ApplicabilityGroups = idx.appGroups.list()

	groups := newGroupSet()
	usedThreatRef := false
	for _, ctrl := range a.Controls {
		gid := controlGroupID(ctrl)
		groups.add(gid, controlGroupTitle(ctrl), "Control category: "+controlGroupTitle(ctrl)+".")
		c, usedThreat := idx.buildControl(ctrl, gid)
		usedThreatRef = usedThreatRef || usedThreat
		cat.Controls = append(cat.Controls, c)
	}

	cat.Groups = groups.list()
	cat.Metadata.MappingReferences = mappingRefs(
		mappingRef(usedThreatRef, gemaraRefThreatCatalog, "Threat Catalog"),
		mappingRef(true, gemaraRefGuidanceCatalog, "Guidance Catalog"),
	)
	return cat
}

// controlIndex holds the lookups the controls of a catalog are built from.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type controlIndex struct {
	verifs    map[string][]model.ControlVerification
	threats   map[string][]string
	applies   map[string][]string
	appGroups *groupSet
	threatIDs map[string]bool
}

// newControlIndex indexes control verifications (source of
// assessment-requirements), threat mitigations (source of threat links) and
// compliance mappings (source of applicability) by control id.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func newControlIndex(bundle model.Bundle) controlIndex {
	a := bundle.Architecture.AuthoredArchitecture
	applies := map[string][]string{}
	for _, m := range bundle.Architecture.Compliance.Mappings {
		applies[m.ModelControlRef] = textutil.AppendUniqueSlice(applies[m.ModelControlRef], m.AppliesTo)
	}
	return controlIndex{
		verifs:    groupVerificationsByControl(a.ControlVerifications),
		threats:   threatsByControl(a),
		applies:   applies,
		appGroups: controlApplicabilityGroups(a),
		threatIDs: threatIDSet(a.ThreatScenarios),
	}
}

// threatsByControl lists the threat scenarios each control mitigates.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatsByControl(a model.AuthoredArchitecture) map[string][]string {
	byControl := map[string][]string{}
	for _, tm := range a.ThreatMitigations {
		addThreatMitigation(byControl, tm)
	}
	for _, ts := range a.ThreatScenarios {
		addRelatedControls(byControl, ts)
	}
	return byControl
}

// addRelatedControls lists the scenario under every control it names as related.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func addRelatedControls(byControl map[string][]string, ts model.ThreatScenario) {
	for _, ctrl := range ts.RelatedControls {
		byControl[ctrl] = textutil.AppendUnique(byControl[ctrl], ts.ID)
	}
}

// addThreatMitigation records the threat of a mitigation that names both ends.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func addThreatMitigation(byControl map[string][]string, tm model.ThreatMitigation) {
	if tm.ControlRef == "" || tm.ThreatScenarioRef == "" {
		return
	}
	byControl[tm.ControlRef] = textutil.AppendUnique(byControl[tm.ControlRef], tm.ThreatScenarioRef)
}

// controlApplicabilityGroups are the functional units plus an all-systems catch-all.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func controlApplicabilityGroups(a model.AuthoredArchitecture) *groupSet {
	appGroups := newGroupSet()
	for _, fu := range a.FunctionalUnits {
		appGroups.add(fu.ID, textutil.Fallback(fu.Name, fu.ID), textutil.Fallback(fu.Prose, textutil.Fallback(fu.Name, fu.ID)))
	}
	appGroups.add(allSystemsApplicability, "All Systems", "Applies across the whole system.")
	return appGroups
}

// threatIDSet is the set of threat scenario ids.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatIDSet(scenarios []model.ThreatScenario) map[string]bool {
	ids := map[string]bool{}
	for _, ts := range scenarios {
		ids[ts.ID] = true
	}
	return ids
}

// threatEntries maps the referenced ids that exist to artifact mappings.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatEntries(refs []string, known map[string]bool) []gemaraproj.ArtifactMapping {
	var entries []gemaraproj.ArtifactMapping
	for _, id := range refs {
		if known[id] {
			entries = append(entries, gemaraproj.ArtifactMapping{ReferenceId: id})
		}
	}
	return entries
}

// applicability resolves a control's applicability ids (valid applicability-group
// ids only), the all-systems catch-all when none resolves.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (idx controlIndex) applicability(controlID string) []string {
	var applicability []string
	for _, at := range idx.applies[controlID] {
		if idx.appGroups.has(at) {
			applicability = textutil.AppendUnique(applicability, at)
		}
	}
	if len(applicability) == 0 {
		return []string{allSystemsApplicability}
	}
	return applicability
}

// buildControl renders one control and reports whether it links the threat catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (idx controlIndex) buildControl(ctrl model.Control, gid string) (gemaraproj.Control, bool) {
	c := gemaraproj.Control{
		Id:        ctrl.ID,
		Title:     textutil.Fallback(ctrl.Name, ctrl.ID),
		Objective: textutil.Fallback(ctrl.Description, textutil.Fallback(ctrl.Name, ctrl.ID)),
		Group:     gid,
		State:     gemaraproj.LifecycleActive,
	}
	c.AssessmentRequirements = buildAssessmentRequirements(ctrl, idx.verifs[ctrl.ID], idx.applicability(ctrl.ID))

	// guidelines (optional): link the control to its L1 guidance family.
	c.Guidelines = []gemaraproj.MultiEntryMapping{{
		ReferenceId: gemaraRefGuidanceCatalog,
		Entries:     []gemaraproj.ArtifactMapping{{ReferenceId: guidelineIDForCategory(ctrl.Category)}},
	}}

	// threats (optional): link mitigated threats to the threat catalog.
	entries := threatEntries(idx.threats[ctrl.ID], idx.threatIDs)
	if len(entries) == 0 {
		return c, false
	}
	c.Threats = []gemaraproj.MultiEntryMapping{{ReferenceId: gemaraRefThreatCatalog, Entries: entries}}
	return c, true
}

// buildAssessmentRequirements synthesizes verifiable requirements for a control.
// Each control verification becomes one assessment requirement; a control with no
// verifications gets a single default requirement derived from its description.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildAssessmentRequirements(ctrl model.Control, verifs []model.ControlVerification, applicability []string) []gemaraproj.AssessmentRequirement {
	var reqs []gemaraproj.AssessmentRequirement
	name := textutil.Fallback(ctrl.Name, ctrl.ID)
	for _, cv := range verifs {
		method := textutil.Fallback(cv.Method, "verification")
		text := fmt.Sprintf("The control %q MUST be verified by %s.", name, method)
		var rec string
		if len(cv.Findings) > 0 {
			rec = strings.Join(cv.Findings, "; ")
		}
		reqs = append(reqs, gemaraproj.AssessmentRequirement{
			Id:             cv.ID,
			Text:           text,
			Applicability:  applicability,
			Recommendation: rec,
			State:          gemaraproj.LifecycleActive,
		})
	}
	if len(reqs) == 0 {
		obj := textutil.Fallback(ctrl.Description, name)
		reqs = append(reqs, gemaraproj.AssessmentRequirement{
			Id:            ctrl.ID + "-AR-1",
			Text:          fmt.Sprintf("The system MUST implement the control %q: %s", name, obj),
			Applicability: applicability,
			State:         gemaraproj.LifecycleActive,
		})
	}
	return reqs
}

// -------------------- L3: Risk Catalog --------------------

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildRiskCatalog(bundle model.Bundle, cfg gemaraConfig) gemaraproj.RiskCatalog {
	a := bundle.Architecture.AuthoredArchitecture
	cat := gemaraproj.RiskCatalog{
		Title:    cfg.modelTitle + " - Risks",
		Metadata: cfg.newMetadata(gemaraRefRiskCatalog, "Risk register for "+cfg.modelTitle, gemaraproj.RiskCatalogArtifact),
	}
	if len(a.Risks) == 0 {
		return cat
	}

	cat.Groups = []gemaraproj.RiskCategory{{
		Id:          defaultRiskCategory,
		Title:       "Operational Risk",
		Description: "Risks to the operation, security, and compliance of the system. Risk appetite not set in the model; default applied.",
		// appetite is required by the schema; the model has no appetite field,
		// so a default is applied and disclosed above. max-severity is optional
		// and stays unset.
		Appetite: gemaraproj.RiskAppetiteLow,
	}}

	threatIDs := threatIDSet(a.ThreatScenarios)
	usedThreatRef := false
	rankByID := riskRankMap(a.Risks)
	for _, r := range a.Risks {
		risk, usedThreat := buildRisk(r, rankByID[r.ID], threatIDs)
		usedThreatRef = usedThreatRef || usedThreat
		cat.Risks = append(cat.Risks, risk)
	}

	cat.Metadata.MappingReferences = mappingRefs(
		mappingRef(usedThreatRef, gemaraRefThreatCatalog, "Threat Catalog"),
	)
	return cat
}

// defaultRiskCategory is the one risk group of the catalog.
const defaultRiskCategory = "operational-risk"

// buildRisk renders one risk and reports whether it links the threat catalog.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildRisk(r model.Risk, rank int, threatIDs map[string]bool) (gemaraproj.Risk, bool) {
	risk := gemaraproj.Risk{
		Id:          r.ID,
		Title:       textutil.Fallback(r.Title, r.ID),
		Description: textutil.Fallback(r.Statement, textutil.Fallback(r.Title, r.ID)),
		Group:       defaultRiskCategory,
		Severity:    deriveSeverity(r.Likelihood, r.Impact),
		Rank:        int64(rank),
		Impact:      strings.TrimSpace(r.Rationale),
	}
	if owner := strings.TrimSpace(r.Owner); owner != "" {
		contact := []gemaraproj.Contact{{Name: owner}}
		risk.Owner = gemaraproj.RACI{Responsible: contact, Accountable: contact, Informed: contact}
	}
	entries := threatEntries(r.ThreatScenarios, threatIDs)
	if len(entries) == 0 {
		return risk, false
	}
	risk.Threats = []gemaraproj.MultiEntryMapping{{ReferenceId: gemaraRefThreatCatalog, Entries: entries}}
	return risk, true
}

// deriveSeverity collapses qualitative likelihood x impact into a Gemara severity.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func deriveSeverity(likelihood, impact string) gemaraproj.Severity {
	l := levelScore(likelihood)
	i := levelScore(impact)
	switch {
	case l >= 3 && i >= 3:
		return gemaraproj.SeverityCritical
	case l*i >= 6:
		return gemaraproj.SeverityHigh
	case l*i >= 3:
		return gemaraproj.SeverityMedium
	default:
		return gemaraproj.SeverityLow
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
		case gemaraproj.SeverityCritical:
			return 0
		case gemaraproj.SeverityHigh:
			return 1
		case gemaraproj.SeverityMedium:
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
	groups []gemaraproj.Group
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
	g.groups = append(g.groups, gemaraproj.Group{Id: id, Title: textutil.Fallback(title, id), Description: textutil.Fallback(desc, textutil.Fallback(title, id))})
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (g *groupSet) has(id string) bool { return g.seen[id] }

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (g *groupSet) list() []gemaraproj.Group {
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

// threatActorRule adds an actor to a scenario whose classification text
// contains one of the keywords.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type threatActorRule struct {
	keywords []string
	actor    gemaraproj.Actor
}

var threatActorRules = []threatActorRule{
	{
		keywords: []string{"eleva", "tamper", "repudiat", "insider"},
		actor:    gemaraproj.Actor{Id: "malicious-insider", Name: "Malicious Insider", Type: gemaraproj.Human, Description: "An authorized party abusing legitimate access."},
	},
	{
		keywords: []string{"supply", "depend", "image", "package"},
		actor:    gemaraproj.Actor{Id: "compromised-dependency", Name: "Compromised Dependency", Type: gemaraproj.Software, Description: "A malicious or compromised third-party component."},
	},
}

// threatActorsFor returns the relevant threat actors for a scenario, derived from
// its STRIDE/category classification.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func threatActorsFor(ts model.ThreatScenario) []gemaraproj.Actor {
	actors := []gemaraproj.Actor{
		{Id: "external-attacker", Name: "External Attacker", Type: gemaraproj.Human, Description: "An unauthenticated adversary outside the trust boundary."},
	}
	tag := strings.ToLower(ts.Stride + " " + ts.Category + " " + ts.Title)
	for _, rule := range threatActorRules {
		if containsAny(tag, rule.keywords) {
			actors = append(actors, rule.actor)
		}
	}
	return actors
}

// containsAny reports whether text contains one of the substrings.
func containsAny(text string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
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
func mappingRef(used bool, id, title string) *gemaraproj.MappingReference {
	if !used {
		return nil
	}
	return &gemaraproj.MappingReference{Id: id, Title: title, Version: Version}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mappingRefs(refs ...*gemaraproj.MappingReference) []gemaraproj.MappingReference {
	var out []gemaraproj.MappingReference
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
		case isASCIILower(r) || isASCIIDigit(r):
			b.WriteRune(r)
			prevDash = false
		case strings.ContainsRune("-_ ./", r) && !prevDash && b.Len() > 0:
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// isASCIILower reports whether r is in a-z.
func isASCIILower(r rune) bool { return r >= 'a' && r <= 'z' }

// isASCIIDigit reports whether r is in 0-9.
func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }
