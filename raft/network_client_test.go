package raft

import (
	"testing"
)

func TestRaftNodeToNodeRequestVote(t *testing.T) {
	serverNode := NewRaft(1, []int{0, 2})

	server, listener, err := StartNetworkServer(
		serverNode,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	defer server.Stop()
	defer listener.Close()

	client, err := NewRaftClient(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to create Raft client: %v", err)
	}

	defer client.Close()

	args := RequestVoteArgs{
		Term:         1,
		CandidateID:  0,
		LastLogIndex: 0,
		LastLogTerm:  0,
	}

	reply, err := client.RequestVote(args)
	if err != nil {
		t.Fatalf("RequestVote RPC failed: %v", err)
	}

	if reply.Term != 1 {
		t.Fatalf(
			"expected term 1, got %d",
			reply.Term,
		)
	}
}

func TestRaftNodeToNodeAppendEntries(t *testing.T) {
	serverNode := NewRaft(1, []int{0, 2})

	server, listener, err := StartNetworkServer(
		serverNode,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	defer server.Stop()
	defer listener.Close()

	client, err := NewRaftClient(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to create Raft client: %v", err)
	}

	defer client.Close()

	args := AppendEntriesArgs{
		Term:         1,
		LeaderID:     0,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{
				Term:    1,
				Command: "ENTER|001|AP39AB1234",
			},
		},
		LeaderCommit: -1,
	}

	reply, err := client.AppendEntries(args)
	if err != nil {
		t.Fatalf("AppendEntries RPC failed: %v", err)
	}

	if !reply.Success {
		t.Fatal("expected AppendEntries to succeed")
	}
}
