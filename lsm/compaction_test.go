package lsm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompactSSTables(t *testing.T) {
	dir := t.TempDir()

	// First SSTable - older data
	mem1 := NewMemTable()
	mem1.Put("slot:001", "AVAILABLE")
	mem1.Put("slot:002", "AVAILABLE")

	path1 := filepath.Join(dir, "table1.sst")

	table1, err := WriteSSTable(path1, mem1)
	if err != nil {
		t.Fatalf("failed to create first SSTable: %v", err)
	}

	// Second SSTable - newer data
	mem2 := NewMemTable()
	mem2.Put("slot:001", "OCCUPIED")
	mem2.Put("slot:003", "AVAILABLE")

	path2 := filepath.Join(dir, "table2.sst")

	table2, err := WriteSSTable(path2, mem2)
	if err != nil {
		t.Fatalf("failed to create second SSTable: %v", err)
	}

	// Compact both SSTables.
	outputPath := filepath.Join(dir, "compacted.sst")

	compacted, err := CompactSSTables(
		outputPath,
		[]*SSTable{table1, table2},
	)

	if err != nil {
		t.Fatalf("compaction failed: %v", err)
	}

	// Check that the compacted file exists.
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("compacted SSTable does not exist: %v", err)
	}

	// slot:001 should contain the newest value.
	value, ok, err := compacted.Get("slot:001")
	if err != nil {
		t.Fatalf("failed to read compacted SSTable: %v", err)
	}

	if !ok {
		t.Fatal("slot:001 was not found after compaction")
	}

	if value != "OCCUPIED" {
		t.Fatalf(
			"expected newest value OCCUPIED, got %s",
			value,
		)
	}

	// slot:002 should still exist.
	value, ok, err = compacted.Get("slot:002")
	if err != nil {
		t.Fatalf("failed to read slot:002: %v", err)
	}

	if !ok {
		t.Fatal("slot:002 was lost during compaction")
	}

	if value != "AVAILABLE" {
		t.Fatalf(
			"expected AVAILABLE for slot:002, got %s",
			value,
		)
	}

	// slot:003 should exist.
	value, ok, err = compacted.Get("slot:003")
	if err != nil {
		t.Fatalf("failed to read slot:003: %v", err)
	}

	if !ok {
		t.Fatal("slot:003 was lost during compaction")
	}

	if value != "AVAILABLE" {
		t.Fatalf(
			"expected AVAILABLE for slot:003, got %s",
			value,
		)
	}

	// A key that never existed should not be found.
	_, ok, err = compacted.Get("slot:999")
	if err != nil {
		t.Fatalf("unexpected error reading missing key: %v", err)
	}

	if ok {
		t.Fatal("slot:999 should not exist")
	}
}

func TestRemoveSSTables(t *testing.T) {
	dir := t.TempDir()

	mem := NewMemTable()
	mem.Put("slot:001", "AVAILABLE")

	path := filepath.Join(dir, "old.sst")

	table, err := WriteSSTable(path, mem)
	if err != nil {
		t.Fatalf("failed to create SSTable: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("SSTable was not created: %v", err)
	}

	err = RemoveSSTables([]*SSTable{table})
	if err != nil {
		t.Fatalf("failed to remove SSTable: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("SSTable still exists after removal")
	}
}
