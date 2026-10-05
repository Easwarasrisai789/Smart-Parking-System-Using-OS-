package raft

import (
	"fmt"
	"testing"
	"time"
)

func TestAutomaticLeaderRecoveryAndCatchUp(t *testing.T) {
	// ---------------------------------------------------------
	// Create three Raft nodes
	// ---------------------------------------------------------

	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	// ---------------------------------------------------------
	// Set initial state BEFORE starting runtimes.
	// Node 0 is the initial leader.
	// ---------------------------------------------------------

	node0.mu.Lock()
	node0.State = Leader
	node0.CurrentTerm = 1
	node0.VotedFor = 0
	node0.lastHeartbeat = time.Now()
	node0.mu.Unlock()

	node1.mu.Lock()
	node1.State = Follower
	node1.CurrentTerm = 1
	node1.VotedFor = -1
	node1.lastHeartbeat = time.Now()
	node1.mu.Unlock()

	node2.mu.Lock()
	node2.State = Follower
	node2.CurrentTerm = 1
	node2.VotedFor = -1
	node2.lastHeartbeat = time.Now()
	node2.mu.Unlock()

	t.Log("node 0 is the initial leader")

	// ---------------------------------------------------------
	// Start gRPC servers
	// ---------------------------------------------------------

	server0, listener0, err := StartNetworkServer(node0, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start node 0 server: %v", err)
	}
	defer server0.Stop()
	defer listener0.Close()

	server1, listener1, err := StartNetworkServer(node1, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start node 1 server: %v", err)
	}
	defer server1.Stop()
	defer listener1.Close()

	server2, listener2, err := StartNetworkServer(node2, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start node 2 server: %v", err)
	}
	defer server2.Stop()
	defer listener2.Close()

	// ---------------------------------------------------------
	// Create clients
	// ---------------------------------------------------------

	client01, err := NewRaftClient(listener1.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 0 -> node 1: %v", err)
	}
	defer client01.Close()

	client02, err := NewRaftClient(listener2.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 0 -> node 2: %v", err)
	}
	defer client02.Close()

	client10, err := NewRaftClient(listener0.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 1 -> node 0: %v", err)
	}
	defer client10.Close()

	client12, err := NewRaftClient(listener2.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 1 -> node 2: %v", err)
	}
	defer client12.Close()

	client20, err := NewRaftClient(listener0.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 2 -> node 0: %v", err)
	}
	defer client20.Close()

	client21, err := NewRaftClient(listener1.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect node 2 -> node 1: %v", err)
	}
	defer client21.Close()

	// ---------------------------------------------------------
	// Create client maps
	// ---------------------------------------------------------

	clients0 := map[int]*RaftClient{
		1: client01,
		2: client02,
	}

	clients1 := map[int]*RaftClient{
		0: client10,
		2: client12,
	}

	clients2 := map[int]*RaftClient{
		0: client20,
		1: client21,
	}

	// ---------------------------------------------------------
	// Create automatic runtimes
	// IMPORTANT:
	// Node 0 also needs a runtime because it is the leader
	// and must automatically replicate its log.
	// ---------------------------------------------------------

	runtime0 := NewNetworkRuntime(node0, clients0)
	runtime1 := NewNetworkRuntime(node1, clients1)
	runtime2 := NewNetworkRuntime(node2, clients2)

	runtime0.Start()
	runtime1.Start()
	time.Sleep(400 * time.Millisecond)
	runtime2.Start()

	defer runtime0.Stop()
	defer runtime1.Stop()
	defer runtime2.Stop()

	// ---------------------------------------------------------
	// Give the runtimes a moment to start
	// ---------------------------------------------------------

	time.Sleep(200 * time.Millisecond)

	// ---------------------------------------------------------
	// Add first command to leader
	// ---------------------------------------------------------

	node0.mu.Lock()

	node0.Log = append(node0.Log, LogEntry{
		Term:    node0.CurrentTerm,
		Command: "ENTER|slot-001|vehicle-001",
	})

	node0.mu.Unlock()

	t.Log("first command added to node 0")

	// ---------------------------------------------------------
	// Wait for automatic replication
	// ---------------------------------------------------------

	time.Sleep(800 * time.Millisecond)

	// ---------------------------------------------------------
	// Verify node 1 received first command
	// ---------------------------------------------------------

	node1.mu.Lock()
	node1LogLength := len(node1.Log)
	node1.mu.Unlock()

	if node1LogLength != 1 {
		t.Fatalf(
			"node 1 expected 1 log entry, got %d",
			node1LogLength,
		)
	}

	// ---------------------------------------------------------
	// Verify node 2 received first command
	// ---------------------------------------------------------

	node2.mu.Lock()
	node2LogLength := len(node2.Log)
	node2.mu.Unlock()

	if node2LogLength != 1 {
		t.Fatalf(
			"node 2 expected 1 log entry, got %d",
			node2LogLength,
		)
	}

	t.Log("first command replicated to nodes 1 and 2")

	// ---------------------------------------------------------
	// Stop node 0
	// Simulate leader failure.
	// ---------------------------------------------------------

	runtime0.Stop()
	server0.Stop()

	t.Log("node 0 has failed")

	// ---------------------------------------------------------
	// Wait for one surviving node to win an election.
	// ---------------------------------------------------------

	deadline := time.Now().Add(5 * time.Second)
	var newLeader *Raft
	var newLeaderID int

	for time.Now().Before(deadline) {
		node1Leader := node1.GetState() == Leader
		node2Leader := node2.GetState() == Leader

		if node1Leader != node2Leader {
			if node1Leader {
				newLeader = node1
				newLeaderID = 1
			} else {
				newLeader = node2
				newLeaderID = 2
			}
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if newLeader == nil {
		t.Fatalf(
			"neither node 1 nor node 2 became the new leader",
		)
	}

	t.Logf(
		"node %d automatically became the new leader",
		newLeaderID,
	)

	// ---------------------------------------------------------
	// Add second command to new leader
	// ---------------------------------------------------------

	newLeader.mu.Lock()

	newLeader.Log = append(
		newLeader.Log,
		LogEntry{
			Term:    newLeader.CurrentTerm,
			Command: "EXIT|slot-001|vehicle-001",
		},
	)

	newLeader.mu.Unlock()

	t.Logf(
		"second command added to node %d",
		newLeaderID,
	)

	// ---------------------------------------------------------
	// Verify surviving follower has both commands
	// ---------------------------------------------------------

	var follower *Raft

	if newLeaderID == 1 {
		follower = node2
	} else {
		follower = node1
	}

	replicationDeadline := time.Now().Add(5 * time.Second)
	followerLogLength := 0
	for time.Now().Before(replicationDeadline) {
		follower.mu.Lock()
		followerLogLength = len(follower.Log)
		follower.mu.Unlock()

		if followerLogLength >= 2 {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if followerLogLength != 2 {
		t.Fatalf(
			"surviving follower expected 2 log entries, got %d",
			followerLogLength,
		)
	}

	t.Log(
		"second command replicated to surviving follower",
	)

	// ---------------------------------------------------------
	// Restart node 0 network server
	// ---------------------------------------------------------

	server0, listener0, err = StartNetworkServer(
		node0,
		"127.0.0.1:0",
	)

	if err != nil {
		t.Fatalf(
			"failed to restart node 0 network server: %v",
			err,
		)
	}

	defer server0.Stop()
	defer listener0.Close()

	t.Log("node 0 network server recovered")

	// ---------------------------------------------------------
	// Create recovery client
	// ---------------------------------------------------------

	recoveryClient, err := NewRaftClient(
		listener0.Addr().String(),
	)

	if err != nil {
		t.Fatalf(
			"failed to connect recovered node 0: %v",
			err,
		)
	}

	defer recoveryClient.Close()

	// ---------------------------------------------------------
	// Take a snapshot of the new leader's log
	// ---------------------------------------------------------

	newLeader.mu.Lock()

	recoveredLog := make([]LogEntry, len(newLeader.Log))
	copy(recoveredLog, newLeader.Log)

	recoveredTerm := newLeader.CurrentTerm

	newLeader.mu.Unlock()

	if len(recoveredLog) != 2 {
		t.Fatalf(
			"new leader expected 2 log entries, got %d",
			len(recoveredLog),
		)
	}

	// ---------------------------------------------------------
	// Send complete log to recovered node 0
	// ---------------------------------------------------------

	reply, err := recoveryClient.AppendEntries(
		AppendEntriesArgs{
			Term:     recoveredTerm,
			LeaderID: newLeaderID,

			PrevLogIndex: -1,
			PrevLogTerm:  0,

			Entries: recoveredLog,

			LeaderCommit: -1,
		},
	)

	if err != nil {
		t.Fatalf(
			"failed to catch up recovered node 0: %v",
			err,
		)
	}

	if !reply.Success {
		t.Fatalf(
			"recovered node 0 rejected catch-up AppendEntries",
		)
	}

	// ---------------------------------------------------------
	// Verify recovered node 0 has both commands
	// ---------------------------------------------------------

	node0.mu.Lock()
	node0LogLength := len(node0.Log)
	node0.mu.Unlock()

	if node0LogLength != 2 {
		t.Fatalf(
			"recovered node 0 expected 2 log entries, got %d",
			node0LogLength,
		)
	}

	// ---------------------------------------------------------
	// Verify recovered commands
	// ---------------------------------------------------------

	node0.mu.Lock()

	firstCommand := node0.Log[0].Command
	secondCommand := node0.Log[1].Command

	node0.mu.Unlock()

	if firstCommand != "ENTER|slot-001|vehicle-001" {
		t.Fatalf(
			"unexpected first command on recovered node: %s",
			firstCommand,
		)
	}

	if secondCommand != "EXIT|slot-001|vehicle-001" {
		t.Fatalf(
			"unexpected second command on recovered node: %s",
			secondCommand,
		)
	}

	// ---------------------------------------------------------
	// Final success message
	// ---------------------------------------------------------

	t.Log("recovered node 0 successfully caught up")

	t.Log(
		fmt.Sprintf(
			"final logs: node0=%d node1=%d node2=%d",
			len(node0.Log),
			len(node1.Log),
			len(node2.Log),
		),
	)
}
