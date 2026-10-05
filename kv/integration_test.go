package kv

import (
	"os"
	"testing"

	"smart-parking-kv/lsm"
	"smart-parking-kv/wal"
)

func TestKVWALLSMIntegration(t *testing.T) {
	walPath := "test-kv-integration.wal"
	lsmDir := "test-kv-lsm"

	defer os.Remove(walPath)
	defer os.RemoveAll(lsmDir)

	log, err := wal.Open(walPath)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}
	defer log.Close()

	tree := lsm.NewLSMTree(lsmDir, 10)

	store := NewStoreWithLSM(log, tree)

	// PUT
	if err := store.Put("slot:001", "OCCUPIED"); err != nil {
		t.Fatalf("PUT failed: %v", err)
	}

	// GET
	value, ok := store.Get("slot:001")

	if !ok {
		t.Fatal("expected slot:001 to exist")
	}

	if value != "OCCUPIED" {
		t.Fatalf(
			"expected OCCUPIED, got %s",
			value,
		)
	}

	// DELETE
	if err := store.Delete("slot:001"); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	_, ok = store.Get("slot:001")

	if ok {
		t.Fatal("slot:001 should be deleted")
	}

	// Verify WAL.
	entries, err := log.Replay()
	if err != nil {
		t.Fatalf("WAL replay failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 WAL entries, got %d",
			len(entries),
		)
	}

	if entries[0].Operation != "PUT" {
		t.Fatalf(
			"expected first operation PUT, got %s",
			entries[0].Operation,
		)
	}

	if entries[1].Operation != "DELETE" {
		t.Fatalf(
			"expected second operation DELETE, got %s",
			entries[1].Operation,
		)
	}
}
