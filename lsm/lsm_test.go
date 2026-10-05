package lsm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLSMTreePutAndGet(t *testing.T) {
	dir := t.TempDir()

	tree := NewLSMTree(dir, 10)

	err := tree.Put("slot:001", "AVAILABLE")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, ok, err := tree.Get("slot:001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !ok {
		t.Fatal("slot:001 was not found")
	}

	if value != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE, got %s", value)
	}
}

func TestLSMTreeAutomaticFlush(t *testing.T) {
	dir := t.TempDir()

	// Flush after 3 keys.
	tree := NewLSMTree(dir, 3)

	if err := tree.Put("slot:001", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := tree.Put("slot:002", "OCCUPIED"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := tree.Put("slot:003", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if tree.SSTableCount() != 1 {
		t.Fatalf(
			"expected 1 SSTable, got %d",
			tree.SSTableCount(),
		)
	}

	value, ok, err := tree.Get("slot:002")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !ok {
		t.Fatal("slot:002 was not found after flush")
	}

	if value != "OCCUPIED" {
		t.Fatalf(
			"expected OCCUPIED, got %s",
			value,
		)
	}
}

func TestLSMTreeManualFlush(t *testing.T) {
	dir := t.TempDir()

	tree := NewLSMTree(dir, 100)

	if err := tree.Put("slot:001", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if tree.SSTableCount() != 0 {
		t.Fatal("SSTable should not exist before manual flush")
	}

	if err := tree.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if tree.SSTableCount() != 1 {
		t.Fatalf(
			"expected 1 SSTable, got %d",
			tree.SSTableCount(),
		)
	}

	value, ok, err := tree.Get("slot:001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !ok {
		t.Fatal("slot:001 was not found after flush")
	}

	if value != "AVAILABLE" {
		t.Fatalf(
			"expected AVAILABLE, got %s",
			value,
		)
	}
}

func TestLSMTreeMissingKey(t *testing.T) {
	dir := t.TempDir()

	tree := NewLSMTree(dir, 10)

	value, ok, err := tree.Get("slot:999")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if ok {
		t.Fatal("missing key should not be found")
	}

	if value != "" {
		t.Fatalf(
			"expected empty value, got %s",
			value,
		)
	}
}

func TestLSMTreeMultipleFlushes(t *testing.T) {
	dir := t.TempDir()

	tree := NewLSMTree(dir, 2)

	if err := tree.Put("slot:001", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := tree.Put("slot:002", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := tree.Put("slot:003", "OCCUPIED"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := tree.Put("slot:004", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if tree.SSTableCount() != 2 {
		t.Fatalf(
			"expected 2 SSTables, got %d",
			tree.SSTableCount(),
		)
	}

	value, ok, err := tree.Get("slot:001")
	if err != nil {
		t.Fatalf("Get slot:001 failed: %v", err)
	}

	if !ok || value != "AVAILABLE" {
		t.Fatalf("unexpected slot:001 value: %s", value)
	}

	value, ok, err = tree.Get("slot:003")
	if err != nil {
		t.Fatalf("Get slot:003 failed: %v", err)
	}

	if !ok || value != "OCCUPIED" {
		t.Fatalf("unexpected slot:003 value: %s", value)
	}
}

func TestLSMTreeSSTableFiles(t *testing.T) {
	dir := t.TempDir()

	tree := NewLSMTree(dir, 2)

	if err := tree.Put("slot:001", "AVAILABLE"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if err := tree.Put("slot:002", "OCCUPIED"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	expectedPath := filepath.Join(dir, "sstable-1.sst")

	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf(
			"expected SSTable file %s: %v",
			expectedPath,
			err,
		)
	}
}
func TestLSMTreeDelete(t *testing.T) {
	dir := t.TempDir()

	tree := NewLSMTree(dir, 10)

	// Add a parking slot.
	err := tree.Put("slot:001", "AVAILABLE")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify it exists.
	value, ok, err := tree.Get("slot:001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !ok {
		t.Fatal("slot:001 should exist")
	}

	if value != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE, got %s", value)
	}

	// Delete the slot.
	err = tree.Delete("slot:001")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it is no longer visible.
	value, ok, err = tree.Get("slot:001")
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}

	if ok {
		t.Fatal("slot:001 should not exist after delete")
	}

	if value != "" {
		t.Fatalf(
			"expected empty value after delete, got %s",
			value,
		)
	}
}

func TestLSMTreeDeleteAfterFlush(t *testing.T) {
	dir := t.TempDir()

	// Small MemTable so operations are flushed to SSTables.
	tree := NewLSMTree(dir, 2)

	// First SSTable.
	err := tree.Put("slot:001", "AVAILABLE")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	err = tree.Put("slot:002", "AVAILABLE")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Delete slot:001.
	err = tree.Delete("slot:001")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Another write triggers another SSTable.
	err = tree.Put("slot:003", "AVAILABLE")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// slot:001 must remain deleted.
	_, ok, err := tree.Get("slot:001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if ok {
		t.Fatal("slot:001 should remain deleted")
	}
}
