package lsm

import (
	"fmt"
	"testing"
)

func BenchmarkLSMTreePut(b *testing.B) {
	dir := b.TempDir()

	tree := NewLSMTree(dir, 100000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("slot:%d", i)
		value := "AVAILABLE"

		if err := tree.Put(key, value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLSMTreeGet(b *testing.B) {
	dir := b.TempDir()

	tree := NewLSMTree(dir, 100000)

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("slot:%d", i)

		if err := tree.Put(key, "AVAILABLE"); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("slot:%d", i%10000)

		_, _, err := tree.Get(key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLSMTreePutGet(b *testing.B) {
	dir := b.TempDir()

	tree := NewLSMTree(dir, 100000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("slot:%d", i)

		if err := tree.Put(key, "OCCUPIED"); err != nil {
			b.Fatal(err)
		}

		_, _, err := tree.Get(key)
		if err != nil {
			b.Fatal(err)
		}
	}
}
