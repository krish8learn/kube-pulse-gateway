Schema Design — Cluster Overview APIs

## REST API Responses (Gateway → Client)

### GET /cluster/summary

```json
{
  "total_nodes": 1,
  "ready_nodes": 1,
  "total_pods": 12,
  "running_pods": 11,
  "latency_ms": 18
}
```

Collector down:
```json
{
  "error": "rpc error: code = Unavailable desc = connection refused"
}
```

### GET /cluster/nodes

```json
{
  "nodes": [
    {
      "name": "minikube",
      "status": "Ready",
      "cpu_capacity": "4",
      "memory_capacity": "8Gi",
      "cpu_allocatable": "4",
      "memory_allocatable": "7952Mi",
      "pod_count": 12,
      "kubernetes_version": "v1.32.0",
      "age": "2026-06-01T10:00:00Z"
    }
  ],
  "latency_ms": 22
}
```

### GET /cluster/nodes/:nodeName

```json
{
  "node": {
    "name": "minikube",
    "status": "Ready",
    "cpu_capacity": "4",
    "memory_capacity": "8Gi",
    "cpu_allocatable": "4",
    "memory_allocatable": "7952Mi",
    "pod_count": 12,
    "kubernetes_version": "v1.32.0",
    "age": "2026-06-01T10:00:00Z"
  },
  "latency_ms": 19
}
```

Node not found:
```json
{
  "error": "rpc error: code = NotFound desc = node \"worker-99\" not found"
}
```

**Notes:**
- All endpoints return HTTP 200 — gateway stays alive even when collector or K8s is down.
- `cpu_capacity` / `memory_capacity` etc. are raw Kubernetes quantity strings (`"4"`, `"8Gi"`, `"500m"`). No parsing — pass them through as-is.
- `pod_count` = pods scheduled on that node (`spec.nodeName`), derived from a full pod list. No metrics-server required.
- Live CPU/memory **usage** belongs to the future `/metrics` phase. This phase exposes only `capacity` and `allocatable` from the `Node` object.

## .proto Contract (gRPC — the real source of truth)

```proto
syntax = "proto3";

package kubepulse.cluster.v1;

option go_package = "github.com/krish/kube-pulse-gateway/gen/cluster/v1";

// Used by collector only (analyzer has no cluster data)
service ClusterService {
  rpc GetClusterSummary(GetClusterSummaryRequest) returns (GetClusterSummaryResponse);
  rpc GetNodes(GetNodesRequest)                   returns (GetNodesResponse);
  rpc GetNode(GetNodeRequest)                     returns (GetNodeResponse);
}

message GetClusterSummaryRequest {}
message GetNodesRequest {}
message GetNodeRequest {
  string name = 1;
}

message NodeInfo {
  string name               = 1;
  string status             = 2;  // "Ready" | "NotReady"
  string cpu_capacity       = 3;
  string memory_capacity    = 4;
  string cpu_allocatable    = 5;
  string memory_allocatable = 6;
  int32  pod_count          = 7;
  string kubernetes_version = 8;
  string age                = 9;  // RFC3339 creation timestamp
}

message ClusterSummary {
  int32 total_nodes  = 1;
  int32 ready_nodes  = 2;
  int32 total_pods   = 3;
  int32 running_pods = 4;
}

message GetClusterSummaryResponse { ClusterSummary summary = 1; }
message GetNodesResponse           { repeated NodeInfo nodes = 1; }
message GetNodeResponse            { NodeInfo node = 1; }
```

**Why empty request messages?** Leaves room to add filters (label selectors, namespace scope) later without a breaking proto change.

**Why string quantities?** Kubernetes resource quantities (`"4"`, `"8Gi"`, `"500m"`) don't map cleanly to integers. Passing the raw string avoids silent truncation; the UI or a future metrics layer can parse if needed.

## Error Case — Node Not Found

If a node name doesn't exist, the collector returns gRPC `NOT_FOUND`. The gateway surfaces this in the `error` field at HTTP 200 — same pattern as health APIs.

```json
{
  "error": "rpc error: code = NotFound desc = node \"worker-99\" not found"
}
```

## Summary — Proto File Contents

| Message/Service | Fields |
| --- | --- |
| `GetClusterSummaryRequest` | *(empty)* |
| `GetNodesRequest` | *(empty)* |
| `GetNodeRequest` | `name` (string) |
| `NodeInfo` | `name`, `status`, `cpu_capacity`, `memory_capacity`, `cpu_allocatable`, `memory_allocatable`, `pod_count`, `kubernetes_version`, `age` |
| `ClusterSummary` | `total_nodes`, `ready_nodes`, `total_pods`, `running_pods` |
| `ClusterService` | `rpc GetClusterSummary`, `rpc GetNodes`, `rpc GetNode` |

## Collector Implementation Notes

- `kube::Client::try_default()` auto-detects: kubeconfig locally, in-cluster service account in a pod.
- Node `status` derived from `node.status.conditions` — look for `type_ == "Ready" && status == "True"`.
- `pod_count` on each node: filter all pods by `pod.spec.node_name == node.name`.
- Capacity/allocatable values come from `node.status.capacity` and `node.status.allocatable` — both are `BTreeMap<String, Quantity>` where `Quantity` wraps a raw string (`.0`).

## Proto Compilation

Gateway:
```bash
protoc \
  --proto_path=proto \
  --go_out=gen \
  --go_opt=paths=source_relative \
  --go-grpc_out=gen \
  --go-grpc_opt=paths=source_relative \
  cluster/v1/cluster.proto
```

Collector (handled automatically by `build.rs` via `tonic-build`).
