package lsm

import (
	"fmt"
	"testing"
)

func TestMemTablePutGet(t *testing.T) {
	mem := NewMemTable()

	mem.Put("slot:001", "AVAILABLE")

	value, ok := mem.Get("slot:001")

	if !ok {
		t.Fatal("key not found")
	}

	if value != "AVAILABLE" {
		t.Fatalf(
			"expected AVAILABLE, got %s",
			value,
		)
	}
}

func TestMemTableDelete(t *testing.T) {
	mem := NewMemTable()

	mem.Put("slot:001", "AVAILABLE")

	mem.Delete("slot:001")

	_, ok := mem.Get("slot:001")

	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestMemTableSize(t *testing.T) {
	mem := NewMemTable()

	mem.Put("slot:001", "AVAILABLE")
	mem.Put("slot:002", "OCCUPIED")
	mem.Put("slot:003", "AVAILABLE")

	if mem.Size() != 3 {
		t.Fatalf(
			"expected size 3, got %d",
			mem.Size(),
		)
	}
}

func TestMemTableSortedKeys(t *testing.T) {
	mem := NewMemTable()

	mem.Put("slot:003", "AVAILABLE")
	mem.Put("slot:001", "AVAILABLE")
	mem.Put("slot:002", "AVAILABLE")

	keys := mem.Keys()

	expected := []string{
		"slot:001",
		"slot:002",
		"slot:003",
	}

	for i := range expected {
		if keys[i] != expected[i] {
			t.Fatalf(
				"expected %s, got %s",
				expected[i],
				keys[i],
			)
		}
	}
}
func TestMemTableConcurrentAccess(t *testing.T) {
	mem := NewMemTable()

	const numberOfKeys = 100

	done := make(chan bool)

	// Concurrent writes.
	for i := 0; i < numberOfKeys; i++ {
		go func(i int) {
			key := fmt.Sprintf("slot:%03d", i)
			value := fmt.Sprintf("STATUS-%d", i)

			mem.Put(key, value)

			done <- true
		}(i)
	}

	// Wait for all writes.
	for i := 0; i < numberOfKeys; i++ {
		<-done
	}

	// Concurrent reads.
	for i := 0; i < numberOfKeys; i++ {
		go func(i int) {
			key := fmt.Sprintf("slot:%03d", i)

			_, ok := mem.Get(key)

			if !ok {
				t.Errorf("key %s was not found", key)
			}

			done <- true
		}(i)
	}

	// Wait for all reads.
	for i := 0; i < numberOfKeys; i++ {
		<-done
	}

	if mem.Size() != numberOfKeys {
		t.Fatalf(
			"expected %d keys, got %d",
			numberOfKeys,
			mem.Size(),
		)
	}
}
