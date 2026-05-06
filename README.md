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
| rhoai | Kustomize | Red Hat OpenShift AI / ODH configuration |

## Workshop

See `content/` for the Antora-based workshop materials, or run `make` to build the docs site.
