// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import "strings"

// NAFV41Viewpoints is the official NAF v4.1 architecture-product registry
// from the framework's viewpoint chapter.
//
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-VALIDATION-ENGINE, FU-NAF-EXPORTER, REF-NAF-V4-1-SPECIFICATION
var NAFV41Viewpoints = map[string]string{
	"C1":    "Capability Taxonomy",
	"C2":    "Enterprise Vision",
	"C3":    "Capability Dependencies",
	"C4":    "Standard Processes",
	"C5":    "Effects",
	"C7":    "Performance Criteria",
	"C8":    "Planning Constraints",
	"Cr":    "Capability Roadmap",
	"S1":    "Service Taxonomy",
	"S2":    "Service Structure",
	"S3":    "Service Interfaces",
	"S4":    "Service Functions",
	"S5":    "Service States",
	"S6":    "Service Sequence",
	"S7":    "Service Interface Parameters",
	"S8":    "Service Constraints",
	"Sr":    "Service Roadmap",
	"C1-S1": "Service to Capability Mapping",
	"L1":    "Logical Taxonomy",
	"L2":    "Logical Structure",
	"L2-L3": "Logical Concept",
	"L3":    "Logical Interactions",
	"L4":    "Logical Activities",
	"L5":    "Logical States",
	"L6":    "Logical Sequence",
	"L7":    "Information Model",
	"L8":    "Logical Constraints",
	"Lr":    "Logical Roadmap",
	"P1":    "Resource Taxonomy",
	"P2":    "Resource Structure",
	"P3":    "Resource Interactions",
	"P4":    "Resource Functions",
	"L4-P4": "Activity to Function Mapping",
	"P5":    "Resource States",
	"P6":    "Resource Sequence",
	"P7":    "Data Model",
	"P8":    "Resource Constraints",
	"Pr":    "Resource Roadmap",
	"A1":    "Metadata Definitions",
	"A2":    "Architecture Products",
	"A3":    "Architecture Correspondence",
	"A4":    "Architecture Methodology",
	"A5":    "Architecture Status",
	"A6":    "Architecture Versions",
	"A7":    "Architecture Metadata",
	"A8":    "Architecture Standards",
	"Ar":    "Architecture Roadmap",
}

// NAFV41ViewpointTitle resolves an official viewpoint code.
// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
func NAFV41ViewpointTitle(code string) (string, bool) {
	title, ok := NAFV41Viewpoints[strings.TrimSpace(code)]
	return title, ok
}
