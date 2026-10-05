package lsm

import "testing"

func TestBloomFilterAddAndContains(t *testing.T) {
	filter := NewBloomFilter(1000, 3)

	filter.Add("slot:001")
	filter.Add("slot:002")
	filter.Add("slot:003")

	if !filter.MightContain("slot:001") {
		t.Fatal("slot:001 should be present")
	}

	if !filter.MightContain("slot:002") {
		t.Fatal("slot:002 should be present")
	}

	if !filter.MightContain("slot:003") {
		t.Fatal("slot:003 should be present")
	}
}

func TestBloomFilterMissingKey(t *testing.T) {
	filter := NewBloomFilter(1000, 3)

	filter.Add("slot:001")
	filter.Add("slot:002")

	if filter.MightContain("slot:999") {
		t.Fatal("slot:999 should probably not be present")
	}
}

func TestBloomFilterEmpty(t *testing.T) {
	filter := NewBloomFilter(1000, 3)

	if filter.MightContain("slot:001") {
		t.Fatal("empty Bloom filter should not contain slot:001")
	}
}

func TestBloomFilterConcurrentAccess(t *testing.T) {
	filter := NewBloomFilter(10000, 3)

	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(i int) {
			key := "slot:" + string(rune(i+1000))
			filter.Add(key)
			filter.MightContain(key)

			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
