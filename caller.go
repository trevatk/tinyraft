package tinyraft

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/trevatk/tinyraft/protos/raft/v1"
)

// Caller
type Caller interface {
	Apply(context.Context, []byte) error
}

type client struct {
	conn *grpc.ClientConn
}

// NewCaller
func NewCaller(addr string) (Caller, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient: %w", err)
	}
	return &client{conn: conn}, nil
}

// internal client constructor
func newClientConn(addr string) (pb.RaftServiceV1Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient: %s %w", addr, err)
	}
	return pb.NewRaftServiceV1Client(conn), nil
}

// Apply
func (c *client) Apply(ctx context.Context, cmd []byte) error {
	// timeout, cancel := context.WithTimeout(ctx, time.Millisecond*250)
	// defer cancel()

	cli := pb.NewRaftServiceV1Client(c.conn)
	_, err := cli.Apply(ctx, &pb.Cmd{Payload: cmd})
	if err != nil {
		return fmt.Errorf("gRPC execute apply: %w", err)
	}

	return nil
}

// func transformLogEntries(s []Log) []*pb.LogEntry {
// 	out := make([]*pb.LogEntry, 0, len(s))
// 	for _, e := range s {
// 		out = append(out, transformLogEntry(e))
// 	}
// 	return out
// }

// func transformLogEntry(log Log) *pb.LogEntry {
// 	return &pb.LogEntry{
// 		Index: log.Index,
// 		Term:  log.Term,
// 		Cmd:   log.Cmd,
// 	}
// }
