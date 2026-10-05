package lsm

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const tombstone = "__TOMBSTONE__"

func CompactSSTables(outputPath string, tables []*SSTable) (*SSTable, error) {
	merged := make(map[string]string)

	// Read tables from oldest to newest.
	// Newer values overwrite older values.
	for _, table := range tables {
		file, err := os.Open(table.Path())
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			parts := strings.SplitN(scanner.Text(), "\t", 2)

			if len(parts) != 2 {
				continue
			}

			key := parts[0]
			value := parts[1]

			merged[key] = value
		}

		if err := scanner.Err(); err != nil {
			file.Close()
			return nil, err
		}

		if err := file.Close(); err != nil {
			return nil, err
		}
	}

	// Remove tombstones during compaction.
	for key, value := range merged {
		if value == tombstone {
			delete(merged, key)
		}
	}

	// Create a new MemTable with the remaining values.
	mem := NewMemTable()

	for key, value := range merged {
		mem.Put(key, value)
	}

	return WriteSSTable(outputPath, mem)
}

func RemoveSSTables(tables []*SSTable) error {
	for _, table := range tables {
		if err := os.Remove(table.Path()); err != nil {
			return fmt.Errorf(
				"failed to remove SSTable %s: %w",
				table.Path(),
				err,
			)
		}
	}

	return nil
}
