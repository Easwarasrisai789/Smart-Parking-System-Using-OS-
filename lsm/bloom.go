package lsm

import (
	"hash/fnv"
	"sync"
)

type BloomFilter struct {
	mu     sync.RWMutex
	bits   []bool
	size   uint
	hashes uint
}

func NewBloomFilter(size uint, hashes uint) *BloomFilter {
	return &BloomFilter{
		bits:   make([]bool, size),
		size:   size,
		hashes: hashes,
	}
}

func (b *BloomFilter) Add(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := uint(0); i < b.hashes; i++ {
		index := b.hash(key, i)
		b.bits[index] = true
	}
}

func (b *BloomFilter) MightContain(key string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for i := uint(0); i < b.hashes; i++ {
		index := b.hash(key, i)

		if !b.bits[index] {
			return false
		}
	}

	return true
}

func (b *BloomFilter) hash(key string, seed uint) uint {
	h := fnv.New64a()

	_, _ = h.Write([]byte(key))

	hashValue := h.Sum64()

	hashValue ^= uint64(seed) * 0x9e3779b97f4a7c15

	return uint(hashValue % uint64(b.size))
}
