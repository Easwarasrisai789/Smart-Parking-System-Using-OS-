package raft

import (
	"context"
	"fmt"
	"net"
	"sync"

	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
)

type NetworkServer struct {
	pb.UnimplementedRaftServiceServer

	mu      sync.Mutex
	node    *Raft
	metrics *Metrics
}

func NewNetworkServer(node *Raft) *NetworkServer {
	return NewNetworkServerWithMetrics(node, nil)
}

func NewNetworkServerWithMetrics(
	node *Raft,
	metrics *Metrics,
) *NetworkServer {
	return &NetworkServer{
		node:    node,
		metrics: metrics,
	}
}

func (s *NetworkServer) RequestVote(
	ctx context.Context,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	finish := s.metrics.observeRPC("RequestVote")
	defer finish()

	args := RequestVoteArgs{
		Term:         int(req.Term),
		CandidateID:  int(req.CandidateId),
		LastLogIndex: int(req.LastLogIndex),
		LastLogTerm:  int(req.LastLogTerm),
	}

	var reply RequestVoteReply

	s.mu.Lock()
	node := s.node
	s.mu.Unlock()

	if node == nil {
		return nil, fmt.Errorf("raft node is not configured")
	}

	node.RequestVote(&args, &reply)

	return &pb.RequestVoteResponse{
		Term:        int32(reply.Term),
		VoteGranted: reply.VoteGranted,
	}, nil
}

func (s *NetworkServer) AppendEntries(
	ctx context.Context,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {
	finish := s.metrics.observeRPC("AppendEntries")
	defer finish()

	entries := make([]LogEntry, len(req.Entries))

	for i, entry := range req.Entries {
		entries[i] = LogEntry{
			Term:    int(entry.Term),
			Command: entry.Command,
		}
	}

	args := AppendEntriesArgs{
		Term:         int(req.Term),
		LeaderID:     int(req.LeaderId),
		PrevLogIndex: int(req.PrevLogIndex),
		PrevLogTerm:  int(req.PrevLogTerm),
		Entries:      entries,
		LeaderCommit: int(req.LeaderCommit),
	}

	var reply AppendEntriesReply

	s.mu.Lock()
	node := s.node
	s.mu.Unlock()

	if node == nil {
		return nil, fmt.Errorf("raft node is not configured")
	}

	node.AppendEntries(&args, &reply)

	return &pb.AppendEntriesResponse{
		Term:    int32(reply.Term),
		Success: reply.Success,
	}, nil
}

func StartNetworkServer(
	node *Raft,
	address string,
) (*grpc.Server, net.Listener, error) {
	return StartNetworkServerWithMetrics(node, address, nil)
}

func StartNetworkServerWithMetrics(
	node *Raft,
	address string,
	metrics *Metrics,
) (*grpc.Server, net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	server := grpc.NewServer()

	raftServer := NewNetworkServerWithMetrics(node, metrics)

	pb.RegisterRaftServiceServer(server, raftServer)

	go func() {
		if err := server.Serve(listener); err != nil {
			fmt.Printf("Raft gRPC server stopped: %v\n", err)
		}
	}()

	return server, listener, nil
}
