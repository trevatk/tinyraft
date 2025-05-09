package tinyraft

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	pb "github.com/trevatk/tinyraft/protos/raft/v1"
)

type state int

const (
	leader state = iota
	candidate
	follower
)

// Raft
type Raft interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

// Server
type Server struct {
	nodeID string

	// bridge between raft and custom implementation
	md Module

	// lifecycle
	g          *errgroup.Group
	ctx        context.Context
	cancelFunc context.CancelFunc

	ticker         *time.Ticker
	defaultTimeout time.Duration

	// persistent state
	term     uint64
	votedFor *string
	entries  []Log
	leaderID string

	state       state
	commitIndex uint64
	lastApplied uint64

	// volatile state on leaders
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	voters map[string]string // [nodeID]addr
	peers  map[string]string // [nodeID]addr

	listenAddr    string
	advertiseAddr string
}

// interface compliance
var _ Raft = (*Server)(nil)

// NewServerAndTransport
func NewServerAndTransport(cfg *Config, module Module) (*Server, pb.RaftServiceV1Server, error) {
	log.SetPrefix(cfg.NodeID + " ")
	log.Println(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	d := time.Duration(cfg.TimeoutMS * int(time.Millisecond))
	rs := &Server{
		nodeID:         cfg.NodeID,
		md:             module,
		g:              g,
		ctx:            ctx,
		cancelFunc:     cancel,
		term:           0,
		votedFor:       nil,
		commitIndex:    0,
		lastApplied:    0,
		matchIndex:     make(map[string]uint64),
		nextIndex:      make(map[string]uint64, 0),
		entries:        make([]Log, 0),
		defaultTimeout: d,
		ticker:         time.NewTicker(time.Second * 3),
		voters:         make(map[string]string),
		peers:          make(map[string]string),
		state:          follower,

		// networking
		listenAddr:    cfg.ListenAddr,
		advertiseAddr: cfg.AdvertiseAddr,
	}

	if cfg.Bootstrap {
		rs.leaderID = cfg.NodeID
	}

	if len(cfg.Join) > 0 && !cfg.Bootstrap {
		for _, addr := range cfg.Join {
			if addr == "" {
				continue
			}
			client, err := newClientConn(addr)
			if err != nil {
				return nil, nil, fmt.Errorf("newClient: %w", err)
			}

			rr, err := client.Join(rs.ctx, &pb.JoinRequest{
				AdvertiseAddr: cfg.AdvertiseAddr,
				Voter:         cfg.Voter,
				RequestedAt:   timestamppb.Now(),
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to join raft: %w", err)
			}

			// add node to voters list
			rs.voters[rr.LeaderId] = addr

			// reassign node id based on raft
			rs.nodeID = rr.AssignedId
			rs.leaderID = rr.LeaderId
			rs.term = rr.Term

			rs.state = follower
		}
	}

	return rs, newTransport(rs), nil
}

// Start begin worker thread
func (rs *Server) Start(ctx context.Context) error {
	rs.g.SetLimit(1)
	started := rs.g.TryGo(rs.run)
	if !started {
		return errors.New("worker thread not started")
	}
	return nil
}

func (rs *Server) run() error {
	for {
		log.Printf("raft state %d\n", rs.state)

		select {
		case <-rs.ctx.Done():
			return rs.ctx.Err()
		case <-rs.ticker.C:
			// heartbeat and message timeout
			// node assume follower
			if rs.state == follower {
				rs.state = candidate
			}
		default:
			// fallthrough
		}

		switch rs.state {
		case leader:
			rs.leaderLifecycle()
		case candidate:
			rs.hostElection()
		}

		for rs.commitIndex > rs.lastApplied {
			l := rs.entries[rs.commitIndex-1]
			rs.applyToModule(l)
			rs.lastApplied += 1
		}

		// follower
		// does nothing expect response to requests from leaders and candidates

		time.Sleep(time.Millisecond * 250)
	}
}

func (rs *Server) Shutdown(ctx context.Context) error {
	rs.cancelFunc()
	return rs.g.Wait()
}

func (rs *Server) leaderLifecycle() {

	// send heartbeat to all nodes in cluster
	for nodeID, addr := range rs.voters {
		client, err := newClientConn(addr)
		if err != nil {
			log.Println(err)
			// remove node
			delete(rs.voters, nodeID)
		}

		resp, err := client.AppendEntries(rs.ctx, &pb.AppendEntriesRequest{
			LeaderId: rs.nodeID,

			Term:         rs.term,
			LeaderCommit: rs.commitIndex,
			RequestedAt:  timestamppb.Now(),
		})
		if err != nil {
			log.Println(err)
			delete(rs.voters, nodeID)
		} else if resp.Term > rs.term {
			// node responded with higher term than server
			// return to follower
			rs.state = follower
			continue
		}
	}
}

func (rs *Server) hostElection() {
	// increment term
	rs.term += 1

	var voteCount = 0
	voteCount++ // vote for self

	if len(rs.voters) == 0 {
		// self elected leader
		rs.state = leader
	}

	for nodeID, addr := range rs.voters {
		client, err := newClientConn(addr)
		if err != nil {
			log.Println(err)
			// remove node
			delete(rs.voters, nodeID)
		}

		if resp, err := client.Vote(rs.ctx, &pb.VoteRequest{
			Term: rs.term,
		}); err != nil {
			log.Println(err)
			delete(rs.voters, nodeID)
		} else if resp.Term > rs.term {
			rs.state = follower
		} else if resp.VoteGranted {
			voteCount++
		}
	}

	if voteCount > len(rs.voters)/2 {
		// node won the majority
		rs.state = leader
	}
}

func (rs *Server) addNode(addr string, voter bool) string {
	id := uuid.New().String()
	if voter {
		rs.voters[id] = addr
	} else {
		rs.peers[id] = addr
	}

	return id
}

// Apply
func (rs *Server) apply(cmd []byte) error {

	// extra logic here
	l := Log{
		Term:  rs.term,
		Index: rs.commitIndex,
		Cmd:   cmd,
	}

	if err := rs.appendEntry(l); err != nil {
		return fmt.Errorf("failed to append entries: %w", err)
	}

	// log entry was appended on all nodes
	//
	// entry is now considered commited
	// increment commit index
	rs.entries = append(rs.entries, l)
	rs.commitIndex += 1

	return nil
}

// appendEntry used by leader to append entry to all nodes
func (rs *Server) appendEntry(log Log) error {

	g, ctx := errgroup.WithContext(rs.ctx)
	for nodeID, addr := range rs.voters {
		g.TryGo(func() error {

			clientConn, err := newClientConn(addr)
			if err != nil {
				return fmt.Errorf("failed to create new client: %w", err)
			}

			rr, err := clientConn.AppendEntries(ctx, &pb.AppendEntriesRequest{
				Term:         rs.term,
				LeaderId:     rs.nodeID,
				PrevLogIndex: rs.lastApplied,
				PrevLogTerm:  rs.matchIndex[nodeID],
				LogEntries: []*pb.LogEntry{{
					Index: log.Index,
					Term:  log.Term,
					Cmd:   log.Cmd,
				}},
			})
			if err != nil {
				return fmt.Errorf("failed to append entries: %w", err)
			}

			if rr.Term > rs.term {
				rs.state = follower
				return nil
			}

			return nil
		})
	}

	// need to continue retry logic until failure

	return g.Wait()
}

// send command payload to module
func (rs *Server) applyToModule(log Log) {
	// TODO
	// define custom behavior using errors
	// if errors matching is not sucessful
	// handle all errors gracefully
	_ = rs.md.Apply(log.Cmd)
}
