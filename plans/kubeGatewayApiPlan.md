### API Groups

#### 1. Health & System APIs
*Build first — always needed.*
* `GET /health` — Gateway liveness check.
* `GET /health/collector` — Ping the Rust collector via gRPC.
* `GET /health/analyzer` — Ping the Rust analyzer via gRPC.

#### 2. Cluster Overview APIs
*Your "dashboard" layer.*
* `GET /cluster/summary` — Node count, pod count, overall health status.
* `GET /cluster/nodes` — List all nodes + CPU/memory usage.
* `GET /cluster/nodes/:nodeName` — Single node detail.

#### 3. Pod APIs
*Core of the monitor.*
* `GET /pods` — List all pods across namespaces (status, restarts, CPU, memory).
* `GET /pods/:namespace` — Pods filtered by namespace.
* `GET /pods/:namespace/:podName` — Single pod detail.
* `GET /pods/:namespace/:podName/logs` — Last N log lines (stretch goal).

#### 4. Anomaly APIs
*Fed by `kube-pulse-analyzer`.*
* `GET /anomalies` — All active anomalies detected.
* `GET /anomalies/:namespace` — Anomalies scoped to a namespace.
* `GET /anomalies/types` — e.g., `cpu_spike`, `crash_loop`, `oom_kill`.

#### 5. Metrics APIs
*Raw numbers from collector.*
* `GET /metrics/pods/:namespace/:podName` — CPU + memory for a pod.
* `GET /metrics/nodes/:nodeName` — CPU + memory for a node.

---

### What Calls What

1.  **Client (curl/browser)**
    * ↓ *REST*
2.  **kube-pulse-gateway (Go)**
    * ↓ *gRPC*
3.  **kube-pulse-collector (Rust)** → Talks to Kubernetes API (via `kube-rs`).
4.  **kube-pulse-analyzer (Rust)** → Receives metrics, returns anomaly decisions.

---

### Build Order Recommendation
*Estimated effort: 3–5 hrs/week.*

| Phase | Focus | Goal |
| :--- | :--- | :--- |
| **Week 1** | `/health/*` | Smoke-test gRPC connections. |
| **Week 2–3** | `/cluster/nodes` + `/pods` | Real K8s data flowing to the frontend. |
| **Week 4** | `/metrics/*` | Performance numbers on screen. |
| **Week 5+** | `/anomalies/*` | Wired up once the analyzer logic is ready. |