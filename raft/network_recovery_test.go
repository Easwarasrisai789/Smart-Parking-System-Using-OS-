package raft

import (
	"testing"
)

func TestNetworkLeaderRecoveryAndCatchUp(t *testing.T) {
	// --------------------------------------------------
	// Create three Raft nodes.
	// --------------------------------------------------

	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	// --------------------------------------------------
	// Start Node 0.
	// --------------------------------------------------

	server0, listener0, err := StartNetworkServer(
		node0,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 0: %v", err)
	}

	// --------------------------------------------------
	// Start Node 1.
	// --------------------------------------------------

	server1, listener1, err := StartNetworkServer(
		node1,
		"127.0.0.1:0",
	)
	if err != nil {
		server0.Stop()
		listener0.Close()
		t.Fatalf("failed to start node 1: %v", err)
	}

	defer server1.Stop()
	defer listener1.Close()

	// --------------------------------------------------
	// Start Node 2.
	// --------------------------------------------------

	server2, listener2, err := StartNetworkServer(
		node2,
		"127.0.0.1:0",
	)
	if err != nil {
		server0.Stop()
		listener0.Close()
		t.Fatalf("failed to start node 2: %v", err)
	}

	defer server2.Stop()
	defer listener2.Close()

	// --------------------------------------------------
	// Store network addresses.
	// --------------------------------------------------

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

	// --------------------------------------------------
	// Create network cluster.
	// --------------------------------------------------

	cluster := NewNetworkCluster(nodes, addresses)

	if err := cluster.Connect(); err != nil {
		t.Fatalf("failed to connect cluster: %v", err)
	}

	defer cluster.Close()

	// ==================================================
	// Node 0 starts as leader.
	// ==================================================

	node0.CurrentTerm = 1
	node1.CurrentTerm = 1
	node2.CurrentTerm = 1

	node0.BecomeLeader()

	if node0.State != Leader {
		t.Fatal("node 0 should be leader")
	}

	t.Log("node 0 is the initial leader")

	// ==================================================
	// Replicate first command to Node 1.
	// ==================================================

	firstEntry := LogEntry{
		Term:    1,
		Command: "ENTER|001|AP39AB1234",
	}

	node0.Log = append(node0.Log, firstEntry)

	firstArgs := AppendEntriesArgs{
		Term:         node0.CurrentTerm,
		LeaderID:     0,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			firstEntry,
		},
		LeaderCommit: -1,
	}

	reply, err := cluster.AppendEntries(
		0,
		1,
		firstArgs,
	)
	if err != nil {
		t.Fatalf(
			"replication from node 0 to node 1 failed: %v",
			err,
		)
	}

	if !reply.Success {
		t.Fatal("node 1 rejected first command")
	}

	// ==================================================
	// Replicate first command to Node 2.
	// ==================================================

	reply, err = cluster.AppendEntries(
		0,
		2,
		firstArgs,
	)
	if err != nil {
		t.Fatalf(
			"replication from node 0 to node 2 failed: %v",
			err,
		)
	}

	if !reply.Success {
		t.Fatal("node 2 rejected first command")
	}

	// --------------------------------------------------
	// Verify first command.
	// --------------------------------------------------

	if len(node1.Log) != 1 {
		t.Fatalf(
			"node 1 expected 1 entry, got %d",
			len(node1.Log),
		)
	}

	if len(node2.Log) != 1 {
		t.Fatalf(
			"node 2 expected 1 entry, got %d",
			len(node2.Log),
		)
	}

	t.Log("first command replicated to all nodes")

	// ==================================================
	// Fail Node 0.
	// ==================================================

	server0.Stop()
	listener0.Close()

	t.Log("node 0 failed")

	// ==================================================
	// Node 1 becomes new leader.
	// ==================================================

	node1.CurrentTerm = 2

	node1.StartElection()

	if node1.State != Candidate {
		t.Fatalf(
			"expected node 1 to become candidate, got %v",
			node1.State,
		)
	}

	// --------------------------------------------------
	// Ask Node 2 for its vote.
	// --------------------------------------------------

	voteReply, err := cluster.RequestVote(
		1,
		2,
		RequestVoteArgs{
			Term:         node1.CurrentTerm,
			CandidateID:  1,
			LastLogIndex: len(node1.Log) - 1,
			LastLogTerm:  node1.Log[len(node1.Log)-1].Term,
		},
	)

	if err != nil {
		t.Fatalf(
			"node 1 -> node 2 vote request failed: %v",
			err,
		)
	}

	if !voteReply.VoteGranted {
		t.Fatal("node 2 did not vote for node 1")
	}

	// --------------------------------------------------
	// Node 1 becomes leader.
	// --------------------------------------------------

	node1.BecomeLeader()

	if node1.State != Leader {
		t.Fatal("node 1 should become leader")
	}

	t.Log("node 1 became new leader")

	// ==================================================
	// Add second command while Node 0 is offline.
	// ==================================================

	secondEntry := LogEntry{
		// IMPORTANT:
		// StartElection increments CurrentTerm.
		// Therefore use the actual current term.
		Term:    node1.CurrentTerm,
		Command: "ENTER|002|AP39CD5678",
	}

	node1.Log = append(node1.Log, secondEntry)

	// ==================================================
	// Replicate second command to Node 2.
	// ==================================================

	secondArgs := AppendEntriesArgs{
		// IMPORTANT:
		// Use Node 1's actual current term.
		Term:         node1.CurrentTerm,
		LeaderID:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  1,
		Entries: []LogEntry{
			secondEntry,
		},
		LeaderCommit: 0,
	}

	reply, err = cluster.AppendEntries(
		1,
		2,
		secondArgs,
	)

	if err != nil {
		t.Fatalf(
			"node 1 -> node 2 second replication failed: %v",
			err,
		)
	}

	if !reply.Success {
		t.Fatal("node 2 rejected second command")
	}

	// --------------------------------------------------
	// Verify Node 2 received second command.
	// --------------------------------------------------

	if len(node2.Log) != 2 {
		t.Fatalf(
			"node 2 expected 2 entries, got %d",
			len(node2.Log),
		)
	}

	if node2.Log[1].Command != "ENTER|002|AP39CD5678" {
		t.Fatalf(
			"node 2 second command is incorrect: %s",
			node2.Log[1].Command,
		)
	}

	t.Log("second command replicated while node 0 was offline")

	// ==================================================
	// Recover Node 0.
	// ==================================================

	server0, listener0, err = StartNetworkServer(
		node0,
		"127.0.0.1:0",
	)

	if err != nil {
		t.Fatalf(
			"failed to restart node 0: %v",
			err,
		)
	}

	defer server0.Stop()
	defer listener0.Close()

	t.Log("node 0 network server recovered")

	// ==================================================
	// Create client from Node 1 to recovered Node 0.
	// ==================================================

	recoveryClient, err := NewRaftClient(
		listener0.Addr().String(),
	)

	if err != nil {
		t.Fatalf(
			"failed to connect to recovered node 0: %v",
			err,
		)
	}

	defer recoveryClient.Close()

	// ==================================================
	// Send missing second entry to recovered Node 0.
	// ==================================================

	recoveryArgs := AppendEntriesArgs{
		Term:         node1.CurrentTerm,
		LeaderID:     1,
		PrevLogIndex: 0,
		PrevLogTerm:  1,
		Entries: []LogEntry{
			secondEntry,
		},
		LeaderCommit: 1,
	}

	recoveryReply, err := recoveryClient.AppendEntries(
		recoveryArgs,
	)

	if err != nil {
		t.Fatalf(
			"recovery AppendEntries failed: %v",
			err,
		)
	}

	if !recoveryReply.Success {
		t.Fatal("recovered node rejected catch-up entry")
	}

	// ==================================================
	// Verify Node 0 caught up.
	// ==================================================

	if len(node0.Log) != 2 {
		t.Fatalf(
			"recovered node 0 expected 2 entries, got %d",
			len(node0.Log),
		)
	}

	if node0.Log[0].Command != "ENTER|001|AP39AB1234" {
		t.Fatalf(
			"node 0 first command is incorrect: %s",
			node0.Log[0].Command,
		)
	}

	if node0.Log[1].Command != "ENTER|002|AP39CD5678" {
		t.Fatalf(
			"node 0 second command is incorrect: %s",
			node0.Log[1].Command,
		)
	}

	t.Log("recovered node 0 successfully caught up")
}
