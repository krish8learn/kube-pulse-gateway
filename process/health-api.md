# Health API — Build Process

How the health API was built, why each decision was made, and how to extend it.

---

## What We're Building

Three REST endpoints served by the Go gateway:

| Endpoint | What it does |
|---|---|
| `GET /health` | Gateway liveness — no downstream calls |
| `GET /health/collector` | Gateway pings the Rust collector over gRPC, wraps the response |
| `GET /health/analyzer` | Gateway pings the Rust analyzer over gRPC, wraps the response |

The `/health/collector` and `/health/analyzer` endpoints are the important ones — they exercise the full gRPC pipe between Go and Rust before any real K8s data is involved.

---

## Repository Layout After This Build

```
kube-pulse-gateway/           ← Go HTTP gateway
├── process/
│   └── health-api.md         ← this file
├── proto/
│   └── health/v1/
│       └── health.proto      ← source of truth for the gRPC contract
├── gen/
│   └── health/v1/
│       ├── health.pb.go      ← generated (do not edit by hand)
│       └── health_grpc.pb.go ← generated (do not edit by hand)
├── internal/
│   ├── grpc/
│   │   └── clients.go        ← gRPC client connections to Rust services
│   └── handler/
│       └── health.go         ← HTTP handler functions
├── main.go                   ← HTTP server, route wiring
└── go.mod

kube-pulse-collector/         ← Rust gRPC server (port 50051)
├── proto/
│   └── health/v1/
│       └── health.proto      ← same file, copied from gateway repo
├── src/
│   ├── main.rs               ← tonic server startup
│   └── health_service.rs     ← HealthService gRPC implementation
├── build.rs                  ← tells cargo to compile the proto at build time
└── Cargo.toml

kube-pulse-analyzer/          ← Rust gRPC server (port 50052)
├── proto/
│   └── health/v1/
│       └── health.proto      ← same file, copied from gateway repo
├── src/
│   ├── main.rs               ← tonic server startup
│   └── health_service.rs     ← HealthService gRPC implementation (identical pattern to collector)
├── build.rs                  ← tells cargo to compile the proto at build time
└── Cargo.toml
```

---

## Prerequisites

### Install protoc Go plugins (one-time)

`protoc` compiles `.proto` files into Go code. It needs two plugins for Go + gRPC:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

After this, `protoc-gen-go` and `protoc-gen-go-grpc` binaries live in `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is in your `PATH`.

### Rust: no extra steps

`tonic-build` (used in `build.rs`) calls `protoc` automatically at `cargo build` time. Rust does not need separate plugin installs.

---

## Step 1 — The `.proto` File

**Location:** `proto/health/v1/health.proto` (same file in both repos)

This is the **contract** between the Go gateway (client) and the Rust collector/analyzer (server). Both sides generate their own code from it — Go generates a client stub, Rust generates a server trait.

```proto
syntax = "proto3";
package kubepulse.health.v1;
option go_package = "github.com/krish/kube-pulse-gateway/gen/health/v1";

service HealthService {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
}

message HealthCheckRequest {
  string service = 1;
}

message HealthCheckResponse {
  string status    = 1;  // "ok" or "unavailable"
  string service   = 2;
  string timestamp = 3;  // RFC3339 UTC
}
```

**Why one service for both collector and analyzer?** They implement the same interface — same RPC shape, different server. The Go client connects to whichever address it's configured with. This avoids duplicating the proto just to change the package name.

---

## Step 2 — Generate Go Code from Proto

Run this from the root of `kube-pulse-gateway`:

```sh
protoc \
  --proto_path=proto \
  --go_out=gen \
  --go_opt=paths=source_relative \
  --go-grpc_out=gen \
  --go-grpc_opt=paths=source_relative \
  health/v1/health.proto
```

This produces two files in `gen/health/v1/`:
- `health.pb.go` — the message structs (`HealthCheckRequest`, `HealthCheckResponse`)
- `health_grpc.pb.go` — the client stub (`HealthServiceClient`) and server interface (`HealthServiceServer`)

**You never edit these files.** If the proto changes, re-run the command above.

---

## Step 3 — Rust Services: Compile Proto via build.rs

This step is **identical in both `kube-pulse-collector` and `kube-pulse-analyzer`**. `tonic-build` in `build.rs` calls `protoc` automatically during `cargo build`. No manual command needed.

```rust
// build.rs  (same file in both repos)
fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::compile_protos("proto/health/v1/health.proto")?;
    Ok(())
}
```

Generated Rust code lands in a temp directory managed by Cargo (`OUT_DIR`). You reference it in code with `include_proto!("kubepulse.health.v1")`.

---

## Step 4 — Rust Services: HealthService Implementation

Both `kube-pulse-collector` and `kube-pulse-analyzer` implement the same `HealthService` gRPC server using the same pattern. The only differences are the service name string and the port they bind on.

**Key struct:** `HealthServiceImpl` in `src/health_service.rs` — present in both repos.

```rust
pub struct HealthServiceImpl;

#[tonic::async_trait]
impl HealthService for HealthServiceImpl {
    async fn check(&self, _request: Request<HealthCheckRequest>)
        -> Result<Response<HealthCheckResponse>, Status>
    {
        let reply = HealthCheckResponse {
            status: "ok".into(),
            service: "<service-name>".into(), // "kube-pulse-collector" or "kube-pulse-analyzer"
            timestamp: Utc::now().to_rfc3339(),
        };
        Ok(Response::new(reply))
    }
}
```

The `main.rs` in each repo starts a `tonic::transport::Server` and registers this service:

| Repo | Binds on |
|---|---|
| `kube-pulse-collector` | `0.0.0.0:50051` |
| `kube-pulse-analyzer` | `0.0.0.0:50052` |

---

## Step 5 — Go Gateway: gRPC Client

`internal/grpc/clients.go` holds the dial logic. It creates a `HealthServiceClient` (generated stub) pointing at the collector's address.

```go
conn, err := grpc.NewClient(collectorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
client := healthpb.NewHealthServiceClient(conn)
```

`insecure.NewCredentials()` — no TLS for local development. When deploying to a real cluster this gets replaced with mTLS.

---

## Step 6 — Go Gateway: HTTP Handlers

`internal/handler/health.go` has three functions:

| Function | What it does |
|---|---|
| `GatewayHealth` | Writes `{"status":"ok","service":"kube-pulse-gateway",...}` immediately |
| `CollectorHealth` | Calls gRPC `Check`, measures latency, writes JSON |
| `AnalyzerHealth` | Same as above but targets the analyzer address |

If the gRPC call fails (connection refused, timeout), the handler returns HTTP 200 with `"status":"unavailable"` and an `"error"` field. The gateway itself is alive — it's reporting that the downstream is not.

---

## Step 7 — Go Gateway: HTTP Server

`main.go` uses `net/http` (`http.NewServeMux`) — no external router needed for this phase.

```
GET /health           → handler.GatewayHealth
GET /health/collector → handler.CollectorHealth
GET /health/analyzer  → handler.AnalyzerHealth
```

Server binds on `:8080` by default.

---

## How to Run (Local / Minikube)

### 1. Start the Rust collector

```sh
cd kube-pulse-collector
cargo run
# listens on 0.0.0.0:50051
```

### 2. Start the Rust analyzer

```sh
cd kube-pulse-analyzer
cargo run
# listens on 0.0.0.0:50052
```

### 3. Start the Go gateway

```sh
cd kube-pulse-gateway
go run main.go
# listens on :8080
```

### 4. Test with Insomnia or curl

#### GET /health

Gateway liveness — no downstream calls. Always returns `"ok"` as long as the gateway process is running.

```sh
curl -s http://localhost:8080/health | jq
```

Expected response:

```json
{
  "status": "ok",
  "service": "kube-pulse-gateway",
  "timestamp": "2026-05-21T10:00:00Z"
}
```

---

#### GET /health/collector

Gateway pings the Rust collector over gRPC and wraps the result.

```sh
curl -s http://localhost:8080/health/collector | jq
```

Expected response (collector running):

```json
{
  "status": "ok",
  "service": "kube-pulse-collector",
  "timestamp": "2026-05-21T10:00:01Z",
  "latency_ms": 4
}
```

Expected response (collector not running):

```json
{
  "status": "unavailable",
  "service": "kube-pulse-collector",
  "timestamp": "2026-05-21T10:00:01Z",
  "error": "connection refused"
}
```

> HTTP status is always 200. The gateway is alive — it is reporting the downstream state, not failing itself.

---

#### GET /health/analyzer

Same shape as `/health/collector` but targets the analyzer on port `50052`.

```sh
curl -s http://localhost:8080/health/analyzer | jq
```

Expected response (analyzer running):

```json
{
  "status": "ok",
  "service": "kube-pulse-analyzer",
  "timestamp": "2026-05-21T10:00:01Z",
  "latency_ms": 3
}
```

Expected response (analyzer not running):

```json
{
  "status": "unavailable",
  "service": "kube-pulse-analyzer",
  "timestamp": "2026-05-21T10:00:01Z",
  "error": "connection refused"
}
```

> HTTP status is always 200 — same reasoning as the collector endpoint above.

---

> **`jq` not installed?** Drop the `| jq` part — the raw JSON still prints, just without formatting.

---

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `GATEWAY_PORT` | `8080` | HTTP listen port |
| `COLLECTOR_ADDR` | `localhost:50051` | gRPC address of the collector |
| `ANALYZER_ADDR` | `localhost:50052` | gRPC address of the analyzer |

---

## What Comes Next (Phase 2)

Phase 2 adds `kube-rs` to the collector to talk to the Kubernetes API. The gRPC contract grows a new `CollectorService` (separate from `HealthService`) with RPCs like `ListNodes` and `ListPods`. The health plumbing built here does not change.
