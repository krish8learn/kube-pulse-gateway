package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	healthpb "github.com/krish/kube-pulse-gateway/gen/health/v1"
)

type healthResponse struct {
	Status    string  `json:"status"`
	Service   string  `json:"service"`
	Timestamp string  `json:"timestamp"`
	LatencyMs *int64  `json:"latency_ms,omitempty"`
	Error     *string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func GatewayHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, healthResponse{
		Status:    "ok",
		Service:   "kube-pulse-gateway",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func CollectorHealth(client healthpb.HealthServiceClient) http.HandlerFunc {
	return checkDownstream(client, "kube-pulse-collector")
}

func AnalyzerHealth(client healthpb.HealthServiceClient) http.HandlerFunc {
	return checkDownstream(client, "kube-pulse-analyzer")
}

func checkDownstream(client healthpb.HealthServiceClient, targetService string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		start := time.Now()
		resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "gateway"})
		latencyMs := time.Since(start).Milliseconds()

		if err != nil {
			log.Error().
				Str("service", targetService).
				Int64("latency_ms", latencyMs).
				Err(err).
				Msg("downstream health check failed")

			errStr := err.Error()
			writeJSON(w, healthResponse{
				Status:    "unavailable",
				Service:   targetService,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Error:     &errStr,
			})
			return
		}

		log.Info().
			Str("service", targetService).
			Int64("latency_ms", latencyMs).
			Str("status", resp.Status).
			Msg("downstream health check ok")

		writeJSON(w, healthResponse{
			Status:    resp.Status,
			Service:   resp.Service,
			Timestamp: resp.Timestamp,
			LatencyMs: &latencyMs,
		})
	}
}
