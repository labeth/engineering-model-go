// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
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

func TestBuildLinkTargets_ProvidesTitleAndNaturalCaseVariants(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor: "REF_IDX-CATALOG_REF_CLOUD_LOGGING_SERVICE",
				ID:     "REF-CLOUD-LOGGING-SERVICE",
				Name:   "cloud logging service",
				Kind:   "Referenced Element",
			},
			{
				Anchor: "REF_IDX-CATALOG_FU_DEVICE_IDENTITY_SECRETS",
				ID:     "FU-DEVICE-IDENTITY-SECRETS",
				Name:   "device identity and secrets",
				Kind:   "Functional Unit",
			},
		},
	})
	if _, ok := targets["Cloud Logging Service"]; !ok {
		t.Fatalf("missing title-case variant for cloud logging service")
	}
	if _, ok := targets["Device Identity and Secrets"]; !ok {
		t.Fatalf("missing natural title-case variant for phrase with minor words")
	}
}

func TestLinkifyText_PreservesExistingLinksAndAddsNewOnes(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor: "REF_IDX-CATALOG_FU_FLEET_OBSERVABILITY_REPORTING",
				ID:     "FU-FLEET-OBSERVABILITY-REPORTING",
				Name:   "fleet observability reporting",
				Kind:   "Functional Unit",
			},
			{
				Anchor: "REF_IDX-CATALOG_REF_CLOUD_LOGGING_SERVICE",
				ID:     "REF-CLOUD-LOGGING-SERVICE",
				Name:   "cloud logging service",
				Kind:   "Referenced Element",
			},
		},
	})
	text := "functional responsibility for <<REF_IDX-CATALOG_FU_FLEET_OBSERVABILITY_REPORTING,fleet observability reporting>>; includes decision flow to Cloud Logging Service."
	got := linkifyText(text, targets)
	if strings.Count(got, "<<REF_IDX-CATALOG_FU_FLEET_OBSERVABILITY_REPORTING,fleet observability reporting>>") != 1 {
		t.Fatalf("expected existing link to be preserved once, got %q", got)
	}
	if !strings.Contains(got, "<<REF_IDX-CATALOG_REF_CLOUD_LOGGING_SERVICE,Cloud Logging Service>>") {
		t.Fatalf("expected cloud logging service to be linkified, got %q", got)
	}
}

func TestLinkifyText_LinksAcronymCasedCatalogPhrase(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor: "REF_IDX-CATALOG_REF_CLOUD_RUNTIME_SDK",
				ID:     "REF-CLOUD-RUNTIME-SDK",
				Name:   "cloud runtime sdk",
				Kind:   "Referenced Element",
			},
		},
	})
	text := "Calls Cloud Runtime SDK for publish and identity token checks."
	got := linkifyText(text, targets)
	if !strings.Contains(got, "<<REF_IDX-CATALOG_REF_CLOUD_RUNTIME_SDK,Cloud Runtime SDK>>") {
		t.Fatalf("expected acronym-cased phrase to be linkified, got %q", got)
	}
}

func TestLinkifyText_LinksMQTTAcronymPhrase(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor: "REF_IDX-CATALOG_REF_MQTT_DEVICE_SDK",
				ID:     "REF-MQTT-DEVICE-SDK",
				Name:   "mqtt device sdk",
				Kind:   "Referenced Element",
			},
		},
	})
	text := "Publishes telemetry through MQTT Device SDK for edge delivery."
	got := linkifyText(text, targets)
	if !strings.Contains(got, "<<REF_IDX-CATALOG_REF_MQTT_DEVICE_SDK,MQTT Device SDK>>") {
		t.Fatalf("expected MQTT acronym phrase to be linkified, got %q", got)
	}
}

func TestLinkifyText_LinksMixedCaseIoTPhrase(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor: "REF_IDX-CATALOG_REF_IOT_INGEST_ENDPOINT",
				ID:     "REF-IOT-INGEST-ENDPOINT",
				Name:   "iot ingest endpoint",
				Kind:   "Referenced Element",
			},
		},
	})
	text := "Publishes telemetry to IoT Ingest Endpoint for cloud intake."
	got := linkifyText(text, targets)
	if !strings.Contains(got, "<<REF_IDX-CATALOG_REF_IOT_INGEST_ENDPOINT,IoT Ingest Endpoint>>") {
		t.Fatalf("expected mixed-case IoT phrase to be linkified, got %q", got)
	}
}

func TestLinkifyText_LinksCatalogAliasPhrase(t *testing.T) {
	targets := buildLinkTargets(asciidocReferenceIndex{
		Catalog: []asciidocReferenceEntry{
			{
				Anchor:  "REF_IDX-CATALOG_EVT_OTA_CAMPAIGN_SCHEDULED",
				ID:      "EVT-OTA-CAMPAIGN-SCHEDULED",
				Name:    "ota campaign scheduled event is received",
				Aliases: []string{"ota commands"},
				Kind:    "Event",
			},
		},
	})
	text := "Machine Edge receives OTA commands and reports apply status."
	got := linkifyText(text, targets)
	if !strings.Contains(got, "<<REF_IDX-CATALOG_EVT_OTA_CAMPAIGN_SCHEDULED,OTA commands>>") {
		t.Fatalf("expected alias phrase to be linkified, got %q", got)
	}
}
