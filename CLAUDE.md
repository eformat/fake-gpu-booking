# CLAUDE.md

## Project overview

This repo deploys a fake GPU booking system on OpenShift. It simulates NVIDIA GPU resources (H200 + MIG slices) using the run-ai fake-gpu-operator, then layers on Kueue scheduling, hardware profiles, RBAC, and an OpenShift console plugin for reservation management.

Two deployment paths exist:
- **Non-ACM**: ArgoCD app-of-apps at `cluster/fake-gpu-app-of-apps.yaml` deploys child Applications to a single cluster.
- **ACM**: PolicyGenerator-based pipeline at `acm/` wraps rendered Helm/kustomize output in ACM Policies, distributed to spoke clusters via Placement.

## Repository structure

```
cluster/
  fake-gpu-app-of-apps.yaml     # Root ArgoCD Application (non-ACM entry point)
  fake-gpu/                      # Child ArgoCD Application manifests
    fake-gpu-operator.yaml       # OCI Helm chart from ghcr.io/run-ai
    gpu-booking-app-plugin.yaml  # Helm chart from eformat repo
    gpu-rbac.yaml                # Local chart (symlink to acm/base/charts/gpu-rbac)
    nfd.yaml                     # Local chart (symlink to acm/base/charts/nfd)
    rhoai.yaml                   # Kustomize overlay

acm/
  fake-gpu.yaml                  # ArgoCD ApplicationSet (ACM entry point)
  base/
    kustomization.yaml           # Renders all Helm charts + rhoai into plain manifests
    gpu-operator/                # Sub-kustomization for fake-gpu-operator
      kustomization.yaml         # Includes namespace patches (chart omits namespace metadata)
      label-nodes-job.yaml       # Job to label worker nodes + apply MIG annotation
      namespace.yaml             # gpu-operator Namespace
      values.yaml                # Chart values with environment.openshift=true
    charts/
      gpu-rbac/                  # Source of truth for gpu-rbac chart (symlinked from repo root)
      nfd/                       # Source of truth for nfd chart (symlinked from repo root)
    rhoai.yaml                   # Pre-rendered output of rhoai/overlay/base
    values-gpu-booking-plugin.yaml
    namespace-gpu-booking-app-plugin.yaml
  overlay/
    kustomization.yaml           # generators: [policy-generator-config.yaml]
    policy-generator-config.yaml # Wraps all manifests in ACM Policy + PlacementBinding
    default/
      kustomization.yaml         # Variant directory referencing ../../base

gpu-rbac -> acm/base/charts/gpu-rbac   # Symlink (single source of truth)
nfd -> acm/base/charts/nfd             # Symlink (single source of truth)
fake-gpu-operator/values.yaml          # Values for non-ACM ArgoCD path
gpu-booking-app-plugin/values.yaml     # Values for non-ACM ArgoCD path

rhoai/                           # RHOAI/ODH kustomize overlays
  base/                          # Base DSC/DSCI/dashboard CRs + OLM chart
  overlay/
    base/                        # Active overlay (used by ArgoCD + pre-rendered for ACM)
    nightly-34/                  # Nightly build overlay

olm/                             # OLM operator wrapper charts
workloads/                       # GPU test workload chart
content/                         # Antora workshop docs
```

## Key architectural decisions

### Symlink pattern for local charts
`gpu-rbac/` and `nfd/` at the repo root are symlinks into `acm/base/charts/`. This is because PolicyGenerator's internal kustomize enforces `LoadRestrictionsRootOnly` — all files must be physically within the `acm/base/` tree. The symlinks point inward (repo root -> acm), so ArgoCD follows them transparently for the non-ACM path.

### Pre-rendered rhoai
`acm/base/rhoai.yaml` is a pre-rendered snapshot of `kustomize build rhoai/overlay/base`. This avoids nested kustomize references that escape the `acm/base/` security boundary. Must be regenerated when rhoai overlays change:
```bash
kustomize build --enable-helm rhoai/overlay/base > acm/base/rhoai.yaml
```

### Namespace patches for fake-gpu-operator
The fake-gpu-operator Helm chart does not set namespace in resource metadata (it relies on Helm's `--namespace` flag at install time). Since ACM ConfigurationPolicy requires explicit namespace on every namespaced resource, `acm/base/gpu-operator/kustomization.yaml` applies JSON patches to add `namespace: gpu-operator` to all Deployments, DaemonSets, Services, etc.

### Node labeling Job
The fake-gpu-operator requires nodes to be pre-labeled before DaemonSets schedule. In the non-ACM path this is done manually. The ACM path includes `acm/base/gpu-operator/label-nodes-job.yaml` which runs a Job to label all worker nodes with `run.ai/simulated-gpu-node-pool=default`, `node-role.kubernetes.io/runai-dynamic-mig=true`, and the MIG device annotation.

## Build and test commands

```bash
# Test ACM base rendering (Helm charts -> plain manifests, no PolicyGenerator)
kustomize build --enable-helm acm/base/

# Test full ACM pipeline (requires PolicyGenerator plugin binary)
kustomize build --enable-alpha-plugins --enable-helm acm/overlay/

# Test non-ACM Helm charts via symlinks
helm template test gpu-rbac
helm template test nfd

# Regenerate pre-rendered rhoai
kustomize build --enable-helm rhoai/overlay/base > acm/base/rhoai.yaml

# Build workshop docs
make
```

Note: The fake-gpu-operator OCI chart (`ghcr.io/run-ai`) requires authentication. Run `helm registry login ghcr.io` before testing locally.

## Values files that must stay in sync

| ACM values file | Original | Notes |
|-----------------|----------|-------|
| `acm/base/gpu-operator/values.yaml` | `fake-gpu-operator/values.yaml` | Adds `environment.openshift: true` |
| `acm/base/values-gpu-booking-plugin.yaml` | `gpu-booking-app-plugin/values.yaml` | Direct copy |
| `acm/base/rhoai.yaml` | `rhoai/overlay/base` | Pre-rendered, regenerate on change |

`gpu-rbac` and `nfd` values are NOT duplicated — the kustomization references charts directly via `chartHome`.

## ACM prerequisites (hub cluster)

- ACM installed with Placement API
- PolicyGenerator plugin in ArgoCD repo server (init container from `registry.redhat.io/rhacm2/multicluster-operators-subscription-rhel9`)
- `POLICY_GEN_ENABLE_HELM=true` env var on ArgoCD repo server
- `ManagedClusterSetBinding` for `openshift-gitops` namespace
- Placement `placement-hub-openshift` (targets hub for ApplicationSet)
- Placement `placement-spokes-fake-gpu` (targets spoke clusters for Policy distribution)
- ArgoCD CMP plugin `argocd-novault-plugin-kustomize` configured

## GPU resources simulated

Per worker node:
- 8x NVIDIA H200 full GPUs (`nvidia.com/gpu`)
- 8x MIG 3g.71gb slices (`nvidia.com/mig-3g.71gb`)
- 8x MIG 2g.35gb slices (`nvidia.com/mig-2g.35gb`)
- 16x MIG 1g.18gb slices (`nvidia.com/mig-1g.18gb`)
