package lsm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type SSTable struct {
	path   string
	filter *BloomFilter
}

func WriteSSTable(path string, mem *MemTable) (*SSTable, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	writer := bufio.NewWriter(file)

	keys := mem.Keys()

	filter := NewBloomFilter(10000, 3)

	for _, key := range keys {
		value, ok := mem.Get(key)

		if !ok {
			continue
		}

		filter.Add(key)

		_, err := fmt.Fprintf(
			writer,
			"%s\t%s\n",
			key,
			value,
		)

		if err != nil {
			file.Close()
			return nil, err
		}
	}

	if err := writer.Flush(); err != nil {
		file.Close()
		return nil, err
	}

	if err := file.Sync(); err != nil {
		file.Close()
		return nil, err
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	return &SSTable{
		path:   path,
		filter: filter,
	}, nil
}

func (s *SSTable) Get(key string) (string, bool, error) {
	// Bloom filter says the key definitely does not exist.
	if s.filter != nil && !s.filter.MightContain(key) {
		return "", false, nil
	}

	file, err := os.Open(s.path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 2)

		if len(parts) != 2 {
			continue
		}

		if parts[0] == key {
			return parts[1], true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", false, err
	}

	return "", false, nil
}

func (s *SSTable) Path() string {
	return s.path
}
