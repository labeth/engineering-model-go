package engmodel

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/labeth/engineering-model-go/model"
)

//go:embed templates/trlc.model.rsl.tmpl
var trlcModelTemplateText string

//go:embed templates/trlc.requirements.trlc.tmpl
var trlcRequirementsTemplateText string

var trlcModelTemplate = template.Must(template.New("trlc-model").Parse(trlcModelTemplateText))
var trlcRequirementsTemplate = template.Must(template.New("trlc-requirements").Parse(trlcRequirementsTemplateText))

type TRLCExportOptions struct {
	PackageName string
}

type TRLCExportResult struct {
	PackageName      string
	ModelRSL         string
	RequirementsTRLC string
}

type trlcTemplateData struct {
	PackageName  string
	Requirements []trlcRequirementRecord
}

type trlcRequirementRecord struct {
	ObjectName string
	ID         string
	Text       string
	Notes      string
	HasNotes   bool
	AppliesTo  []string
}

func GenerateTRLCRequirementsFromFile(requirementsPath string, options TRLCExportOptions) (TRLCExportResult, error) {
	reqDoc, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return TRLCExportResult{}, err
	}
	pkg := strings.TrimSpace(options.PackageName)
	if pkg == "" {
		base := strings.TrimSuffix(filepath.Base(requirementsPath), filepath.Ext(requirementsPath))
		pkg = sanitizeTRLCIdentifier(base)
		if pkg == "" {
			pkg = "Requirements"
		}
	}
	return GenerateTRLCRequirements(reqDoc, TRLCExportOptions{PackageName: pkg})
}

func GenerateTRLCRequirements(requirements model.RequirementsDocument, options TRLCExportOptions) (TRLCExportResult, error) {
	pkg := sanitizeTRLCIdentifier(strings.TrimSpace(options.PackageName))
	if pkg == "" {
		pkg = sanitizeTRLCIdentifier(strings.TrimSpace(requirements.LintRun.ID))
	}
	if pkg == "" {
		pkg = "Requirements"
	}

	used := map[string]bool{}
	reqs := make([]trlcRequirementRecord, 0, len(requirements.Requirements))
	for _, r := range requirements.Requirements {
		obj := uniqueTRLCObjectName(sanitizeTRLCIdentifier(strings.TrimSpace(r.ID)), used)
		if obj == "" {
			obj = uniqueTRLCObjectName("requirement", used)
		}
		applies := append([]string(nil), r.AppliesTo...)
		sort.Strings(applies)
		reqs = append(reqs, trlcRequirementRecord{
			ObjectName: obj,
			ID:         strings.TrimSpace(r.ID),
			Text:       sanitizeTRLCString(strings.TrimSpace(r.Text)),
			Notes:      sanitizeTRLCString(strings.TrimSpace(r.Notes)),
			HasNotes:   strings.TrimSpace(r.Notes) != "",
			AppliesTo:  applies,
		})
	}
	sort.SliceStable(reqs, func(i, j int) bool { return reqs[i].ID < reqs[j].ID })

	tplData := trlcTemplateData{PackageName: pkg, Requirements: reqs}
	modelRSL, err := executeTRLCTemplate(trlcModelTemplate, tplData)
	if err != nil {
		return TRLCExportResult{}, err
	}
	reqTRLC, err := executeTRLCTemplate(trlcRequirementsTemplate, tplData)
	if err != nil {
		return TRLCExportResult{}, err
	}

	return TRLCExportResult{PackageName: pkg, ModelRSL: modelRSL, RequirementsTRLC: reqTRLC}, nil
}

func executeTRLCTemplate(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("execute TRLC template: %w", err)
	}
	return b.String(), nil
}

func sanitizeTRLCIdentifier(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	b := strings.Builder{}
	for i, r := range in {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit || r == '_' {
			if i == 0 && isDigit {
				b.WriteByte('R')
				b.WriteByte('_')
			}
			if r == '-' {
				b.WriteByte('_')
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '-' || r == ' ' || r == '.' || r == '/' {
			b.WriteByte('_')
			continue
		}
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if out == "" {
		return ""
	}
	return out
}

func sanitizeTRLCString(in string) string {
	in = strings.ReplaceAll(in, "\\", "\\\\")
	in = strings.ReplaceAll(in, "\"", "\\\"")
	in = strings.ReplaceAll(in, "\r", " ")
	in = strings.ReplaceAll(in, "\n", " ")
	in = strings.Join(strings.Fields(in), " ")
	return strings.TrimSpace(in)
}

func uniqueTRLCObjectName(base string, used map[string]bool) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "requirement"
	}
	if !used[base] {
		used[base] = true
		return base
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s_%d", base, i)
		if !used[cand] {
			used[cand] = true
			return cand
		}
	}
}
