# Sample Layered Architecture + EARS Project

This sample demonstrates the new layered architecture direction where authored design is stable and
runtime/code realization is inferred.

Authored architecture centers on:

- Functional Groups (`FG-*`)
- Functional Units (`FU-*`)
- Actors
- Attack Vectors
- Referenced Elements

Inferred layers are represented by example ownership and runtime hints:

- Runtime inferred from `infra/terraform`, `infra/flux`, `infra/helm`
- Code ownership inferred from package/module-level owner annotations in `src/`
- Fine-grained requirement traces from `TRACE-REQS` markers in code

The implementation files are intentionally dummy but believable:

- they include trace markers for requirement linkage
- they include coarse ownership markers for FU mapping
- they log/print expected actions in each subsystem
- they are meant for documentation and traceability, not production execution

Story highlights:

- checkout and authorization are authored in `FG-PAYMENTS`
- risk scoring and support review are authored in `FG-FRAUD`
- cluster provisioning and GitOps operations are authored in `FG-PLATFORM`
- Flux/Terraform artifacts annotate ownership upward with `engmodel.dev/owner-unit`

## Files

- `catalog.yml`
- `requirements.yml`
- `architecture.yml`
- `design.yml`
- `infra/terraform/main.tf`
- `infra/flux/...`
- `infra/helm/...`
- `src/...`

Note:
- This example now follows the proposed new conceptual model.
- Existing generator implementation may not parse this format yet.
- The sample is intentionally updated first to validate target structure and terminology.

## Upstream References

These are known working upstream projects/docs for the same stack:

- Terraform AWS EKS module: https://github.com/terraform-aws-modules/terraform-aws-eks
- Flux bootstrap and GitOps toolkit docs: https://fluxcd.io/flux/
- Flux HelmRelease API docs: https://fluxcd.io/flux/components/helm/helmreleases/
- Helm chart docs: https://helm.sh/docs/topics/charts/
