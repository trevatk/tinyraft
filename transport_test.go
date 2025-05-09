package tinyraft

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/trevatk/tinyraft/protos/raft/v1"
)

type TransportSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func (suite *TransportSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *TransportSuite) TestJoin() {

	assert := suite.Assert()
	ctx := context.TODO()

	cfg := testConfigBootstrap()

	tm := newTestModule()
	rs, tr, err := NewServerAndTransport(cfg, tm)
	assert.NoError(err)

	s2 := grpc.NewServer()
	s2.RegisterService(&pb.RaftServiceV1_ServiceDesc, tr)

	s1 := httptest.NewUnstartedServer(s2)
	s1.EnableHTTP2 = true

	tt := []struct {
		expected error
		in       *pb.JoinRequest
	}{
		{
			expected: nil,
			in: &pb.JoinRequest{
				AdvertiseAddr: "127.0.0.1:50052",
				Voter:         true,
				RequestedAt:   timestamppb.Now(),
			},
		},
	}

	for _, tc := range tt {
		rr, err := tr.Join(ctx, tc.in)
		assert.Equal(tc.expected, err)

		if tc.expected == nil {
			assert.NotEmpty(rr.AssignedId)
			assert.Equal("00000000-0000-0000-0000-000000000000", rr.LeaderId)
			assert.Equal(rs.nodeID, rr.LeaderId)
			assert.NotEmpty(rr.CompletedAt)
		}
	}
}

func (suite *TransportSuite) TestAppendEntries() {

	assert := suite.Assert()

	tm := newTestModule()

	cfg := testConfigBootstrap()
	rs, tr, err := NewServerAndTransport(cfg, tm)
	assert.NoError(err)

	assert.NoError(rs.Start(suite.ctx))
	defer func() { _ = rs.Shutdown(suite.ctx) }()

	tt := []struct {
		expected *pb.AppendEntriesResponse
		request  *pb.AppendEntriesRequest
	}{
		{
			// increment term
			request: &pb.AppendEntriesRequest{
				Term:        2, // incremented from default config
				LeaderId:    "",
				LogEntries:  make([]*pb.LogEntry, 0), // simulate heartbeat message
				RequestedAt: timestamppb.Now(),
			},
			expected: &pb.AppendEntriesResponse{
				Term:    2,
				Success: true,
			},
		},
	}

	for _, tc := range tt {
		rr, err := tr.AppendEntries(suite.ctx, tc.request)
		assert.NoError(err)

		assert.Equal(tc.expected.Term, rr.Term)
	}
}

func (suite *TransportSuite) TestApply() {}

func (suite *TransportSuite) TestVote() {}

func TestTransportSuite(t *testing.T) {
	suite.Run(t, new(TransportSuite))
}
