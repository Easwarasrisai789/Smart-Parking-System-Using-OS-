package wal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Entry struct {
	Operation string
	Key       string
	Value     string
}

type WAL struct {
	mu   sync.Mutex
	file *os.File
}

func Open(path string) (*WAL, error) {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_RDWR,
		0644,
	)

	if err != nil {
		return nil, err
	}

	return &WAL{
		file: file,
	}, nil
}

func (w *WAL) Put(key string, value string) error {
	return w.writeEntry(Entry{
		Operation: "PUT",
		Key:       key,
		Value:     value,
	})
}

func (w *WAL) Delete(key string) error {
	return w.writeEntry(Entry{
		Operation: "DELETE",
		Key:       key,
	})
}

func (w *WAL) writeEntry(entry Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var line string

	switch entry.Operation {
	case "PUT":
		line = fmt.Sprintf(
			"PUT|%s|%s\n",
			entry.Key,
			entry.Value,
		)

	case "DELETE":
		line = fmt.Sprintf(
			"DELETE|%s\n",
			entry.Key,
		)

	default:
		return fmt.Errorf(
			"unknown WAL operation: %s",
			entry.Operation,
		)
	}

	_, err := w.file.WriteString(line)

	if err != nil {
		return err
	}

	// Force the data to disk.
	return w.file.Sync()
}

func (w *WAL) Replay() ([]Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(w.file)

	var entries []Entry

	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.SplitN(line, "|", 3)

		switch parts[0] {
		case "PUT":
			if len(parts) != 3 {
				return nil, fmt.Errorf(
					"invalid PUT entry: %s",
					line,
				)
			}

			entries = append(entries, Entry{
				Operation: "PUT",
				Key:       parts[1],
				Value:     parts[2],
			})

		case "DELETE":
			if len(parts) != 2 {
				return nil, fmt.Errorf(
					"invalid DELETE entry: %s",
					line,
				)
			}

			entries = append(entries, Entry{
				Operation: "DELETE",
				Key:       parts[1],
			})

		default:
			return nil, fmt.Errorf(
				"unknown WAL operation: %s",
				parts[0],
			)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return to the end so future writes append correctly.
	_, err := w.file.Seek(0, 2)

	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}
