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

console-plugin/                  # GPU Config console plugin (React + Go)
  cmd/backend/main.go            # Go entry point (TLS, routing)
  pkg/api/                       # HTTP handlers (profiles, deploy, auth)
  pkg/kube/                      # K8s client (label nodes, patch CMs, restart workloads)
  src/components/                # React components (ProfileCard, MigConfigPanel, StatusPanel, etc.)
  src/utils/                     # API client, hooks, types
  chart/                         # Helm chart (ConsolePlugin CR, Deployment, RBAC)
  Containerfile                  # 3-stage build (nodejs-22 / go-toolset:1.25 / ubi-minimal)

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

Per worker node (configurable via gpu-config-plugin or values files):
- 8x NVIDIA full GPUs (`nvidia.com/gpu`)
- MIG slices vary by GPU profile (see below)

### Supported GPU profiles

| Profile | GPU Name | Memory | MIG | Architecture |
|---------|----------|--------|-----|--------------|
| a100 | NVIDIA A100-SXM4-40GB | 40 GiB HBM2e | Yes | Ampere |
| h100 | NVIDIA H100 80GB HBM3 | 80 GiB HBM3 | Yes | Hopper |
| h200 | NVIDIA H200 | 141 GiB HBM3e | Yes | Hopper |
| b200 | NVIDIA B200 | 192 GiB HBM3e | Yes | Blackwell |
| gb200 | NVIDIA GB200 NVL | 192 GiB HBM3e | Yes | Blackwell |
| l40s | NVIDIA L40S | 48 GiB GDDR6 | No | Ada Lovelace |
| t4 | NVIDIA T4 | 16 GiB GDDR6 | No | Turing |

### MIG slice families (from fake-gpu-operator)

- **\*40GB\*** (A100): 1g.5gb, 1g.5gb+me, 1g.10gb, 2g.10gb, 3g.20gb, 4g.20gb, 7g.40gb
- **\*80GB\*** (H100): 1g.10gb, 1g.10gb+me, 1g.20gb, 2g.20gb, 3g.40gb, 4g.40gb, 7g.80gb
- **\*H200\***: 1g.18gb, 2g.35gb, 3g.71gb

## GPU config plugin

The `console-plugin/` directory contains an OpenShift console plugin for selecting and deploying GPU profiles. Tech stack: React 17 + PatternFly v6 + Go 1.25 + gorilla/mux.

### How it works

1. User selects a builtin GPU profile (or enters a custom product name)
2. For MIG-capable profiles, user configures slice counts
3. Backend deploys the configuration in 4 steps:
   - Persists selection to ConfigMap `gpu-config-selected` in gpu-operator namespace
   - Updates the `topology` ConfigMap with new GPU product/count/memory/MIG config
   - Labels worker nodes (`run.ai/simulated-gpu-node-pool`, MIG labels/annotations)
   - Deletes per-node topology ConfigMaps and restarts operator workloads
4. The fake-gpu-operator picks up the new topology and registers GPU resources on nodes

### Build and deploy

```bash
cd console-plugin
podman build -t quay.io/eformat/gpu-config-plugin:latest -f Containerfile .
podman push quay.io/eformat/gpu-config-plugin:latest
helm install gpu-config-plugin chart/ -n gpu-config-plugin --create-namespace
```

### Key constraint: ACM conflict

If the fake-gpu-operator topology ConfigMap is managed by an ACM ConfigurationPolicy, ACM will revert changes made by the config plugin. Either disable the ACM policy for the topology ConfigMap, or deploy the config plugin standalone without ACM managing the operator resources.
