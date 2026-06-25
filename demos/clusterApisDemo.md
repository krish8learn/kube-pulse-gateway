# Cluster APIs — Demo

End-to-end walkthrough of setting up the environment and testing the three Cluster Overview endpoints.

---

## Prerequisites

- Docker Desktop installed and running
- `kube-pulse-collector` and `kube-pulse-gateway` built and ready to run

---

## 1. Install Minikube & kubectl

Open a PowerShell terminal and run:

```powershell
winget install Kubernetes.minikube
winget install Kubernetes.kubectl
```

Close and reopen the terminal after installation so the PATH changes take effect.

---

## 2. Start Minikube

```powershell
minikube start --driver=docker
```

Verify the cluster is up:

```powershell
minikube status
kubectl get nodes
```

Expected output from `kubectl get nodes`:

```
NAME       STATUS   ROLES           AGE   VERSION
minikube   Ready    control-plane   1m    v1.35.1
```

---

## 3. Run the Services

Open three separate terminals:

**Terminal 1 — Collector (Rust gRPC server on port 50051):**
```powershell
cd kube-pulse-collector
cargo run
```

**Terminal 2 — Gateway (Go HTTP server on port 8080):**
```powershell
cd kube-pulse-gateway
go run main.go
```

---

## 4. Test the APIs

### GET /cluster/summary

Returns node count, pod count, and overall cluster health.

```powershell
curl http://localhost:8080/cluster/summary
```

Expected response:

```json
{
  "total_nodes": 1,
  "ready_nodes": 1,
  "total_pods": 7,
  "running_pods": 7,
  "latency_ms": 14
}
```

![Cluster Summary](./images/cluster-summary.png)

---

### GET /cluster/nodes

Returns all nodes with CPU/memory capacity and pod count.

```powershell
curl http://localhost:8080/cluster/nodes
```

Expected response:

```json
{
  "nodes": [
    {
      "name": "minikube",
      "status": "Ready",
      "cpu_capacity": "8",
      "memory_capacity": "7866392Ki",
      "cpu_allocatable": "8",
      "memory_allocatable": "7866392Ki",
      "pod_count": 7,
      "kubernetes_version": "v1.35.1",
      "age": "2026-06-25T16:45:59+00:00"
    }
  ],
  "latency_ms": 46
}
```

![Cluster All Nodes](./images/cluster-nodes.png)

---

### GET /cluster/nodes/:nodeName

Returns detail for a single node. The node name is `minikube` by default when using Minikube.

```powershell
curl http://localhost:8080/cluster/nodes/minikube
```

Expected response:

```json
{
  "node": {
    "name": "minikube",
    "status": "Ready",
    "cpu_capacity": "8",
    "memory_capacity": "7866392Ki",
    "cpu_allocatable": "8",
    "memory_allocatable": "7866392Ki",
    "pod_count": 7,
    "kubernetes_version": "v1.35.1",
    "age": "2026-06-25T16:45:59+00:00"
  },
  "latency_ms": 24
}
```

![Cluster Node Detail](./images/cluster-node-detail.png)

---

### Node not found

```powershell
curl http://localhost:8080/cluster/nodes/doesnotexist
```

Expected response (HTTP 200, error field):

```json
{
  "error": "rpc error: code = NotFound desc = node \"doesnotexist\" not found"
}
```

---

## 5. Stop Minikube

When done testing:

```powershell
minikube stop
```

To completely delete the cluster and free disk space:

```powershell
minikube delete
```

---

## Notes

- All endpoints return **HTTP 200** even when the collector is down — the `error` field in the JSON body signals the failure.
- `cpu_capacity` / `memory_capacity` are raw Kubernetes quantity strings (e.g. `"8"`, `"7866392Ki"`). Live usage metrics are a future phase.
- `pod_count` counts pods currently scheduled on that node — no metrics-server required.
