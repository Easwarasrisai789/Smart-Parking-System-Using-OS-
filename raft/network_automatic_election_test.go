package raft

import (
	"testing"
	"time"
)

func TestAutomaticNetworkElection(t *testing.T) {
	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	server0, listener0, err := StartNetworkServer(node0, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server1, listener1, err := StartNetworkServer(node1, "127.0.0.1:0")
	if err != nil {
		server0.Stop()
		listener0.Close()
		t.Fatal(err)
	}

	server2, listener2, err := StartNetworkServer(node2, "127.0.0.1:0")
	if err != nil {
		server0.Stop()
		listener0.Close()
		server1.Stop()
		listener1.Close()
		t.Fatal(err)
	}

	defer server0.Stop()
	defer server1.Stop()
	defer server2.Stop()

	defer listener0.Close()
	defer listener1.Close()
	defer listener2.Close()

	client01, err := NewRaftClient(listener1.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	client02, err := NewRaftClient(listener2.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	defer client01.Close()
	defer client02.Close()

	clients0 := map[int]*RaftClient{
		1: client01,
		2: client02,
	}

	runtime := NewNetworkRuntime(node0, clients0)
	runtime.Start()
	defer runtime.Stop()

	// Force node 0 to have an expired election timer.
	node0.ResetElectionTimer()
	node0.mu.Lock()
	node0.lastHeartbeat = time.Now().Add(-2 * time.Second)
	node0.mu.Unlock()

	// Wait for the automatic election.
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if node0.GetState() == Leader {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if node0.GetState() != Leader {
		t.Fatalf(
			"expected node 0 to become leader, got %v",
			node0.GetState(),
		)
	}

	t.Log("automatic network election successful")
}
