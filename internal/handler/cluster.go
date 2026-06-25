package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	clusterpb "github.com/krish/kube-pulse-gateway/gen/cluster/v1"
)

type clusterSummaryResponse struct {
	TotalNodes  int32   `json:"total_nodes"`
	ReadyNodes  int32   `json:"ready_nodes"`
	TotalPods   int32   `json:"total_pods"`
	RunningPods int32   `json:"running_pods"`
	LatencyMs   *int64  `json:"latency_ms,omitempty"`
	Error       *string `json:"error,omitempty"`
}

type nodeInfoJSON struct {
	Name              string `json:"name"`
	Status            string `json:"status"`
	CpuCapacity       string `json:"cpu_capacity"`
	MemoryCapacity    string `json:"memory_capacity"`
	CpuAllocatable    string `json:"cpu_allocatable"`
	MemoryAllocatable string `json:"memory_allocatable"`
	PodCount          int32  `json:"pod_count"`
	KubernetesVersion string `json:"kubernetes_version"`
	Age               string `json:"age"`
}

type nodesResponse struct {
	Nodes     []nodeInfoJSON `json:"nodes"`
	LatencyMs *int64         `json:"latency_ms,omitempty"`
	Error     *string        `json:"error,omitempty"`
}

type nodeResponse struct {
	Node      *nodeInfoJSON `json:"node,omitempty"`
	LatencyMs *int64        `json:"latency_ms,omitempty"`
	Error     *string       `json:"error,omitempty"`
}

func protoToNodeInfoJSON(n *clusterpb.NodeInfo) nodeInfoJSON {
	return nodeInfoJSON{
		Name:              n.Name,
		Status:            n.Status,
		CpuCapacity:       n.CpuCapacity,
		MemoryCapacity:    n.MemoryCapacity,
		CpuAllocatable:    n.CpuAllocatable,
		MemoryAllocatable: n.MemoryAllocatable,
		PodCount:          n.PodCount,
		KubernetesVersion: n.KubernetesVersion,
		Age:               n.Age,
	}
}

func ClusterSummary(client clusterpb.ClusterServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		start := time.Now()
		resp, err := client.GetClusterSummary(ctx, &clusterpb.GetClusterSummaryRequest{})
		latencyMs := time.Since(start).Milliseconds()

		if err != nil {
			errStr := err.Error()
			log.Error().Err(err).Int64("latency_ms", latencyMs).Msg("get cluster summary failed")
			writeJSON(w, clusterSummaryResponse{Error: &errStr})
			return
		}

		s := resp.Summary
		log.Info().
			Int64("latency_ms", latencyMs).
			Int32("total_nodes", s.TotalNodes).
			Int32("total_pods", s.TotalPods).
			Msg("cluster summary ok")

		writeJSON(w, clusterSummaryResponse{
			TotalNodes:  s.TotalNodes,
			ReadyNodes:  s.ReadyNodes,
			TotalPods:   s.TotalPods,
			RunningPods: s.RunningPods,
			LatencyMs:   &latencyMs,
		})
	}
}

func ListNodes(client clusterpb.ClusterServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		start := time.Now()
		resp, err := client.GetNodes(ctx, &clusterpb.GetNodesRequest{})
		latencyMs := time.Since(start).Milliseconds()

		if err != nil {
			errStr := err.Error()
			log.Error().Err(err).Int64("latency_ms", latencyMs).Msg("list nodes failed")
			writeJSON(w, nodesResponse{Error: &errStr})
			return
		}

		nodes := make([]nodeInfoJSON, 0, len(resp.Nodes))
		for _, n := range resp.Nodes {
			nodes = append(nodes, protoToNodeInfoJSON(n))
		}

		log.Info().Int64("latency_ms", latencyMs).Int("count", len(nodes)).Msg("list nodes ok")
		writeJSON(w, nodesResponse{Nodes: nodes, LatencyMs: &latencyMs})
	}
}

func GetNode(client clusterpb.ClusterServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.PathValue("nodeName")

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		start := time.Now()
		resp, err := client.GetNode(ctx, &clusterpb.GetNodeRequest{Name: nodeName})
		latencyMs := time.Since(start).Milliseconds()

		if err != nil {
			errStr := err.Error()
			log.Error().Err(err).Str("node_name", nodeName).Int64("latency_ms", latencyMs).Msg("get node failed")
			writeJSON(w, nodeResponse{Error: &errStr})
			return
		}

		node := protoToNodeInfoJSON(resp.Node)
		log.Info().Str("node_name", nodeName).Int64("latency_ms", latencyMs).Msg("get node ok")
		writeJSON(w, nodeResponse{Node: &node, LatencyMs: &latencyMs})
	}
}
