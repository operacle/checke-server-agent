
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "monitoring-agent/proto"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.MonitoringServiceClient
	ctx    context.Context
}

func NewGRPCClient(serverAddress string) (*GRPCClient, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewMonitoringServiceClient(conn)
	
	return &GRPCClient{
		conn:   conn,
		client: client,
		ctx:    context.Background(),
	}, nil
}

func (c *GRPCClient) SendMetrics(agentID string, cpuUsage, memoryUsage, diskUsage float64, goRoutines int, status string) (*pb.MetricsResponse, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	req := &pb.MetricsRequest{
		AgentId:     agentID,
		Timestamp:   time.Now().Unix(),
		CpuUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		DiskUsage:   diskUsage,
		NetworkStats: &pb.NetworkStats{
			BytesSent:       0, // Would implement actual network stats
			BytesReceived:   0,
			PacketsSent:     0,
			PacketsReceived: 0,
		},
		Uptime:      int64(time.Hour.Seconds()), // Placeholder
		GoRoutines:  int32(goRoutines),
		Status:      status,
	}

	return c.client.SendMetrics(ctx, req)
}

func (c *GRPCClient) SendHealthCheck(agentID, version string) (*pb.HealthResponse, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	req := &pb.HealthRequest{
		AgentId:   agentID,
		Timestamp: time.Now().Unix(),
		Version:   version,
	}

	return c.client.SendHealthCheck(ctx, req)
}

func (c *GRPCClient) GetRemoteCommand(agentID string) (*pb.CommandResponse, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	req := &pb.CommandRequest{
		AgentId: agentID,
	}

	return c.client.GetRemoteCommand(ctx, req)
}

func (c *GRPCClient) UpdateAgentStatus(agentID, status, message string) (*pb.StatusResponse, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	req := &pb.StatusRequest{
		AgentId: agentID,
		Status:  status,
		Message: message,
	}

	return c.client.UpdateAgentStatus(ctx, req)
}

func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}