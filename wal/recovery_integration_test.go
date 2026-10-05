package wal

import (
	"os"
	"testing"
)

func TestWALRecoveryAfterRestart(t *testing.T) {
	path := "test-recovery.wal"

	defer os.Remove(path)

	// -----------------------------
	// Simulate first application run
	// -----------------------------

	log, err := Open(path)

	if err != nil {
		t.Fatalf(
			"failed to open WAL: %v",
			err,
		)
	}

	if err := log.Put(
		"slot:001",
		"OCCUPIED",
	); err != nil {
		t.Fatalf(
			"failed to write slot: %v",
			err,
		)
	}

	if err := log.Put(
		"vehicle:AP39AB1234",
		"001",
	); err != nil {
		t.Fatalf(
			"failed to write vehicle: %v",
			err,
		)
	}

	if err := log.Put(
		"slot:002",
		"OCCUPIED",
	); err != nil {
		t.Fatalf(
			"failed to write slot 002: %v",
			err,
		)
	}

	if err := log.Close(); err != nil {
		t.Fatalf(
			"failed to close WAL: %v",
			err,
		)
	}

	// -----------------------------
	// Simulate application restart
	// -----------------------------

	recoveredLog, err := Open(path)

	if err != nil {
		t.Fatalf(
			"failed to reopen WAL: %v",
			err,
		)
	}

	defer recoveredLog.Close()

	entries, err := recoveredLog.Replay()

	if err != nil {
		t.Fatalf(
			"failed to replay WAL: %v",
			err,
		)
	}

	// We should recover all three operations.
	if len(entries) != 3 {
		t.Fatalf(
			"expected 3 recovered entries, got %d",
			len(entries),
		)
	}

	// Verify slot 001.
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

	// Verify vehicle mapping.
	if entries[1].Key != "vehicle:AP39AB1234" {
		t.Fatalf(
			"unexpected vehicle key: %s",
			entries[1].Key,
		)
	}

	if entries[1].Value != "001" {
		t.Fatalf(
			"expected vehicle slot 001, got %s",
			entries[1].Value,
		)
	}

	// Verify slot 002.
	if entries[2].Key != "slot:002" {
		t.Fatalf(
			"unexpected third key: %s",
			entries[2].Key,
		)
	}
}

func TestWALDeleteRecovery(t *testing.T) {
	path := "test-delete-recovery.wal"

	defer os.Remove(path)

	log, err := Open(path)

	if err != nil {
		t.Fatalf(
			"failed to open WAL: %v",
			err,
		)
	}

	// Write a value.
	if err := log.Put(
		"slot:001",
		"OCCUPIED",
	); err != nil {
		t.Fatalf(
			"PUT failed: %v",
			err,
		)
	}

	// Delete the value.
	if err := log.Delete(
		"slot:001",
	); err != nil {
		t.Fatalf(
			"DELETE failed: %v",
			err,
		)
	}

	if err := log.Close(); err != nil {
		t.Fatalf(
			"close failed: %v",
			err,
		)
	}

	// Reopen after simulated restart.
	recoveredLog, err := Open(path)

	if err != nil {
		t.Fatalf(
			"reopen failed: %v",
			err,
		)
	}

	defer recoveredLog.Close()

	entries, err := recoveredLog.Replay()

	if err != nil {
		t.Fatalf(
			"replay failed: %v",
			err,
		)
	}

	if len(entries) != 2 {
		t.Fatalf(
			"expected 2 entries, got %d",
			len(entries),
		)
	}

	if entries[0].Operation != "PUT" {
		t.Fatalf(
			"expected PUT, got %s",
			entries[0].Operation,
		)
	}

	if entries[1].Operation != "DELETE" {
		t.Fatalf(
			"expected DELETE, got %s",
			entries[1].Operation,
		)

	}
}
