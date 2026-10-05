package raft

import (
	"context"
	"testing"
	"time"

	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRaftNetworkServer(t *testing.T) {
	node := NewRaft(0, []int{1, 2})

	server, listener, err := StartNetworkServer(node, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start network server: %v", err)
	}

	defer server.Stop()
	defer listener.Close()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create gRPC connection: %v", err)
	}
	defer conn.Close()

	client := pb.NewRaftServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	response, err := client.RequestVote(
		ctx,
		&pb.RequestVoteRequest{
			Term:         1,
			CandidateId:  1,
			LastLogIndex: 0,
			LastLogTerm:  0,
		},
	)

	if err != nil {
		t.Fatalf("RequestVote RPC failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected a response, got nil")
	}
}
