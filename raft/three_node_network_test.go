package raft

import (
	"testing"
)

func TestThreeNodeNetworkCommunication(t *testing.T) {
	// Create three independent Raft nodes.
	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	// Start Node 0 network server.
	server0, listener0, err := StartNetworkServer(
		node0,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 0: %v", err)
	}

	defer server0.Stop()
	defer listener0.Close()

	// Start Node 1 network server.
	server1, listener1, err := StartNetworkServer(
		node1,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 1: %v", err)
	}

	defer server1.Stop()
	defer listener1.Close()

	// Start Node 2 network server.
	server2, listener2, err := StartNetworkServer(
		node2,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 2: %v", err)
	}

	defer server2.Stop()
	defer listener2.Close()

	// Node 0 -> Node 1
	client0To1, err := NewRaftClient(listener1.Addr().String())
	if err != nil {
		t.Fatalf("node 0 -> node 1 connection failed: %v", err)
	}
	defer client0To1.Close()

	// Node 0 -> Node 2
	client0To2, err := NewRaftClient(listener2.Addr().String())
	if err != nil {
		t.Fatalf("node 0 -> node 2 connection failed: %v", err)
	}
	defer client0To2.Close()

	// Node 1 -> Node 0
	client1To0, err := NewRaftClient(listener0.Addr().String())
	if err != nil {
		t.Fatalf("node 1 -> node 0 connection failed: %v", err)
	}
	defer client1To0.Close()

	// Node 1 -> Node 2
	client1To2, err := NewRaftClient(listener2.Addr().String())
	if err != nil {
		t.Fatalf("node 1 -> node 2 connection failed: %v", err)
	}
	defer client1To2.Close()

	// Node 2 -> Node 0
	client2To0, err := NewRaftClient(listener0.Addr().String())
	if err != nil {
		t.Fatalf("node 2 -> node 0 connection failed: %v", err)
	}
	defer client2To0.Close()

	// Node 2 -> Node 1
	client2To1, err := NewRaftClient(listener1.Addr().String())
	if err != nil {
		t.Fatalf("node 2 -> node 1 connection failed: %v", err)
	}
	defer client2To1.Close()

	// --------------------------------------------------
	// Test RequestVote
	// --------------------------------------------------

	voteArgs := RequestVoteArgs{
		Term:         1,
		CandidateID:  0,
		LastLogIndex: 0,
		LastLogTerm:  0,
	}

	// Node 0 -> Node 1
	reply, err := client0To1.RequestVote(voteArgs)
	if err != nil {
		t.Fatalf(
			"node 0 -> node 1 RequestVote failed: %v",
			err,
		)
	}

	if reply.Term != 1 {
		t.Fatalf(
			"node 1 returned term %d, expected 1",
			reply.Term,
		)
	}

	// Node 0 -> Node 2
	reply, err = client0To2.RequestVote(voteArgs)
	if err != nil {
		t.Fatalf(
			"node 0 -> node 2 RequestVote failed: %v",
			err,
		)
	}

	if reply.Term != 1 {
		t.Fatalf(
			"node 2 returned term %d, expected 1",
			reply.Term,
		)
	}

	// --------------------------------------------------
	// Test AppendEntries
	// --------------------------------------------------

	appendArgs := AppendEntriesArgs{
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

	// Node 0 -> Node 1
	appendReply, err := client0To1.AppendEntries(appendArgs)
	if err != nil {
		t.Fatalf(
			"node 0 -> node 1 AppendEntries failed: %v",
			err,
		)
	}

	if !appendReply.Success {
		t.Fatal("node 1 rejected AppendEntries")
	}

	// Node 0 -> Node 2
	appendReply, err = client0To2.AppendEntries(appendArgs)
	if err != nil {
		t.Fatalf(
			"node 0 -> node 2 AppendEntries failed: %v",
			err,
		)
	}

	if !appendReply.Success {
		t.Fatal("node 2 rejected AppendEntries")
	}

	// --------------------------------------------------
	// Test reverse communication
	// --------------------------------------------------

	reverseVoteArgs := RequestVoteArgs{
		Term:         2,
		CandidateID:  2,
		LastLogIndex: 1,
		LastLogTerm:  1,
	}

	// Node 2 -> Node 0
	reply, err = client2To0.RequestVote(reverseVoteArgs)
	if err != nil {
		t.Fatalf(
			"node 2 -> node 0 RequestVote failed: %v",
			err,
		)
	}

	if reply.Term != 2 {
		t.Fatalf(
			"node 0 returned term %d, expected 2",
			reply.Term,
		)
	}

	// Node 2 -> Node 1
	reply, err = client2To1.RequestVote(reverseVoteArgs)
	if err != nil {
		t.Fatalf(
			"node 2 -> node 1 RequestVote failed: %v",
			err,
		)
	}

	if reply.Term != 2 {
		t.Fatalf(
			"node 1 returned term %d, expected 2",
			reply.Term,
		)
	}

	// --------------------------------------------------
	// Node 1 communication
	// --------------------------------------------------

	_, err = client1To0.RequestVote(voteArgs)
	if err != nil {
		t.Fatalf(
			"node 1 -> node 0 RequestVote failed: %v",
			err,
		)
	}

	_, err = client1To2.RequestVote(voteArgs)
	if err != nil {
		t.Fatalf(
			"node 1 -> node 2 RequestVote failed: %v",
			err,
		)
	}
}
