# Workshop Diagrams

Mermaid source for all diagrams used in the workshop.

## 1. Architecture Overview (index.adoc)

```mermaid
flowchart TB
    subgraph Console["OpenShift Console"]
        BookingApp["GPU Booking App\n(Console Plugin)"]
    end

    subgraph RHOAI["Red Hat OpenShift AI"]
        Workbench["Workbenches"]
        HWProfiles["Hardware Profiles"]
        ModelServing["Model Serving"]
    end

    subgraph Kueue["Kueue Scheduler"]
        Cohort["Cohort: unreserved"]
        CQ_Unreserved["ClusterQueue:\nunreserved"]
        CQ_User["ClusterQueue:\nuser-alice"]
        LQ["LocalQueues"]
        RF["ResourceFlavors"]
    end

    subgraph Nodes["Cluster Nodes"]
        FakeGPU["Fake GPU Operator"]
        DevPlugin["Device Plugin"]
        NFD["Node Feature\nDiscovery"]
        GPU_H200["H200 x8"]
        MIG["MIG Partitions\n3g / 2g / 1g"]
    end

    BookingApp -->|"creates reservations"| CQ_User
    BookingApp -->|"creates"| HWProfiles
    BookingApp -->|"syncs usage"| LQ
    Workbench -->|"selects"| HWProfiles
    HWProfiles -->|"maps to"| LQ
    LQ -->|"submits to"| CQ_Unreserved
    LQ -->|"submits to"| CQ_User
    CQ_Unreserved --- Cohort
    CQ_User --- Cohort
    Cohort -->|"schedules on"| RF
    RF -->|"node affinity"| FakeGPU
    FakeGPU --> DevPlugin
    NFD -->|"labels nodes"| DevPlugin
    DevPlugin --> GPU_H200
    DevPlugin --> MIG

    style Console fill:#4394E5,color:#fff
    style RHOAI fill:#EE0000,color:#fff
    style Kueue fill:#326CE5,color:#fff
    style Nodes fill:#2D2D2D,color:#fff
```

## 2. Fake GPU Operator Components (module-01)

```mermaid
flowchart LR
    subgraph Operator["Fake GPU Operator"]
        DP["Device Plugin"]
        SU["Status Updater"]
        NSI["nvidia-smi\nInjection"]
    end

    subgraph Node["Worker Node"]
        Kubelet["Kubelet"]
        Prom["Prometheus\nMetrics"]
    end

    subgraph Pod["GPU Pod"]
        Container["Container"]
        NSMI["nvidia-smi"]
    end

    Helm["Helm Values\n(topology)"] -->|"gpuProduct: H200\ngpuCount: 8\nmigDevices: 32"| Operator
    Label["Node Label\nrun.ai/simulated-\ngpu-node-pool"] -->|"selects"| Node
    DP -->|"registers\nnvidia.com/gpu\nnvidia.com/mig-*"| Kubelet
    SU -->|"reports\nutilization"| Prom
    NSI -->|"injects\nfake binary"| NSMI
    Kubelet -->|"allocates\nGPU to pod"| Container
    Container --- NSMI

    style Operator fill:#76B900,color:#fff
    style Node fill:#2D2D2D,color:#fff
    style Pod fill:#326CE5,color:#fff
```

## 3. Booking Sync Lifecycle (module-02)

```mermaid
sequenceDiagram
    participant User as User
    participant App as Booking App
    participant DB as SQLite DB
    participant K8s as Kubernetes API
    participant Kueue as Kueue

    Note over App: Kueue Sync (every 30s)
    App->>K8s: Poll LocalQueues
    K8s-->>App: flavorUsage data
    App->>DB: Create/update consumed bookings

    Note over App: Reservation Sync (every 10min)
    User->>App: Create reservation
    App->>DB: Store reserved booking
    App->>K8s: Create ClusterQueue (user-alice)
    App->>K8s: Create LocalQueue
    App->>K8s: Create HardwareProfile
    App->>K8s: Update unreserved Cohort quota
    K8s->>Kueue: Quota change triggers preemption

    Note over App: Expiration Cleaner (every 10min)
    App->>K8s: Phase 1: Set stopPolicy HoldAndDrain
    App->>K8s: Delete LocalQueue & HardwareProfile
    App->>K8s: Phase 2: Delete ClusterQueue
```

## 4. Kueue Resource Hierarchy (module-04)

```mermaid
flowchart TB
    subgraph Cohort["Cohort: unreserved"]
        CQ1["ClusterQueue: unreserved\n─────────────\nnominalQuota: (total - reserved)\nreclaimWithinCohort: Never\nborrowWithinCohort: Always"]
        CQ2["ClusterQueue: unreserved-priority\n─────────────\nnominalQuota: 0\nPreempts: low-priority borrowing"]
        CQ3["ClusterQueue: user-alice\n─────────────\nnominalQuota: 2 GPU\nreclaimWithinCohort: Any\nborrowWithinCohort: Never"]
        CQ4["ClusterQueue: user-bob\n─────────────\nnominalQuota: 1 GPU\nreclaimWithinCohort: Any\nborrowWithinCohort: Never"]
    end

    RF["ResourceFlavor: h200\n(node affinity + tolerations)"]

    subgraph NS1["Namespace: alice"]
        LQ1["LocalQueue: default"]
        LQ2["LocalQueue: unreserved"]
    end

    subgraph NS2["Namespace: bob"]
        LQ3["LocalQueue: default"]
        LQ4["LocalQueue: unreserved"]
    end

    LQ1 -->|"points to"| CQ3
    LQ2 -->|"points to"| CQ1
    LQ3 -->|"points to"| CQ4
    LQ4 -->|"points to"| CQ1

    CQ1 -->|"uses"| RF
    CQ2 -->|"uses"| RF
    CQ3 -->|"uses"| RF
    CQ4 -->|"uses"| RF

    style Cohort fill:#326CE5,color:#fff
    style NS1 fill:#4394E5,color:#fff
    style NS2 fill:#4394E5,color:#fff
    style RF fill:#76B900,color:#fff
```

## 5. Kueue Hierarchy with Reservations (module-05)

```mermaid
flowchart TB
    subgraph Cohort["Cohort: unreserved"]
        CQ1["ClusterQueue: unreserved\n─────────────\nnominalQuota: 5 GPU\n(reduced from 8)\nreclaimWithinCohort: Never"]
        CQ2["ClusterQueue: unreserved-priority\n─────────────\nnominalQuota: 0\nPreempts: low-priority borrowing"]
        CQ3["ClusterQueue: user-admin\n─────────────\nnominalQuota: 2 GPU\nreclaimWithinCohort: Any\nborrowWithinCohort: Never"]
        CQ4["ClusterQueue: user-admin1\n─────────────\nnominalQuota: 1 GPU\nreclaimWithinCohort: Any\nborrowWithinCohort: Never"]
    end

    RF["ResourceFlavor: h200\n(node affinity + tolerations)"]

    subgraph NS1["Namespace: user-admin"]
        LQ1a["LocalQueue: default"]
        LQ1b["LocalQueue: unreserved"]
        LQ1c["LocalQueue: user-admin\n(reserved)"]
    end

    subgraph NS2["Namespace: user-admin1"]
        LQ2a["LocalQueue: default"]
        LQ2b["LocalQueue: unreserved"]
        LQ2c["LocalQueue: user-admin1\n(reserved)"]
    end

    LQ1a -->|"points to"| CQ1
    LQ1b -->|"points to"| CQ1
    LQ1c -->|"points to"| CQ3
    LQ2a -->|"points to"| CQ1
    LQ2b -->|"points to"| CQ1
    LQ2c -->|"points to"| CQ4

    CQ1 -->|"uses"| RF
    CQ2 -->|"uses"| RF
    CQ3 -->|"uses"| RF
    CQ4 -->|"uses"| RF

    style Cohort fill:#326CE5,color:#fff
    style NS1 fill:#4394E5,color:#fff
    style NS2 fill:#4394E5,color:#fff
    style RF fill:#76B900,color:#fff
    style LQ1c fill:#EE0000,color:#fff
    style LQ2c fill:#EE0000,color:#fff
    style CQ3 fill:#1a4d8f,color:#fff
    style CQ4 fill:#1a4d8f,color:#fff
```

## 6. Preemption Flow (module-05)

```mermaid
sequenceDiagram
    participant U1 as Unreserved User
    participant U2 as Reserved User
    participant Kueue as Kueue
    participant CQ_U as CQ: unreserved
    participant CQ_R as CQ: user-reserved
    participant Node as GPU Node

    U1->>CQ_U: Submit workload (8 GPUs)
    CQ_U->>Kueue: Admit (borrow from Cohort)
    Kueue->>Node: Schedule pod (8/8 GPUs used)
    Note over Node: All GPUs consumed

    U2->>CQ_R: Submit workload (1 GPU)
    CQ_R->>Kueue: Admit (reclaimWithinCohort: Any)
    Kueue->>CQ_U: Preempt 1 GPU from unreserved
    CQ_U-->>Node: Evict unreserved pod
    Note over Node: Pod recreated with 7 GPUs
    Kueue->>Node: Schedule reserved pod (1 GPU)
    Note over Node: 7 unreserved + 1 reserved = 8 GPUs
```
