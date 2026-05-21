package grpc

import (
	"fmt"

	healthpb "github.com/krish/kube-pulse-gateway/gen/health/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	Collector healthpb.HealthServiceClient
	Analyzer  healthpb.HealthServiceClient

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
		Collector:     healthpb.NewHealthServiceClient(collectorConn),
		Analyzer:      healthpb.NewHealthServiceClient(analyzerConn),
		collectorConn: collectorConn,
		analyzerConn:  analyzerConn,
	}, nil
}

func (c *Clients) Close() {
	c.collectorConn.Close()
	c.analyzerConn.Close()
}
