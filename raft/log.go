package raft

import (
	"fmt"
	"strconv"
	"strings"

	"smart-parking-kv/wal"
)

func (r *Raft) AppendLog(command string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := LogEntry{
		Term:    r.CurrentTerm,
		Command: command,
	}

	r.Log = append(r.Log, entry)

	return nil
}

func SaveLog(logStore *wal.WAL, entries []LogEntry) error {
	for _, entry := range entries {
		command := fmt.Sprintf(
			"%d|%s",
			entry.Term,
			entry.Command,
		)

		if err := logStore.Put("raft-log", command); err != nil {
			return err
		}
	}

	return nil
}

func LoadLog(logStore *wal.WAL) ([]LogEntry, error) {
	entries, err := logStore.Replay()
	if err != nil {
		return nil, err
	}

	var result []LogEntry

	for _, entry := range entries {
		if entry.Operation != "PUT" {
			continue
		}

		if entry.Key != "raft-log" {
			continue
		}

		parts := strings.SplitN(entry.Value, "|", 2)

		if len(parts) != 2 {
			return nil, fmt.Errorf(
				"invalid raft log entry: %s",
				entry.Value,
			)
		}

		term, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf(
				"invalid raft term: %s",
				parts[0],
			)
		}

		result = append(result, LogEntry{
			Term:    term,
			Command: parts[1],
		})
	}

	return result, nil
}
