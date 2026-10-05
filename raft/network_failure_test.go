package raft

import (
	"testing"
)

func TestNetworkLeaderFailureAndNewElection(t *testing.T) {
	// Create three Raft nodes.
	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	// Start Node 0.
	server0, listener0, err := StartNetworkServer(
		node0,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 0: %v", err)
	}

	// Start Node 1.
	server1, listener1, err := StartNetworkServer(
		node1,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 1: %v", err)
	}

	defer server1.Stop()
	defer listener1.Close()

	// Start Node 2.
	server2, listener2, err := StartNetworkServer(
		node2,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 2: %v", err)
	}

	defer server2.Stop()
	defer listener2.Close()

	addresses := map[int]string{
		0: listener0.Addr().String(),
		1: listener1.Addr().String(),
		2: listener2.Addr().String(),
	}

	nodes := map[int]*Raft{
		0: node0,
		1: node1,
		2: node2,
	}

	cluster := NewNetworkCluster(nodes, addresses)

	if err := cluster.Connect(); err != nil {
		t.Fatalf("failed to connect cluster: %v", err)
	}

	defer cluster.Close()

	// Node 0 is initially the leader.
	node0.BecomeLeader()

	if node0.State != Leader {
		t.Fatal("node 0 should initially be leader")
	}

	node0.CurrentTerm = 1
	node1.CurrentTerm = 1
	node2.CurrentTerm = 1

	t.Log("node 0 is the initial leader")

	// --------------------------------------------------
	// Simulate Node 0 failure.
	// --------------------------------------------------

	server0.Stop()
	listener0.Close()

	t.Log("node 0 has failed")

	// --------------------------------------------------
	// Node 1 starts a new election.
	// --------------------------------------------------

	node1.StartElection()

	if node1.State != Candidate {
		t.Fatalf(
			"expected node 1 to become candidate, got state %v",
			node1.State,
		)
	}

	// Node 1 requests a vote from Node 2.
	reply, err := cluster.RequestVote(
		1,
		2,
		RequestVoteArgs{
			Term:         node1.CurrentTerm,
			CandidateID:  1,
			LastLogIndex: len(node1.Log) - 1,
			LastLogTerm:  0,
		},
	)

	if err != nil {
		t.Fatalf(
			"node 1 -> node 2 RequestVote failed: %v",
			err,
		)
	}

	votes := 1

	if reply.VoteGranted {
		votes++
	}

	// A 3-node cluster needs 2 votes for majority.
	if votes < 2 {
		t.Fatalf(
			"node 1 did not receive majority: %d votes",
			votes,
		)
	}

	// Node 1 becomes the new leader.
	node1.BecomeLeader()

	if node1.State != Leader {
		t.Fatal("node 1 should become the new leader")
	}

	t.Logf(
		"new election successful: node 1 became leader with %d votes",
		votes,
	)

	// --------------------------------------------------
	// Verify Node 1 can still communicate with Node 2.
	// --------------------------------------------------

	appendArgs := AppendEntriesArgs{
		Term:         node1.CurrentTerm,
		LeaderID:     node1.ID,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{
				Term:    node1.CurrentTerm,
				Command: "ENTER|002|AP39CD5678",
			},
		},
		LeaderCommit: -1,
	}

	appendReply, err := cluster.AppendEntries(
		1,
		2,
		appendArgs,
	)

	if err != nil {
		t.Fatalf(
			"new leader failed to communicate with node 2: %v",
			err,
		)
	}

	if !appendReply.Success {
		t.Fatal("node 2 rejected AppendEntries from new leader")
	}

	t.Log("new leader successfully communicated with surviving follower")
}
