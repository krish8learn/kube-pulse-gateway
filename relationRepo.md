# Architectural Overview

```text
Client
  │
  ▼
Gateway
  ├──gRPC──▶ Collector  →  "give me raw pod/node metrics"
  │              │
  │              ▼
  │          K8s API
  │
  └──gRPC──▶ Analyzer  →  "are there any anomalies right now?"
                 │
                 ▼
              returns: [CPU spike on pod-X, crash loop on pod-Y]
```

# Service Responsibilities and Data Acquisition

The two services answer fundamentally different questions:

* **Collector:** "What is the current state?" (raw metrics, e.g., CPU %, memory, pod status).
* **Analyzer:** "Is anything wrong?" (anomaly verdicts, e.g., spike detected, crash loop detected).

## Data Acquisition Strategy

Both the **Analyzer** and the **Collector** independently call the K8s API directly via `kube-rs`. This approach is preferred for being simpler and ensuring the services remain fully independent.

```text
Collector ──▶ K8s API (raw metrics)
Analyzer  ──▶ K8s API (anomaly detection)
```

## Health Check

```json
GET /health  →  Gateway calls both services in parallel
                returns combined JSON:
                {
                  "metrics": { ... },   ← from Collector
                  "anomalies": [ ... ]  ← from Analyzer
                }
```
