package raft

import (
	"testing"
)

func TestNetworkRaftElection(t *testing.T) {
	node0 := NewRaft(0, []int{1, 2})
	node1 := NewRaft(1, []int{0, 2})
	node2 := NewRaft(2, []int{0, 1})

	server0, listener0, err := StartNetworkServer(
		node0,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 0: %v", err)
	}

	defer server0.Stop()
	defer listener0.Close()

	server1, listener1, err := StartNetworkServer(
		node1,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("failed to start node 1: %v", err)
	}

	defer server1.Stop()
	defer listener1.Close()

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

	// Node 0 starts an election.
	node0.StartElection()

	term := node0.CurrentTerm

	if term != 1 {
		t.Fatalf(
			"expected node 0 term 1, got %d",
			term,
		)
	}

	votes := 1

	for _, peerID := range []int{1, 2} {
		reply, err := cluster.RequestVote(
			0,
			peerID,
			RequestVoteArgs{
				Term:         node0.CurrentTerm,
				CandidateID:  0,
				LastLogIndex: len(node0.Log) - 1,
				LastLogTerm:  0,
			},
		)

		if err != nil {
			t.Fatalf(
				"node 0 -> node %d RequestVote failed: %v",
				peerID,
				err,
			)
		}

		if reply.VoteGranted {
			votes++
		}
	}

	if votes < 2 {
		t.Fatalf(
			"expected majority of 2 votes, got %d",
			votes,
		)
	}

	node0.BecomeLeader()

	if node0.State != Leader {
		t.Fatal("node 0 should become leader")
	}

	t.Logf(
		"network election successful: node 0 became leader with %d votes",
		votes,
	)
}
