# Health APIs — Demo

Quick walkthrough of testing the three Health endpoints. No Minikube required — the health APIs only check if the services are reachable.

---

## 1. Run the Services

Open three separate terminals:

**Terminal 1 — Collector (port 50051):**
```powershell
cd kube-pulse-collector
cargo run
```

**Terminal 2 — Analyzer (port 50052):**
```powershell
cd kube-pulse-analyzer
cargo run
```

**Terminal 3 — Gateway (port 8080):**
```powershell
cd kube-pulse-gateway
go run main.go
```

---

## 2. Test the APIs

### GET /health

Gateway liveness check. No downstream calls — always returns `ok` as long as the gateway is running.

```powershell
curl http://localhost:8080/health
```

```json
{
  "status": "ok",
  "service": "kube-pulse-gateway",
  "timestamp": "2026-05-21T10:00:00Z"
}
```

---

### GET /health/collector

Gateway pings the Rust collector over gRPC.

```powershell
curl http://localhost:8080/health/collector
```

```json
{
  "status": "ok",
  "service": "kube-pulse-collector",
  "timestamp": "2026-05-21T10:00:01Z",
  "latency_ms": 4
}
```

---

### GET /health/analyzer

Gateway pings the Rust analyzer over gRPC.

```powershell
curl http://localhost:8080/health/analyzer
```

```json
{
  "status": "ok",
  "service": "kube-pulse-analyzer",
  "timestamp": "2026-05-21T10:00:01Z",
  "latency_ms": 3
}
```

---

## 3. Collector/Analyzer Down

If a service is not running, the gateway still returns **HTTP 200** — it reports the downstream state rather than failing itself.

```json
{
  "status": "unavailable",
  "service": "kube-pulse-collector",
  "timestamp": "2026-05-21T10:00:01Z",
  "error": "connection refused"
}
```
