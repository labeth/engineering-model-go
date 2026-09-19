# Engineering Assurance Summary

This file is generated from the canonical Engineering Model. Empty generated documents are not proof of coverage; use the status table below.

- Model: `engineering-model-go`
- Generated artifacts: 35
- Covered assurance areas: 12
- Missing model areas: 5
- Optional outputs not applicable: 0
- Blocked optional outputs: 0

| Assurance area | Status | Evidence count | Required | Next action |
|---|---:|---:|---:|---|
| architecture-decisions | covered | 19 | 1 | Record material architectural and governance trade-offs as decisions. |
| architecture-structure | covered | 38 | 1 | Model functional boundaries, interfaces, and important flows. |
| architecture-view/architecture-intent | covered | 1 | 1 | Author the architecture-intent view with relevant roots, scope, audience, and design narrative. |
| architecture-view/communication | missing-model | 0 | 1 | Author the communication view with relevant roots, scope, audience, and design narrative. |
| architecture-view/deployment | missing-model | 0 | 1 | Author the deployment view with relevant roots, scope, audience, and design narrative. |
| architecture-view/interaction-flow | covered | 1 | 1 | Author the interaction-flow view with relevant roots, scope, audience, and design narrative. |
| architecture-view/security | covered | 1 | 1 | Author the security view with relevant roots, scope, audience, and design narrative. |
| architecture-view/state-lifecycle | covered | 1 | 1 | Author the state-lifecycle view with relevant roots, scope, audience, and design narrative. |
| architecture-view/traceability | covered | 1 | 1 | Author the traceability view with relevant roots, scope, audience, and design narrative. |
| compliance | missing-model | 0 | 1 | Select applicable compliance profiles and map implemented controls, or record a reviewed non-applicability decision. |
| data-classification | missing-model | 0 | 12 | Model every data object with sensitivity, classification, CIA impact, retention, and explicit regulatory tags. |
| governance-controls | covered | 7 | 1 | Model controls with responsible verification evidence. |
| implementation-traceability | covered | 45 | 45 | Add TRLC-LINKS implementation evidence for each non-delegated requirement. |
| requirements-engineering | covered | 45 | 1 | Author at least one durable requirement with implementation and verification traceability. |
| risk-and-poam | covered | 5 | 1 | Model identified risks and add POA&M items for accepted remediation work. |
| security-threat-model | covered | 8 | 1 | Model trust boundaries, attack vectors, and threat scenarios or record why they are not applicable. |
| verification-traceability | missing-model | 38 | 45 | Add observable verification evidence for each non-delegated requirement. |

## Generated artifacts

| Path | Domain | Format | Status |
|---|---|---|---|
| `ARCHITECTURE.adoc` | architecture | adoc | generated |
| `ASSURANCE-MANIFEST.json` | assurance | json | generated |
| `ASSURANCE-SUMMARY.md` | assurance | md | generated |
| `DECISIONS.adoc` | assurance | adoc | generated |
| `STRUCTURIZR.dsl` | architecture | dsl | generated |
| `TRACE-MATRIX.csv` | requirements | csv | generated |
| `TRACE-MATRIX.json` | requirements | json | generated |
| `VIEW-ARCHITECTURE-INTENT.mmd` | architecture | mmd | generated |
| `VIEW-INTERACTION-FLOW.mmd` | architecture | mmd | generated |
| `VIEW-SECURITY.mmd` | architecture | mmd | generated |
| `VIEW-STATE-LIFECYCLE.mmd` | architecture | mmd | generated |
| `VIEW-TRACEABILITY.mmd` | architecture | mmd | generated |
| `compliance/GEMARA-OSCAL-ASSESSMENT-RESULTS.json` | compliance | json | generated |
| `compliance/GEMARA-OSCAL-CATALOG.json` | compliance | json | generated |
| `compliance/OSCAL-ASSESSMENT-RESULTS.json` | compliance | json | generated |
| `compliance/OSCAL-POAM.json` | compliance | json | generated |
| `compliance/OSCAL-SSP.json` | compliance | json | generated |
| `governance/gemara/audit-log.yaml` | governance | yaml | generated |
| `governance/gemara/capability-catalog.yaml` | governance | yaml | generated |
| `governance/gemara/control-catalog.yaml` | governance | yaml | generated |
| `governance/gemara/control-threat-mapping.yaml` | governance | yaml | generated |
| `governance/gemara/enforcement-log.yaml` | governance | yaml | generated |
| `governance/gemara/evaluation-log.yaml` | governance | yaml | generated |
| `governance/gemara/guidance-catalog.yaml` | governance | yaml | generated |
| `governance/gemara/lexicon.yaml` | governance | yaml | generated |
| `governance/gemara/policy.yaml` | governance | yaml | generated |
| `governance/gemara/principle-catalog.yaml` | governance | yaml | generated |
| `governance/gemara/risk-catalog.yaml` | governance | yaml | generated |
| `governance/gemara/threat-catalog.yaml` | governance | yaml | generated |
| `governance/gemara/vector-catalog.yaml` | governance | yaml | generated |
| `requirements/requirements.rsl` | requirements | rsl | generated |
| `requirements/requirements.trlc` | requirements | trlc | generated |
| `requirements/tests.lobster.json` | requirements | json | generated |
| `security/THREAT-MODEL.open-otm.json` | security | json | generated |
| `security/THREAT-MODEL.threat-dragon.json` | security | json | generated |
