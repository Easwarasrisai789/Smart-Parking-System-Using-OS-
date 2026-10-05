package lsm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSSTableWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sst")

	mem := NewMemTable()

	mem.Put("slot:003", "AVAILABLE")
	mem.Put("slot:001", "OCCUPIED")
	mem.Put("slot:002", "AVAILABLE")

	table, err := WriteSSTable(path, mem)
	if err != nil {
		t.Fatalf("failed to write SSTable: %v", err)
	}

	if table.Path() != path {
		t.Fatalf("expected path %s, got %s", path, table.Path())
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("SSTable file does not exist: %v", err)
	}

	value, ok, err := table.Get("slot:001")
	if err != nil {
		t.Fatalf("failed to read SSTable: %v", err)
	}

	if !ok {
		t.Fatal("slot:001 was not found")
	}

	if value != "OCCUPIED" {
		t.Fatalf("expected OCCUPIED, got %s", value)
	}

	value, ok, err = table.Get("slot:002")
	if err != nil {
		t.Fatalf("failed to read SSTable: %v", err)
	}

	if !ok {
		t.Fatal("slot:002 was not found")
	}

	if value != "AVAILABLE" {
		t.Fatalf("expected AVAILABLE, got %s", value)
	}

	_, ok, err = table.Get("slot:999")
	if err != nil {
		t.Fatalf("unexpected error reading missing key: %v", err)
	}

	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestSSTableSortedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sorted.sst")

	mem := NewMemTable()

	mem.Put("slot:003", "AVAILABLE")
	mem.Put("slot:001", "OCCUPIED")
	mem.Put("slot:002", "AVAILABLE")

	_, err := WriteSSTable(path, mem)
	if err != nil {
		t.Fatalf("failed to write SSTable: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read SSTable file: %v", err)
	}

	expected := "slot:001\tOCCUPIED\n" +
		"slot:002\tAVAILABLE\n" +
		"slot:003\tAVAILABLE\n"

	if string(data) != expected {
		t.Fatalf(
			"unexpected SSTable order\nexpected:\n%s\ngot:\n%s",
			expected,
			string(data),
		)
	}
}
