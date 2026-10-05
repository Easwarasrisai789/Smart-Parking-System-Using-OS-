package raft

import (
	"testing"
)

func TestNetworkLogReplication(t *testing.T) {
	// Create three independent Raft nodes.
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

	defer server0.Stop()
	defer listener0.Close()

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

	// Create network cluster.
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

	// Node 0 becomes leader.
	node0.BecomeLeader()

	if node0.State != Leader {
		t.Fatal("node 0 should be leader")
	}

	// Create a parking command.
	entry := LogEntry{
		Term:    node0.CurrentTerm,
		Command: "ENTER|001|AP39AB1234",
	}

	// Add the entry to the leader's log.
	node0.Log = append(node0.Log, entry)

	// Prepare AppendEntries request.
	args := AppendEntriesArgs{
		Term:         node0.CurrentTerm,
		LeaderID:     node0.ID,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			entry,
		},
		LeaderCommit: 0,
	}

	// Replicate to Node 1.
	reply1, err := cluster.AppendEntries(
		0,
		1,
		args,
	)

	if err != nil {
		t.Fatalf(
			"node 0 -> node 1 replication failed: %v",
			err,
		)
	}

	if !reply1.Success {
		t.Fatal("node 1 rejected replicated log entry")
	}

	// Replicate to Node 2.
	reply2, err := cluster.AppendEntries(
		0,
		2,
		args,
	)

	if err != nil {
		t.Fatalf(
			"node 0 -> node 2 replication failed: %v",
			err,
		)
	}

	if !reply2.Success {
		t.Fatal("node 2 rejected replicated log entry")
	}

	// Verify Node 1 received the entry.
	if len(node1.Log) != 1 {
		t.Fatalf(
			"node 1 expected 1 log entry, got %d",
			len(node1.Log),
		)
	}

	if node1.Log[0].Command != "ENTER|001|AP39AB1234" {
		t.Fatalf(
			"node 1 has wrong command: %s",
			node1.Log[0].Command,
		)
	}

	// Verify Node 2 received the entry.
	if len(node2.Log) != 1 {
		t.Fatalf(
			"node 2 expected 1 log entry, got %d",
			len(node2.Log),
		)
	}

	if node2.Log[0].Command != "ENTER|001|AP39AB1234" {
		t.Fatalf(
			"node 2 has wrong command: %s",
			node2.Log[0].Command,
		)
	}

	t.Log("network log replication successful")
}
