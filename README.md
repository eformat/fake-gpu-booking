# fake-gpu-booking

GPU resource booking and scheduling for OpenShift using fake GPUs (via [run-ai/fake-gpu-operator](https://github.com/run-ai/fake-gpu-operator)), Kueue, and hardware profiles.

## Deployment

### Single cluster (ArgoCD)

Apply the app-of-apps directly:

```bash
oc apply -f cluster/fake-gpu-app-of-apps.yaml
```

### Multi-cluster (ACM + ArgoCD)

Deploy the ACM ApplicationSet to a hub cluster with PolicyGenerator configured:

```bash
oc apply -f acm/fake-gpu.yaml
```

ACM distributes resources to spoke clusters via Policy + Placement. See `acm/` for details.

### Prerequisites

- OpenShift 4.x cluster
- ArgoCD / OpenShift GitOps
- For ACM path: ACM with PolicyGenerator plugin (`POLICY_GEN_ENABLE_HELM=true`)

## Components

| Component | Source | Description |
|-----------|--------|-------------|
| fake-gpu-operator | OCI chart (`ghcr.io/run-ai`) | Simulates GPU resources on nodes |
| gpu-rbac | Local Helm chart | RBAC, Kueue queues, hardware profiles |
| nfd | Local Helm chart | Node Feature Discovery operator |
| gpu-booking-plugin | Helm repo (`eformat`) | OpenShift console plugin for GPU booking |
| gpu-config-plugin | Local (`console-plugin/`) | OpenShift console plugin for GPU profile selection |
| rhoai | Kustomize | Red Hat OpenShift AI / ODH configuration |

## GPU Config Plugin

The `console-plugin/` directory contains an OpenShift console plugin that provides a UI for selecting GPU profiles and deploying them to the cluster.

Features:
- Select from 9 builtin GPU profiles (A100, H100, H200, B200, GB200, GB300, Vera Rubin, L40S, T4)
- Configure MIG slice counts per node for MIG-capable GPUs
- Custom/legacy mode for arbitrary GPU product names
- Deploys configuration by updating the topology ConfigMap, labeling nodes, and restarting operator workloads
- Status panel showing node readiness, DaemonSet health, and current active profile

```bash
cd console-plugin
podman build -t quay.io/eformat/gpu-config-plugin:latest -f Containerfile .
podman push quay.io/eformat/gpu-config-plugin:latest
helm install gpu-config-plugin chart/ -n gpu-config-plugin --create-namespace
```

## Workshop

See `content/` for the Antora-based workshop materials, or run `make` to build the docs site.

## Supported GPUs

Builtin GPU profiles with the fake-gpu-operator.

| Profile | GPU Name | Memory | MIG | Architecture |
|---------|----------|--------|-----|--------------|
| a100 | NVIDIA A100-SXM4-40GB | 40 GiB HBM2e | Yes | Ampere |
| h100 | NVIDIA H100 80GB HBM3 | 80 GiB HBM3 | Yes | Hopper |
| b200 | NVIDIA B200 | 192 GiB HBM3e | Yes | Blackwell |
| gb200 | NVIDIA GB200 NVL | 192 GiB HBM3e | Yes | Blackwell |
| gb300 | NVIDIA GB300 NVL | 288 GiB HBM3e | Yes | Blackwell Ultra |
| vera-rubin | NVIDIA Vera Rubin NVL | 288 GiB HBM4 | Yes | Rubin |
| l40s | NVIDIA L40S | 48 GiB GDDR6 | No | Ada Lovelace |
| t4 | NVIDIA T4 | 16 GiB GDDR6 | No | Turing |

MIG support:

| Match | Slices |
|-------|--------|
| **40GB** (A100) | 1g.5gb, 1g.5gb+me, 1g.10gb, 2g.10gb, 3g.20gb, 4g.20gb, 7g.40gb |
| **80GB** (H100) | 1g.10gb, 1g.10gb+me, 1g.20gb, 2g.20gb, 3g.40gb, 4g.40gb, 7g.80gb |
| **H200** | 1g.18gb, 2g.35gb, 3g.71gb |
| **GB200** | 1g.23gb, 1g.47gb, 2g.47gb, 3g.93gb, 4g.93gb, 7g.189gb |
| **GB300** | 1g.35gb, 1g.70gb, 2g.70gb, 3g.139gb, 4g.139gb, 7g.278gb |
| **Vera Rubin** | 1g.35gb, 1g.70gb, 2g.70gb, 3g.139gb, 4g.139gb, 7g.278gb |

These can be expanded by changes to the fake-gpu-operator codebase.
