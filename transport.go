package tinyraft

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/trevatk/tinyraft/protos/raft/v1"
)

func grpcStatusInternalError() error {
	return status.Error(codes.Internal, codes.Internal.String())
}

func grpcStatusInvalidArgument() error {
	return status.Error(codes.InvalidArgument, codes.InvalidArgument.String())
}

func grpcStatusFailedCondition() error {
	return status.Error(codes.FailedPrecondition, codes.FailedPrecondition.String())
}

type transport struct {
	pb.UnimplementedRaftServiceV1Server

	rs *Server
}

// interface compliance
var _ pb.RaftServiceV1Server = (*transport)(nil)

// NewTransport
func newTransport(server *Server) pb.RaftServiceV1Server {
	return &transport{
		rs: server,
	}
}

// Apply
func (t transport) Apply(ctx context.Context, in *pb.Cmd) (*emptypb.Empty, error) {
	grpclog.Info("Apply")

	if t.rs.state == leader {
		err := t.rs.apply(in.Payload)
		if err != nil {
			return nil, grpcStatusInternalError()
		}
		return newEmptyResponse(), nil
	}

	if t.rs.state != leader {
		// TODO
		// forward request to leader
		return newEmptyResponse(), nil
	}

	if err := t.rs.apply(in.Payload); err != nil {
		return nil, grpcStatusInternalError()
	}

	return newEmptyResponse(), nil
}

// AppendEntries
func (t transport) AppendEntries(ctx context.Context, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	grpclog.Info("AppendEntries")
	t.rs.ticker.Reset(t.rs.defaultTimeout)
	// requested term is before current term
	if in.Term < t.rs.term {
		return nil, grpcStatusFailedCondition()
	}

	if in.Term > t.rs.term {
		t.rs.state = follower
		t.rs.leaderID = in.LeaderId
		t.rs.term = in.Term
		return newAppendEntriesResponse(t.rs.term, true), nil
	}

	for _, entry := range in.LogEntries {
		t.rs.entries = append(t.rs.entries, newLog(entry.Term, entry.Index, entry.Cmd))
		t.rs.commitIndex += 1
	}

	return newAppendEntriesResponse(t.rs.term, true), nil
}

// Join
func (t transport) Join(ctx context.Context, in *pb.JoinRequest) (*pb.JoinResponse, error) {
	grpclog.Info("Join")

	if in.RequestedAt == nil {
		return nil, grpcStatusInvalidArgument()
	}

	assignedID := t.rs.addNode(in.AdvertiseAddr, in.Voter)

	return newJoinResponse(t.rs.term, assignedID, t.rs.leaderID), nil
}

// Vote
func (t transport) Vote(ctx context.Context, in *pb.VoteRequest) (*pb.VoteResponse, error) {
	grpclog.Info("Vote")

	if in.Term < t.rs.term {
		return newVoteResponse(t.rs.term, false), nil
	}

	return newVoteResponse(t.rs.term, true), nil
}

func newJoinResponse(term uint64, assignedID, leaderID string) *pb.JoinResponse {
	return &pb.JoinResponse{
		AssignedId:  assignedID,
		LeaderId:    leaderID,
		Term:        term,
		CompletedAt: timestamppb.Now(),
	}
}

func newAppendEntriesResponse(term uint64, success bool) *pb.AppendEntriesResponse {
	return &pb.AppendEntriesResponse{
		Term:    term,
		Success: success,
	}
}

func newVoteResponse(term uint64, success bool) *pb.VoteResponse {
	return &pb.VoteResponse{
		Term:        term,
		VoteGranted: success,
	}
}

func newEmptyResponse() *emptypb.Empty {
	return &emptypb.Empty{}
}
