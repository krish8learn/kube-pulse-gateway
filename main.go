package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	grpcclient "github.com/krish/kube-pulse-gateway/internal/grpc"
	"github.com/krish/kube-pulse-gateway/internal/handler"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339

	port := envOr("GATEWAY_PORT", "8080")
	collectorAddr := envOr("COLLECTOR_ADDR", "localhost:50051")
	analyzerAddr := envOr("ANALYZER_ADDR", "localhost:50052")

	clients, err := grpcclient.NewClients(collectorAddr, analyzerAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialise grpc clients")
	}
	defer clients.Close()

	log.Info().
		Str("port", port).
		Str("collector_addr", collectorAddr).
		Str("analyzer_addr", analyzerAddr).
		Msg("kube-pulse-gateway starting")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.GatewayHealth)
	mux.HandleFunc("GET /health/collector", handler.CollectorHealth(clients.Collector))
	mux.HandleFunc("GET /health/analyzer", handler.AnalyzerHealth(clients.Analyzer))

	mux.HandleFunc("GET /cluster/summary", handler.ClusterSummary(clients.ClusterCollector))
	mux.HandleFunc("GET /cluster/nodes", handler.ListNodes(clients.ClusterCollector))
	mux.HandleFunc("GET /cluster/nodes/{nodeName}", handler.GetNode(clients.ClusterCollector))

	addr := fmt.Sprintf(":%s", port)
	log.Info().Str("addr", addr).Msg("listening")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
