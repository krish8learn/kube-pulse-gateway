Schema Design — Health APIs

## REST API Responses (Gateway → Client)

### GET /health
Gateway only, no gRPC call:

```json
{
  "status": "ok",
  "service": "kube-pulse-gateway",
  "version": "0.1.0",
  "timestamp": "2026-04-22T10:00:00Z"
}
```

### GET /health/collector and GET /health/analyzer
Gateway calls Rust via gRPC, wraps response:

```json
{
  "status": "ok",
  "service": "kube-pulse-collector",
  "timestamp": "2026-04-22T10:00:00Z",
  "latency_ms": 12
}
```

**Notes:**
- `latency_ms` — gateway measures round-trip gRPC ping time. Costs nothing to add now, useful signal later.
- `status` is either "ok" or "unavailable" — no other values for health.

## .proto Contract (gRPC — the real source of truth)

```proto
syntax = "proto3";

package kubepulse.health.v1;

option go_package = "github.com/krish/kube-pulse-gateway/gen/health/v1";

// Used by BOTH collector and analyzer
service HealthService {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
}

message HealthCheckRequest {
  string service = 1; // caller identifies itself: "gateway"
}

message HealthCheckResponse {
  string status    = 1; // "ok" or "unavailable"
  string service   = 2; // responder identifies itself: "collector"
  string timestamp = 3; // RFC3339 UTC
}
```

**Why one service, not two?** Collector and Analyzer both implement the same HealthService contract — same interface, different servers. Think of it like a Go interface: HealthChecker implemented by two structs.

## Error Case — Service Down

If the Rust service is down, the Gateway returns HTTP 200 but with degraded status — do **not** return HTTP 503. Reasoning: the gateway itself is alive; the health endpoint's job is to report state, not fail.

```json
{
  "status": "unavailable",
  "service": "kube-pulse-collector",
  "timestamp": "2026-04-22T10:00:00Z",
  "latency_ms": null,
  "error": "connection refused"
}
```

## Summary — Proto File Contents

| Message/Service | Fields |
| --- | --- |
| `HealthCheckRequest` | `service` (string) |
| `HealthCheckResponse` | `status`, `service`, `timestamp` (all string) |
| `HealthService` | `rpc Check(HealthCheckRequest) returns (HealthCheckResponse)` |