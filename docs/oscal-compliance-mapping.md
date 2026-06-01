# OSCAL Compliance Mapping

OSCAL export is driven by top-level `compliance` data. The old authored control allocation path has been removed to avoid duplicate implementations of the same mapping.

## Model Shape

Use `compliance.profiles[]` to select the profile and catalog sources:

```yaml
compliance:
  profiles:
    - id: PROF-NIST-800-53-LOW
      href: ./oscal/profile-nist-800-53-low.json
      catalogHref: ./oscal/catalog-nist-800-53-rev5-subset.json
```

Use `compliance.mappings[]` to map local implementation controls to selected OSCAL controls:

```yaml
  mappings:
    - id: MAP-PAYMENTS-SSO-MFA
      profileRef: PROF-NIST-800-53-LOW
      modelControlRef: CTRL-PAYMENTS-SSO-MFA
      controlIds: [ac-2, ia-2, ia-2.1]
      appliesTo: [FU-SUPPORT-REVIEW, FU-GITOPS-OPERATIONS]
      implementationType: technical
      implementationStatus: implemented
```

Rules:

- `modelControlRef` points to a local `CTRL-*` implementation control.
- `controlIds` must be selected by the referenced profile when a local profile/catalog can be loaded.
- Unmapped local controls are not exported as OSCAL implementations.
- Out-of-scope rationale belongs in authored assumptions/out-of-scope records, not in fake mappings.

## Payments Low Baseline Example

The payments sample uses a local NIST 800-53 Low profile subset:

- `examples/payments-engineering-sample/oscal/profile-nist-800-53-low.json`
- `examples/payments-engineering-sample/oscal/catalog-nist-800-53-rev5-subset.json`

The Low mapping currently exports:

- `CTRL-PAYMENTS-SSO-MFA` -> `ac-2`, `ia-2`, `ia-2.1`
- `CTRL-PAYMENTS-IMAGE-DIGEST` -> `cm-6`

`CTRL-PAYMENTS-CALLBACK-NONCE` remains a local control but is out of scope for the Low profile and is not exported as `si-10`.

## Generate

```bash
go run ./cmd/engoscal \
  --model examples/payments-engineering-sample/architecture.yml \
  --requirements examples/payments-engineering-sample/requirements.yml \
  --code-root examples/payments-engineering-sample/src \
  --profile examples/payments-engineering-sample/oscal/profile-nist-800-53-low.json \
  --catalog examples/payments-engineering-sample/oscal/catalog-nist-800-53-rev5-subset.json \
  --ssp-out examples/payments-engineering-sample/generated/ARCHITECTURE.ssp.json \
  --ar-out examples/payments-engineering-sample/generated/ARCHITECTURE.ar.json \
  --poam-out examples/payments-engineering-sample/generated/ARCHITECTURE.poam.json
```

## Verify

```bash
rg '"control-id": "(ac-2|cm-6|ia-2|ia-2\.1|si-10)"' \
  examples/payments-engineering-sample/generated/ARCHITECTURE.ssp.json \
  examples/payments-engineering-sample/generated/ARCHITECTURE.ar.json
```

Expected result: `ac-2`, `cm-6`, `ia-2`, and `ia-2.1` are present; `si-10` is absent for the Low profile.
