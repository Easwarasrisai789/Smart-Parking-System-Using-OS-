package raft

import (
	"testing"
	"time"
)

func TestAutomaticNetworkHeartbeats(t *testing.T) {
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

		server1.Stop()
		listener1.Close()

		t.Fatalf("failed to start node 2: %v", err)
	}

	defer server0.Stop()
	defer server1.Stop()
	defer server2.Stop()

	defer listener0.Close()
	defer listener1.Close()
	defer listener2.Close()

	// --------------------------------------------------
	// Create clients.
	// --------------------------------------------------

	client01, err := NewRaftClient(listener1.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 0 -> node 1: %v", err)
	}

	client02, err := NewRaftClient(listener2.Addr().String())
	if err != nil {
		client01.Close()

		t.Fatalf("failed to connect node 0 -> node 2: %v", err)
	}

	defer client01.Close()
	defer client02.Close()

	clients := map[int]*RaftClient{
		1: client01,
		2: client02,
	}

	// --------------------------------------------------
	// Make Node 0 the leader.
	// --------------------------------------------------

	node0.CurrentTerm = 1
	node1.CurrentTerm = 1
	node2.CurrentTerm = 1

	node0.BecomeLeader()

	if node0.GetState() != Leader {
		t.Fatalf("node 0 should be leader")
	}

	t.Log("node 0 is leader")

	// --------------------------------------------------
	// Start automatic runtime.
	// --------------------------------------------------

	runtime := NewNetworkRuntime(
		node0,
		clients,
	)

	runtime.Start()
	defer runtime.Stop()

	// --------------------------------------------------
	// Give heartbeat loop time to run.
	// --------------------------------------------------

	time.Sleep(500 * time.Millisecond)

	// --------------------------------------------------
	// Verify followers remain followers.
	// --------------------------------------------------

	if node1.GetState() != Follower {
		t.Fatalf(
			"node 1 should remain follower, got %v",
			node1.GetState(),
		)
	}

	if node2.GetState() != Follower {
		t.Fatalf(
			"node 2 should remain follower, got %v",
			node2.GetState(),
		)
	}

	// --------------------------------------------------
	// Verify leader is still leader.
	// --------------------------------------------------

	if node0.GetState() != Leader {
		t.Fatalf(
			"node 0 should remain leader, got %v",
			node0.GetState(),
		)
	}

	t.Log("automatic heartbeats successfully maintained the leader")
}
