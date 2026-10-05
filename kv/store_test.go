package kv

import (
	"path/filepath"
	"testing"

	"smart-parking-kv/wal"
)

func TestStore(t *testing.T) {
	store := NewStore(nil)

	err := store.Put("slot:001", "AVAILABLE")
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	value, ok := store.Get("slot:001")

	if !ok {
		t.Fatal("slot was not found")
	}

	if value != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE, got %s", value)
	}
}

func TestStoreWALIntegration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parking.wal")

	// Open WAL.
	log, err := wal.Open(path)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}

	// Create KV store using the WAL.
	store := NewStore(log)

	// Store parking information.
	err = store.Put("slot:001", "OCCUPIED")
	if err != nil {
		t.Fatalf("failed to store slot: %v", err)
	}

	err = store.Put("vehicle:AP39AB1234", "001")
	if err != nil {
		t.Fatalf("failed to store vehicle: %v", err)
	}

	err = store.Delete("vehicle:AP39AB1234")
	if err != nil {
		t.Fatalf("failed to delete vehicle: %v", err)
	}

	// Replay the WAL.
	entries, err := log.Replay()
	if err != nil {
		t.Fatalf("failed to replay WAL: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf(
			"expected 3 WAL entries, got %d",
			len(entries),
		)
	}

	// Check first PUT.
	if entries[0].Operation != "PUT" {
		t.Fatalf(
			"expected first operation PUT, got %s",
			entries[0].Operation,
		)
	}

	if entries[0].Key != "slot:001" {
		t.Fatalf(
			"expected first key slot:001, got %s",
			entries[0].Key,
		)
	}

	if entries[0].Value != "OCCUPIED" {
		t.Fatalf(
			"expected first value OCCUPIED, got %s",
			entries[0].Value,
		)
	}

	// Check second PUT.
	if entries[1].Operation != "PUT" {
		t.Fatalf(
			"expected second operation PUT, got %s",
			entries[1].Operation,
		)
	}

	if entries[1].Key != "vehicle:AP39AB1234" {
		t.Fatalf(
			"expected vehicle key, got %s",
			entries[1].Key,
		)
	}

	// Check DELETE.
	if entries[2].Operation != "DELETE" {
		t.Fatalf(
			"expected third operation DELETE, got %s",
			entries[2].Operation,
		)
	}

	if entries[2].Key != "vehicle:AP39AB1234" {
		t.Fatalf(
			"expected vehicle key in DELETE, got %s",
			entries[2].Key,
		)
	}

	err = log.Close()
	if err != nil {
		t.Fatalf("failed to close WAL: %v", err)
	}
}
