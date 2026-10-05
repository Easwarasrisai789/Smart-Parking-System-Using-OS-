package kv

import (
	"fmt"
	"sync"

	"smart-parking-kv/lsm"
	"smart-parking-kv/wal"
)

type Store struct {
	mu sync.RWMutex

	data map[string]string

	log *wal.WAL

	tree *lsm.LSMTree
}

func NewStore(log *wal.WAL) *Store {
	return &Store{
		data: make(map[string]string),
		log:  log,
	}
}

func NewStoreWithLSM(
	log *wal.WAL,
	tree *lsm.LSMTree,
) *Store {
	return &Store{
		data: make(map[string]string),
		log:  log,
		tree: tree,
	}
}

func (s *Store) Put(key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Write to WAL first.
	if s.log != nil {
		if err := s.log.Put(key, value); err != nil {
			return fmt.Errorf("WAL PUT failed: %w", err)
		}
	}

	// 2. Write to LSM tree.
	if s.tree != nil {
		if err := s.tree.Put(key, value); err != nil {
			return fmt.Errorf("LSM PUT failed: %w", err)
		}
	}

	// 3. Update in-memory state.
	s.data[key] = value

	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read from LSM when configured.
	if s.tree != nil {
		value, found, err := s.tree.Get(key)

		if err == nil && found {
			return value, true
		}
	}

	// Fallback to in-memory map.
	value, ok := s.data[key]

	return value, ok
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Write DELETE to WAL first.
	if s.log != nil {
		if err := s.log.Delete(key); err != nil {
			return fmt.Errorf("WAL DELETE failed: %w", err)
		}
	}

	// 2. Write tombstone to LSM.
	if s.tree != nil {
		if err := s.tree.Delete(key); err != nil {
			return fmt.Errorf("LSM DELETE failed: %w", err)
		}
	}

	// 3. Remove from in-memory map.
	delete(s.data, key)

	return nil
}
