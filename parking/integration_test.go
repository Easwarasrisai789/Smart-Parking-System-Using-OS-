package parking

import (
	"os"
	"testing"

	"smart-parking-kv/kv"
	"smart-parking-kv/lsm"
	"smart-parking-kv/wal"
)

func TestParkingWithWALAndLSM(t *testing.T) {
	walPath := "test-parking-integration.wal"
	lsmDir := "test-parking-lsm"

	defer os.Remove(walPath)
	defer os.RemoveAll(lsmDir)

	log, err := wal.Open(walPath)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}
	defer log.Close()

	tree := lsm.NewLSMTree(lsmDir, 10)

	store := kv.NewStoreWithLSM(
		log,
		tree,
	)

	parkingLot := NewParkingLotWithStore(store)

	// ENTER
	err = parkingLot.Enter(
		"001",
		"AP39AB1234",
	)

	if err != nil {
		t.Fatalf("ENTER failed: %v", err)
	}

	if parkingLot.Status("001") != "OCCUPIED" {
		t.Fatal("slot 001 should be OCCUPIED")
	}

	// EXIT
	err = parkingLot.Exit(
		"001",
		"AP39AB1234",
	)

	if err != nil {
		t.Fatalf("EXIT failed: %v", err)
	}

	if parkingLot.Status("001") != "AVAILABLE" {
		t.Fatal("slot 001 should be AVAILABLE")
	}

	// ENTER writes:
	// PUT slot
	// PUT vehicle
	//
	// EXIT writes:
	// PUT slot
	// DELETE vehicle
	//
	// Total = 4 WAL operations.
	entries, err := log.Replay()

	if err != nil {
		t.Fatalf("WAL replay failed: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf(
			"expected 4 WAL entries, got %d",
			len(entries),
		)
	}
}
