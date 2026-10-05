package wal

import (
	"path/filepath"
	"testing"
)

func TestWALPutAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.wal")

	w, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}
	defer w.Close()

	err = w.Put("slot:001", "OCCUPIED")
	if err != nil {
		t.Fatalf("failed to write PUT: %v", err)
	}

	err = w.Put("vehicle:AP39AB1234", "001")
	if err != nil {
		t.Fatalf("failed to write second PUT: %v", err)
	}

	entries, err := w.Replay()
	if err != nil {
		t.Fatalf("failed to replay WAL: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Operation != "PUT" {
		t.Fatalf("expected PUT, got %s", entries[0].Operation)
	}

	if entries[0].Key != "slot:001" {
		t.Fatalf("expected slot:001, got %s", entries[0].Key)
	}

	if entries[0].Value != "OCCUPIED" {
		t.Fatalf("expected OCCUPIED, got %s", entries[0].Value)
	}
}

func TestWALDeleteAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.wal")

	w, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}
	defer w.Close()

	err = w.Put("slot:001", "OCCUPIED")
	if err != nil {
		t.Fatalf("failed to write PUT: %v", err)
	}

	err = w.Delete("slot:001")
	if err != nil {
		t.Fatalf("failed to write DELETE: %v", err)
	}

	entries, err := w.Replay()
	if err != nil {
		t.Fatalf("failed to replay WAL: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[1].Operation != "DELETE" {
		t.Fatalf(
			"expected DELETE, got %s",
			entries[1].Operation,
		)
	}

	if entries[1].Key != "slot:001" {
		t.Fatalf(
			"expected slot:001, got %s",
			entries[1].Key,
		)
	}
}

func TestWALPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.wal")

	// Open WAL and write data.
	w, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}

	err = w.Put("slot:001", "AVAILABLE")
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("failed to close WAL: %v", err)
	}

	// Reopen the same WAL file.
	w, err = Open(path)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer w.Close()

	entries, err := w.Replay()
	if err != nil {
		t.Fatalf("failed to replay after reopening: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Key != "slot:001" {
		t.Fatalf(
			"expected slot:001, got %s",
			entries[0].Key,
		)
	}

	if entries[0].Value != "AVAILABLE" {
		t.Fatalf(
			"expected AVAILABLE, got %s",
			entries[0].Value,
		)
	}
}
