package raft

import (
	"testing"

	parkinglogic "smart-parking-kv/parking"
)

func TestThreeNodeElection(t *testing.T) {
	cluster := NewCluster(3)

	votes := cluster.RequestVotes(0)

	if votes != 3 {
		t.Fatalf(
			"expected 3 votes, got %d",
			votes,
		)
	}

	leader := cluster.Node(0)

	if leader.State != Leader {
		t.Fatal("Node 0 should become leader")
	}

	if leader.CurrentTerm != 1 {
		t.Fatalf(
			"expected term 1, got %d",
			leader.CurrentTerm,
		)
	}

	if cluster.Node(1).VotedFor != 0 {
		t.Fatal("Node 1 should vote for Node 0")
	}

	if cluster.Node(2).VotedFor != 0 {
		t.Fatal("Node 2 should vote for Node 0")
	}
}

func TestReplicateCommand(t *testing.T) {
	cluster := NewCluster(3)

	votes := cluster.RequestVotes(0)

	if votes < 2 {
		t.Fatalf(
			"leader election failed, got %d votes",
			votes,
		)
	}

	parkingLot := parkinglogic.NewParkingLot()

	success := cluster.ReplicateCommand(
		0,
		"ENTER 001 AP39AB1234",
		parkingLot,
	)

	if !success {
		t.Fatal("command should be replicated and committed")
	}

	// Check all three nodes received the command.
	for i := 0; i < 3; i++ {
		node := cluster.Node(i)

		if len(node.Log) != 1 {
			t.Fatalf(
				"Node %d expected 1 log entry, got %d",
				i,
				len(node.Log),
			)
		}

		if node.Log[0].Command != "ENTER 001 AP39AB1234" {
			t.Fatalf(
				"Node %d has unexpected command: %s",
				i,
				node.Log[0].Command,
			)
		}
	}

	// Leader should have committed the entry.
	if cluster.Node(0).CommitIndex != 0 {
		t.Fatalf(
			"expected leader commit index 0, got %d",
			cluster.Node(0).CommitIndex,
		)
	}

	// Command should automatically update parking state.
	status := parkingLot.Status("001")

	if status != "OCCUPIED" {
		t.Fatalf(
			"expected slot 001 to be OCCUPIED, got %s",
			status,
		)
	}

	// Leader should have applied the entry.
	if cluster.Node(0).LastApplied != 1 {
		t.Fatalf(
			"expected LastApplied 1, got %d",
			cluster.Node(0).LastApplied,
		)
	}
}

func TestApplyParkingCommand(t *testing.T) {
	parkingLot := parkinglogic.NewParkingLot()

	err := ApplyCommand(
		parkingLot,
		"ENTER 001 AP39AB1234",
	)

	if err != nil {
		t.Fatalf(
			"ENTER command failed: %v",
			err,
		)
	}

	status := parkingLot.Status("001")

	if status != "OCCUPIED" {
		t.Fatalf(
			"expected OCCUPIED, got %s",
			status,
		)
	}

	err = ApplyCommand(
		parkingLot,
		"EXIT 001 AP39AB1234",
	)

	if err != nil {
		t.Fatalf(
			"EXIT command failed: %v",
			err,
		)
	}

	status = parkingLot.Status("001")

	if status != "AVAILABLE" {
		t.Fatalf(
			"expected AVAILABLE, got %s",
			status,
		)
	}
}

func TestLeaderFailureAndNewElection(t *testing.T) {
	cluster := NewCluster(3)

	// Elect Node 0 as the first leader.
	votes := cluster.RequestVotes(0)

	if votes != 3 {
		t.Fatalf(
			"expected 3 votes for Node 0, got %d",
			votes,
		)
	}

	if cluster.Node(0).State != Leader {
		t.Fatal("Node 0 should be the initial leader")
	}

	// Simulate Node 0 failure.
	cluster.FailNode(0)

	if cluster.IsAlive(0) {
		t.Fatal("Node 0 should be marked as failed")
	}

	// Node 1 starts a new election.
	newVotes := cluster.RequestVotes(1)

	if newVotes != 2 {
		t.Fatalf(
			"expected 2 votes for Node 1, got %d",
			newVotes,
		)
	}

	// Node 1 should become the new leader.
	if cluster.Node(1).State != Leader {
		t.Fatal("Node 1 should become the new leader")
	}

	// Node 1 should have a newer term.
	if cluster.Node(1).CurrentTerm != 2 {
		t.Fatalf(
			"expected Node 1 term 2, got %d",
			cluster.Node(1).CurrentTerm,
		)
	}

	// Node 0 should remain failed.
	if cluster.IsAlive(0) {
		t.Fatal("Node 0 should still be failed")
	}
}

func TestNodeRecoveryAndCatchUp(t *testing.T) {
	cluster := NewCluster(3)

	// Elect Node 0 as leader.
	votes := cluster.RequestVotes(0)

	if votes != 3 {
		t.Fatalf(
			"expected 3 votes, got %d",
			votes,
		)
	}

	if cluster.Node(0).State != Leader {
		t.Fatal("Node 0 should be leader")
	}

	// Node 2 fails.
	cluster.FailNode(2)

	if cluster.IsAlive(2) {
		t.Fatal("Node 2 should be failed")
	}

	parkingLot := parkinglogic.NewParkingLot()

	// Replicate a command while Node 2 is down.
	success := cluster.ReplicateCommand(
		0,
		"ENTER 001 AP39AB1234",
		parkingLot,
	)

	if !success {
		t.Fatal("command should commit with two available nodes")
	}

	// Node 2 should not have received the new command.
	if len(cluster.Node(2).Log) != 0 {
		t.Fatalf(
			"failed Node 2 should have 0 log entries, got %d",
			len(cluster.Node(2).Log),
		)
	}

	// Recover Node 2.
	cluster.RecoverNode(2)

	if !cluster.IsAlive(2) {
		t.Fatal("Node 2 should be recovered")
	}

	// Catch Node 2 up with the leader.
	success = cluster.CatchUpNode(2, 0)

	if !success {
		t.Fatal("Node 2 should catch up successfully")
	}

	// Node 2 should now have the missing command.
	if len(cluster.Node(2).Log) != 1 {
		t.Fatalf(
			"expected Node 2 to have 1 log entry, got %d",
			len(cluster.Node(2).Log),
		)
	}

	if cluster.Node(2).Log[0].Command != "ENTER 001 AP39AB1234" {
		t.Fatalf(
			"unexpected recovered command: %s",
			cluster.Node(2).Log[0].Command,
		)
	}

	if cluster.Node(2).CommitIndex != 0 {
		t.Fatalf(
			"expected Node 2 commit index 0, got %d",
			cluster.Node(2).CommitIndex,
		)
	}
}

func TestMultipleParkingCommands(t *testing.T) {
	cluster := NewCluster(3)

	// Elect Node 0 as leader.
	votes := cluster.RequestVotes(0)

	if votes < 2 {
		t.Fatalf(
			"leader election failed, got %d votes",
			votes,
		)
	}

	parkingLot := parkinglogic.NewParkingLot()

	// Command 1
	if !cluster.ReplicateCommand(
		0,
		"ENTER 001 AP39AB1234",
		parkingLot,
	) {
		t.Fatal("ENTER 001 should succeed")
	}

	// Command 2
	if !cluster.ReplicateCommand(
		0,
		"ENTER 002 AP39AB5678",
		parkingLot,
	) {
		t.Fatal("ENTER 002 should succeed")
	}

	// Command 3
	if !cluster.ReplicateCommand(
		0,
		"EXIT 001 AP39AB1234",
		parkingLot,
	) {
		t.Fatal("EXIT 001 should succeed")
	}

	// Command 4
	if !cluster.ReplicateCommand(
		0,
		"ENTER 003 AP39AB9999",
		parkingLot,
	) {
		t.Fatal("ENTER 003 should succeed")
	}

	// Check final parking state.
	if status := parkingLot.Status("001"); status != "AVAILABLE" {
		t.Fatalf(
			"slot 001: expected AVAILABLE, got %s",
			status,
		)
	}

	if status := parkingLot.Status("002"); status != "OCCUPIED" {
		t.Fatalf(
			"slot 002: expected OCCUPIED, got %s",
			status,
		)
	}

	if status := parkingLot.Status("003"); status != "OCCUPIED" {
		t.Fatalf(
			"slot 003: expected OCCUPIED, got %s",
			status,
		)
	}

	// All three nodes should have four log entries.
	for i := 0; i < 3; i++ {
		node := cluster.Node(i)

		if len(node.Log) != 4 {
			t.Fatalf(
				"Node %d: expected 4 log entries, got %d",
				i,
				len(node.Log),
			)
		}

		if node.CommitIndex != 3 {
			t.Fatalf(
				"Node %d: expected commit index 3, got %d",
				i,
				node.CommitIndex,
			)
		}
	}

	// Leader should have applied all four entries.
	if cluster.Node(0).LastApplied != 4 {
		t.Fatalf(
			"expected LastApplied 4, got %d",
			cluster.Node(0).LastApplied,
		)
	}
}

func TestFollowerStateMachineApplication(t *testing.T) {
	cluster := NewCluster(3)

	// Elect Node 0 as leader.
	votes := cluster.RequestVotes(0)

	if votes < 2 {
		t.Fatalf(
			"leader election failed, got %d votes",
			votes,
		)
	}

	leaderParking := parkinglogic.NewParkingLot()

	// Replicate parking operation.
	success := cluster.ReplicateCommand(
		0,
		"ENTER 010 AP39AB1111",
		leaderParking,
	)

	if !success {
		t.Fatal("command should be committed")
	}

	// Check leader.
	if leaderParking.Status("010") != "OCCUPIED" {
		t.Fatal("leader should show slot 010 as OCCUPIED")
	}

	// Check follower 1.
	follower1 := cluster.StateMachine(1)

	if follower1.Status("010") != "OCCUPIED" {
		t.Fatal("Follower 1 should apply the committed command")
	}

	// Check follower 2.
	follower2 := cluster.StateMachine(2)

	if follower2.Status("010") != "OCCUPIED" {
		t.Fatal("Follower 2 should apply the committed command")
	}

	// Verify LastApplied.
	for i := 0; i < 3; i++ {
		node := cluster.Node(i)

		if node.LastApplied != 1 {
			t.Fatalf(
				"Node %d: expected LastApplied 1, got %d",
				i,
				node.LastApplied,
			)
		}
	}
}

func TestClusterParkingConsistency(t *testing.T) {
	cluster := NewCluster(3)

	// Elect Node 0 as leader.
	votes := cluster.RequestVotes(0)

	if votes < 2 {
		t.Fatalf(
			"leader election failed, got %d votes",
			votes,
		)
	}

	leaderParking := parkinglogic.NewParkingLot()

	commands := []string{
		"ENTER 001 AP39AB1234",
		"ENTER 002 AP39AB5678",
		"EXIT 001 AP39AB1234",
		"ENTER 003 AP39AB9999",
	}

	for _, command := range commands {
		if !cluster.ReplicateCommand(
			0,
			command,
			leaderParking,
		) {
			t.Fatalf(
				"command failed: %s",
				command,
			)
		}
	}

	expected := map[string]string{
		"001": "AVAILABLE",
		"002": "OCCUPIED",
		"003": "OCCUPIED",
	}

	// Check leader state.
	for slot, expectedStatus := range expected {
		actual := leaderParking.Status(slot)

		if actual != expectedStatus {
			t.Fatalf(
				"leader slot %s: expected %s, got %s",
				slot,
				expectedStatus,
				actual,
			)
		}
	}

	// Check every follower state.
	for nodeID := 1; nodeID < 3; nodeID++ {
		followerParking := cluster.StateMachine(nodeID)

		for slot, expectedStatus := range expected {
			actual := followerParking.Status(slot)

			if actual != expectedStatus {
				t.Fatalf(
					"Node %d slot %s: expected %s, got %s",
					nodeID,
					slot,
					expectedStatus,
					actual,
				)
			}
		}
	}

	// Verify logs and indexes.
	for nodeID := 0; nodeID < 3; nodeID++ {
		node := cluster.Node(nodeID)

		if len(node.Log) != 4 {
			t.Fatalf(
				"Node %d: expected 4 log entries, got %d",
				nodeID,
				len(node.Log),
			)
		}

		if node.CommitIndex != 3 {
			t.Fatalf(
				"Node %d: expected commit index 3, got %d",
				nodeID,
				node.CommitIndex,
			)
		}

		if node.LastApplied != 4 {
			t.Fatalf(
				"Node %d: expected LastApplied 4, got %d",
				nodeID,
				node.LastApplied,
			)
		}
	}
}

func TestFailureRecoveryConsistency(t *testing.T) {
	cluster := NewCluster(3)

	// Elect Node 0 as leader.
	votes := cluster.RequestVotes(0)

	if votes < 2 {
		t.Fatalf(
			"leader election failed, got %d votes",
			votes,
		)
	}

	leaderParking := parkinglogic.NewParkingLot()

	// First command is replicated to all nodes.
	if !cluster.ReplicateCommand(
		0,
		"ENTER 001 AP39AB1234",
		leaderParking,
	) {
		t.Fatal("first command should succeed")
	}

	// Fail Node 2.
	cluster.FailNode(2)

	if cluster.IsAlive(2) {
		t.Fatal("Node 2 should be failed")
	}

	// Second command commits with Node 0 and Node 1.
	if !cluster.ReplicateCommand(
		0,
		"ENTER 002 AP39AB5678",
		leaderParking,
	) {
		t.Fatal("second command should commit with majority")
	}

	// Failed Node 2 should still have only the first command.
	if len(cluster.Node(2).Log) != 1 {
		t.Fatalf(
			"failed Node 2 should have 1 log entry, got %d",
			len(cluster.Node(2).Log),
		)
	}

	// Recover Node 2.
	cluster.RecoverNode(2)

	if !cluster.IsAlive(2) {
		t.Fatal("Node 2 should be recovered")
	}

	// Catch Node 2 up with the leader.
	if !cluster.CatchUpNode(2, 0) {
		t.Fatal("Node 2 should catch up")
	}

	// Node 2 should now have both commands.
	if len(cluster.Node(2).Log) != 2 {
		t.Fatalf(
			"Node 2 should have 2 log entries after recovery, got %d",
			len(cluster.Node(2).Log),
		)
	}

	// Verify recovered state machine.
	recoveredParking := cluster.StateMachine(2)

	if recoveredParking.Status("001") != "OCCUPIED" {
		t.Fatal(
			"recovered Node 2 should have slot 001 occupied",
		)
	}

	if recoveredParking.Status("002") != "OCCUPIED" {
		t.Fatal(
			"recovered Node 2 should have slot 002 occupied",
		)
	}

	if cluster.Node(2).CommitIndex != 1 {
		t.Fatalf(
			"expected Node 2 commit index 1, got %d",
			cluster.Node(2).CommitIndex,
		)
	}

	if cluster.Node(2).LastApplied != 2 {
		t.Fatalf(
			"expected Node 2 LastApplied 2, got %d",
			cluster.Node(2).LastApplied,
		)
	}
}
