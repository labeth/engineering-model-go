package engmodel

import (
	"strings"
	"testing"
)

func TestBuildLinkTargets_UsesRegistryAnchorAndPluralVariants(t *testing.T) {
	ref := asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor:       "REF_IDX-ENGMODEL_EM_FUNCTIONAL_GROUP",
				TargetAnchor: "REF_ENGMODEL_EM_FUNCTIONAL_GROUP",
				ID:           "EM-FUNCTIONAL-GROUP",
				Name:         "functional group",
				Kind:         "Engineering Model Term",
			},
		},
	}

	targets := buildLinkTargets(ref)
	singular, ok := targets["functional group"]
	if !ok {
		t.Fatalf("missing singular token link target")
	}
	if singular.Anchor != "REF_IDX-ENGMODEL_EM_FUNCTIONAL_GROUP" {
		t.Fatalf("expected registry/index anchor, got %q", singular.Anchor)
	}

	plural, ok := targets["functional groups"]
	if !ok {
		t.Fatalf("missing plural token link target")
	}
	if plural.Anchor != "REF_IDX-ENGMODEL_EM_FUNCTIONAL_GROUP" {
		t.Fatalf("expected plural to resolve to same anchor, got %q", plural.Anchor)
	}

	if _, ok := targets["Functional Groups"]; !ok {
		t.Fatalf("missing title-case plural token variant")
	}
}

func TestLinkifyText_ConnectsPluralPhrases(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor: "REF_IDX-ENGMODEL_EM_FUNCTIONAL_GROUP",
				ID:     "EM-FUNCTIONAL-GROUP",
				Name:   "functional group",
				Kind:   "Engineering Model Term",
			},
			{
				Anchor: "REF_IDX-ENGMODEL_EM_FUNCTIONAL_UNIT",
				ID:     "EM-FUNCTIONAL-UNIT",
				Name:   "functional unit",
				Kind:   "Engineering Model Term",
			},
		},
	})

	text := "Functional Groups and Functional Units are stable authored design anchors."
	got := linkifyText(text, targets)
	if !strings.Contains(got, "<<REF_IDX-ENGMODEL_EM_FUNCTIONAL_GROUP,Functional Groups>>") {
		t.Fatalf("expected plural functional groups to be linkified, got %q", got)
	}
	if !strings.Contains(got, "<<REF_IDX-ENGMODEL_EM_FUNCTIONAL_UNIT,Functional Units>>") {
		t.Fatalf("expected plural functional units to be linkified, got %q", got)
	}
}
