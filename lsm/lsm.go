package lsm

import (
	"fmt"
	"path/filepath"
	"sync"
)

type LSMTree struct {
	mu sync.RWMutex

	dir string

	memTable *MemTable

	sstables []*SSTable

	maxMemTableSize int
}

// NewLSMTree creates a new LSM tree.
func NewLSMTree(dir string, maxMemTableSize int) *LSMTree {
	return &LSMTree{
		dir:             dir,
		memTable:        NewMemTable(),
		sstables:        make([]*SSTable, 0),
		maxMemTableSize: maxMemTableSize,
	}
}

// Put adds or updates a key in the MemTable.
func (l *LSMTree) Put(key string, value string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.memTable.Put(key, value)

	if l.memTable.Size() >= l.maxMemTableSize {
		return l.flush()
	}

	return nil
}

// Get searches the MemTable first and then SSTables.
func (l *LSMTree) Get(key string) (string, bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check MemTable first.
	if value, ok := l.memTable.Get(key); ok {
		if value == "__TOMBSTONE__" {
			return "", false, nil
		}

		return value, true, nil
	}

	// Search newest SSTable first.
	for i := len(l.sstables) - 1; i >= 0; i-- {
		value, ok, err := l.sstables[i].Get(key)

		if err != nil {
			return "", false, err
		}

		if ok {
			if value == "__TOMBSTONE__" {
				return "", false, nil
			}

			return value, true, nil
		}
	}

	return "", false, nil
}

// Flush writes the current MemTable to an SSTable.
func (l *LSMTree) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.flush()
}

func (l *LSMTree) flush() error {
	if l.memTable.Size() == 0 {
		return nil
	}

	fileName := fmt.Sprintf(
		"sstable-%d.sst",
		len(l.sstables)+1,
	)

	path := filepath.Join(l.dir, fileName)

	table, err := WriteSSTable(path, l.memTable)
	if err != nil {
		return err
	}

	l.sstables = append(l.sstables, table)

	l.memTable = NewMemTable()

	return nil
}

// SSTableCount returns the number of SSTables.
func (l *LSMTree) SSTableCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.sstables)
}

// Delete marks a key as deleted.
func (l *LSMTree) Delete(key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.memTable.Put(key, "__TOMBSTONE__")

	if l.memTable.Size() >= l.maxMemTableSize {
		return l.flush()
	}

	return nil
}
