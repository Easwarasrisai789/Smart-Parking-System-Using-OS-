package raft

import (
	"testing"
	"time"
)

func TestAutomaticLeaderFailureAndElection(t *testing.T) {
	// --------------------------------------------------
	// Create three Raft nodes.
	// --------------------------------------------------

	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	// --------------------------------------------------
	// Set initial Raft state BEFORE starting runtimes.
	// --------------------------------------------------

	node0.mu.Lock()
	node0.CurrentTerm = 1
	node0.State = Leader
	node0.VotedFor = -1
	node0.mu.Unlock()

	node1.mu.Lock()
	node1.CurrentTerm = 1
	node1.State = Follower
	node1.VotedFor = -1
	node1.mu.Unlock()

	node2.mu.Lock()
	node2.CurrentTerm = 1
	node2.State = Follower
	node2.VotedFor = -1
	node2.mu.Unlock()

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

	defer server1.Stop()
	defer server2.Stop()

	defer listener1.Close()
	defer listener2.Close()

	// --------------------------------------------------
	// Create Node 1 clients.
	// --------------------------------------------------

	client10, err := NewRaftClient(
		listener0.Addr().String(),
	)
	if err != nil {
		t.Fatalf(
			"node 1 -> node 0 connection failed: %v",
			err,
		)
	}

	client12, err := NewRaftClient(
		listener2.Addr().String(),
	)
	if err != nil {
		client10.Close()

		t.Fatalf(
			"node 1 -> node 2 connection failed: %v",
			err,
		)
	}

	defer client10.Close()
	defer client12.Close()

	// --------------------------------------------------
	// Create Node 2 clients.
	// --------------------------------------------------

	client20, err := NewRaftClient(
		listener0.Addr().String(),
	)
	if err != nil {
		t.Fatalf(
			"node 2 -> node 0 connection failed: %v",
			err,
		)
	}

	client21, err := NewRaftClient(
		listener1.Addr().String(),
	)
	if err != nil {
		client20.Close()

		t.Fatalf(
			"node 2 -> node 1 connection failed: %v",
			err,
		)
	}

	defer client20.Close()
	defer client21.Close()

	// --------------------------------------------------
	// Create automatic runtimes.
	// --------------------------------------------------

	runtime1 := NewNetworkRuntime(
		node1,
		map[int]*RaftClient{
			0: client10,
			2: client12,
		},
	)

	runtime2 := NewNetworkRuntime(
		node2,
		map[int]*RaftClient{
			0: client20,
			1: client21,
		},
	)

	// --------------------------------------------------
	// Verify initial state BEFORE runtimes start.
	// --------------------------------------------------

	if node0.GetState() != Leader {
		t.Fatal("node 0 should be the initial leader")
	}

	if node1.GetState() != Follower {
		t.Fatal("node 1 should initially be a follower")
	}

	if node2.GetState() != Follower {
		t.Fatal("node 2 should initially be a follower")
	}

	t.Log("node 0 is the initial leader")

	// --------------------------------------------------
	// Start automatic runtimes.
	// --------------------------------------------------

	runtime1.Start()
	time.Sleep(400 * time.Millisecond)
	runtime2.Start()

	defer runtime1.Stop()
	defer runtime2.Stop()

	// --------------------------------------------------
	// Give cluster time to communicate.
	// --------------------------------------------------

	time.Sleep(300 * time.Millisecond)

	// --------------------------------------------------
	// Fail Node 0.
	// --------------------------------------------------

	server0.Stop()
	listener0.Close()

	t.Log("node 0 has failed")

	// --------------------------------------------------
	// Wait for a surviving node to win an election.
	// --------------------------------------------------

	deadline := time.Now().Add(5 * time.Second)
	var node1State, node2State State
	node1Leader, node2Leader := false, false

	for time.Now().Before(deadline) {
		node1State = node1.GetState()
		node2State = node2.GetState()
		node1Leader = node1State == Leader
		node2Leader = node2State == Leader

		if node1Leader != node2Leader {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	// --------------------------------------------------
	// At least one surviving node must become leader.
	// --------------------------------------------------

	if !node1Leader && !node2Leader {
		t.Fatalf(
			"no new leader elected: node1=%v node2=%v",
			node1State,
			node2State,
		)
	}

	// --------------------------------------------------
	// Only one surviving node should be leader.
	// --------------------------------------------------

	if node1Leader && node2Leader {
		t.Fatalf(
			"both surviving nodes became leaders: node1=%v node2=%v",
			node1State,
			node2State,
		)
	}

	// --------------------------------------------------
	// Report new leader.
	// --------------------------------------------------

	if node1Leader {
		t.Log("node 1 automatically became the new leader")
	} else {
		t.Log("node 2 automatically became the new leader")
	}
}
