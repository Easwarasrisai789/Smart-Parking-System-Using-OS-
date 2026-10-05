package raft

import (
	"os"
	"testing"

	"smart-parking-kv/kv"
	"smart-parking-kv/lsm"
	parkinglogic "smart-parking-kv/parking"
	"smart-parking-kv/wal"
)

func TestRaftWithIntegratedStorage(t *testing.T) {
	cluster := NewCluster(3)

	votes := cluster.RequestVotes(0)

	if votes < 2 {
		t.Fatalf(
			"leader election failed, got %d votes",
			votes,
		)
	}

	walPath := "test-raft-storage.wal"
	lsmDir := "test-raft-lsm"

	defer os.Remove(walPath)
	defer os.RemoveAll(lsmDir)

	log, err := wal.Open(walPath)

	if err != nil {
		t.Fatalf(
			"failed to open WAL: %v",
			err,
		)
	}

	defer log.Close()

	tree := lsm.NewLSMTree(lsmDir, 10)

	store := kv.NewStoreWithLSM(
		log,
		tree,
	)

	parkingLot := parkinglogic.NewParkingLotWithStore(
		store,
	)

	// Command enters Raft first.
	success := cluster.ReplicateCommand(
		0,
		"ENTER 001 AP39AB1234",
		parkingLot,
	)

	if !success {
		t.Fatal(
			"Raft command should replicate and commit",
		)
	}

	// Parking state machine.
	if parkingLot.Status("001") != "OCCUPIED" {
		t.Fatal(
			"slot 001 should be OCCUPIED",
		)
	}

	// KV / LSM state.
	value, ok := store.Get("slot:001")

	if !ok {
		t.Fatal(
			"slot:001 should exist in integrated storage",
		)
	}

	if value != "OCCUPIED" {
		t.Fatalf(
			"expected OCCUPIED, got %s",
			value,
		)
	}

	// Raft log.
	leader := cluster.Node(0)

	if len(leader.Log) != 1 {
		t.Fatalf(
			"expected 1 Raft log entry, got %d",
			len(leader.Log),
		)
	}

	if leader.CommitIndex != 0 {
		t.Fatalf(
			"expected commit index 0, got %d",
			leader.CommitIndex,
		)
	}

	if leader.LastApplied != 1 {
		t.Fatalf(
			"expected LastApplied 1, got %d",
			leader.LastApplied,
		)
	}

	// ENTER creates two persistent KV writes:
	// slot:001 -> OCCUPIED
	// vehicle:AP39AB1234 -> 001
	entries, err := log.Replay()

	if err != nil {
		t.Fatalf(
			"WAL replay failed: %v",
			err,
		)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 WAL entries, got %d",
			len(entries),
		)
	}
}
func TestRaftMultipleCommandsWithIntegratedStorage(t *testing.T) {
	cluster := NewCluster(3)

	if cluster.RequestVotes(0) < 2 {
		t.Fatal("leader election failed")
	}

	walPath := "test-raft-multiple.wal"
	lsmDir := "test-raft-multiple-lsm"

	defer os.Remove(walPath)
	defer os.RemoveAll(lsmDir)

	log, err := wal.Open(walPath)

	if err != nil {
		t.Fatalf(
			"failed to open WAL: %v",
			err,
		)
	}

	defer log.Close()

	tree := lsm.NewLSMTree(lsmDir, 10)

	store := kv.NewStoreWithLSM(
		log,
		tree,
	)

	parkingLot := parkinglogic.NewParkingLotWithStore(
		store,
	)

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
			parkingLot,
		) {
			t.Fatalf(
				"command failed: %s",
				command,
			)
		}
	}

	if parkingLot.Status("001") != "AVAILABLE" {
		t.Fatal(
			"slot 001 should be AVAILABLE",
		)
	}

	if parkingLot.Status("002") != "OCCUPIED" {
		t.Fatal(
			"slot 002 should be OCCUPIED",
		)
	}

	if parkingLot.Status("003") != "OCCUPIED" {
		t.Fatal(
			"slot 003 should be OCCUPIED",
		)
	}

	leader := cluster.Node(0)

	if len(leader.Log) != 4 {
		t.Fatalf(
			"expected 4 Raft entries, got %d",
			len(leader.Log),
		)
	}

	if leader.CommitIndex != 3 {
		t.Fatalf(
			"expected commit index 3, got %d",
			leader.CommitIndex,
		)
	}

	if leader.LastApplied != 4 {
		t.Fatalf(
			"expected LastApplied 4, got %d",
			leader.LastApplied,
		)
	}
}
