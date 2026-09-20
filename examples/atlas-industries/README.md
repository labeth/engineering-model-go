# Atlas Industries model workspace

This directory simulates seven independently publishable repositories. Four shared
modules (`shared-edge-platform`, `shared-cloud-platform`, `shared-security-services`,
and `shared-compliance-baseline`) are reused by both product modules
(`aegis-sentinel` and `orion-relay`). The `company-portfolio` module composes both
products. Product requirements allocate to shared-module contract capabilities;
portfolio requirements allocate to product contract capabilities.

The shared compliance baseline also models non-technical company controls once
for both products. Its employee security lifecycle covers personnel screening,
employment and acceptable-use acknowledgment, initial and annual awareness
training, role-change access review, and termination offboarding. Representative
evidence is mapped to NIST SP 800-53 AC-2, AT-2, PS-3, and PS-4 and to ISO/IEC
27001:2022 A.5.18, A.6.1, A.6.2, A.6.3, and A.6.5. A POA&M item deliberately
tracks incomplete contractor training reporting, demonstrating remediation as
well as implemented controls; the example is not a certification claim.

For local development, `engmod.work.yml` replaces every authored OCI identity with
its directory under `repos/`. Generate every supported projection for all seven
repositories into the maintained `generated/` directory:

```sh
./examples/atlas-industries/generate.sh
go test . -run TestAtlasIndustriesCompanyExample
```

The generation script emits AsciiDoc, decisions, traceability, Structurizr,
SysML v2, TRLC, LOBSTER, Threat Dragon, Open OTM, OSCAL, and Gemara outputs.
Pass an explicit output path only for temporary validation copies.

CI or a consumer repository omits `engmod.work.yml` and resolves the same exact
`v0.1.0` coordinates from the OCI registry selected by `CUE_REGISTRY`. Publish
all modules after authenticating with the registry's normal OCI credential
helper:

```sh
export CUE_REGISTRY=registry.example.com
./examples/atlas-industries/publish.sh
```

Engmod materializes fetched modules beneath the workspace `.engmod/modules`
cache and reuses exact-version entries offline. Delete only that cache when
deliberately forcing a fresh registry fetch. Every repository contains all five
versioned canonical YAML documents and a publishable `cue.mod/module.cue`.
