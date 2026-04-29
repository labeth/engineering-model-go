// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package diagramstyle

import "strings"

var mermaidClassDefs = []string{
	"classDef system_boundary fill:#f5f5f5,stroke:#424242,stroke-width:2px,color:#212121;",
	"classDef functional_group fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px,color:#1b5e20;",
	"classDef functional_unit fill:#e3f2fd,stroke:#0d47a1,stroke-width:1px,color:#0d47a1;",
	"classDef actor fill:#fff8e1,stroke:#ef6c00,stroke-width:1px,color:#bf360c;",
	"classDef attack_vector fill:#ffebee,stroke:#b71c1c,stroke-width:1px,color:#7f0000;",
	"classDef referenced_element fill:#f3e5f5,stroke:#6a1b9a,stroke-width:1px,color:#4a148c;",
	"classDef interface fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,stroke-dasharray: 6 3,color:#1b5e20;",
	"classDef data_object fill:#fff3e0,stroke:#ef6c00,stroke-width:2px,stroke-dasharray: 2 2,color:#e65100;",
	"classDef deployment_target fill:#ede7f6,stroke:#5e35b1,stroke-width:1px,color:#4527a0;",
	"classDef control fill:#fce4ec,stroke:#ad1457,stroke-width:1px,color:#880e4f;",
	"classDef trust_boundary fill:#e0f7fa,stroke:#006064,stroke-width:2px,stroke-dasharray: 10 4,color:#004d40;",
	"classDef state fill:#e1f5fe,stroke:#0277bd,stroke-width:1px,color:#01579b;",
	"classDef event fill:#fffde7,stroke:#f9a825,stroke-width:1px,color:#f57f17;",
	"classDef flow fill:#ede7f6,stroke:#4527a0,stroke-width:2px,color:#311b92;",
	"classDef flow_step fill:#f1f8e9,stroke:#33691e,stroke-width:2px,color:#1b5e20;",
	"classDef requirement fill:#fffde7,stroke:#f9a825,stroke-width:1px,color:#7f6000;",
	"classDef verification fill:#fce4ec,stroke:#ad1457,stroke-width:1px,color:#880e4f;",
	"classDef runtime_element fill:#b2ebf2,stroke:#00838f,stroke-width:1px,color:#006064;",
	"classDef deployment_element fill:#d7ccc8,stroke:#4e342e,stroke-width:1px,color:#2f1b14;",
	"classDef code_element fill:#eceff1,stroke:#37474f,stroke-width:1px,color:#263238;",
	"classDef unknown fill:#ffffff,stroke:#adb5bd,stroke-width:1px,color:#4f5b62;",
}

func MermaidClassDefs() []string {
	return append([]string(nil), mermaidClassDefs...)
}

func MermaidClassDefsWithIndent(indent string) []string {
	defs := MermaidClassDefs()
	if strings.TrimSpace(indent) == "" {
		return defs
	}
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, indent+d)
	}
	return out
}

func MermaidClassDefsBlock(indent string) string {
	return strings.Join(MermaidClassDefsWithIndent(indent), "\n")
}
