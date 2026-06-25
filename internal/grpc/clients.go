package grpc

import (
	"fmt"

	clusterpb "github.com/krish/kube-pulse-gateway/gen/cluster/v1"
	healthpb "github.com/krish/kube-pulse-gateway/gen/health/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {

	// health API clients for the gRPC services
	Collector healthpb.HealthServiceClient
	Analyzer  healthpb.HealthServiceClient

	// cluster API client for the gRPC service
	ClusterCollector clusterpb.ClusterServiceClient

	// gRPC connections for cleanup
	collectorConn *grpc.ClientConn
	analyzerConn  *grpc.ClientConn
}

func NewClients(collectorAddr, analyzerAddr string) (*Clients, error) {
	collectorConn, err := grpc.NewClient(collectorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial collector: %w", err)
	}

	analyzerConn, err := grpc.NewClient(analyzerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		collectorConn.Close()
		return nil, fmt.Errorf("dial analyzer: %w", err)
	}

	return &Clients{
		Collector:        healthpb.NewHealthServiceClient(collectorConn),
		Analyzer:         healthpb.NewHealthServiceClient(analyzerConn),
		ClusterCollector: clusterpb.NewClusterServiceClient(collectorConn),
		collectorConn:    collectorConn,
		analyzerConn:     analyzerConn,
	}, nil
}

func (c *Clients) Close() {
	c.collectorConn.Close()
	c.analyzerConn.Close()
}
