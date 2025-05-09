package tinyraft

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	pb "github.com/trevatk/tinyraft/protos/raft/v1"
)

type ServerBootstrapSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc

	s *Server
	t pb.RaftServiceV1Server
}

func (suite *ServerBootstrapSuite) SetupSuite() {
	assert := suite.Assert()

	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	cfg := testConfigBootstrap()
	tm := newTestModule()

	rs, tr, err := NewServerAndTransport(cfg, tm)
	assert.NoError(err)

	suite.s = rs
	suite.t = tr

	assert.NoError(suite.s.Start(suite.ctx))
}

func (suite *ServerBootstrapSuite) TestApply() {
	assert := suite.Assert()

	tt := []struct {
		expected error
		cmd      *pb.Cmd
	}{
		{
			// success
			expected: nil,
			cmd: &pb.Cmd{
				Payload: []byte("hello world"),
			},
		},
	}

	for _, tc := range tt {
		_, err := suite.t.Apply(suite.ctx, tc.cmd)
		assert.Equal(tc.expected, err)
	}
}

func (suite *ServerBootstrapSuite) TeardownSuite() {
	assert := suite.Assert()
	assert.NoError(suite.s.Shutdown(suite.ctx))
	suite.cancel()
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerBootstrapSuite))
}
