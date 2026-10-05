package raft

import (
	"path/filepath"
	"testing"

	"smart-parking-kv/wal"
)

func TestRaftLogPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raft.wal")

	logStore, err := wal.Open(path)
	if err != nil {
		t.Fatalf("failed to open WAL: %v", err)
	}
	defer logStore.Close()

	r := NewRaft(0, []int{0, 1, 2})

	r.CurrentTerm = 1

	if err := r.AppendLog("ENTER slot:001"); err != nil {
		t.Fatalf("AppendLog failed: %v", err)
	}

	if err := r.AppendLog("ENTER slot:002"); err != nil {
		t.Fatalf("AppendLog failed: %v", err)
	}

	if err := SaveLog(logStore, r.Log); err != nil {
		t.Fatalf("SaveLog failed: %v", err)
	}

	recovered, err := LoadLog(logStore)
	if err != nil {
		t.Fatalf("LoadLog failed: %v", err)
	}

	if len(recovered) != 2 {
		t.Fatalf(
			"expected 2 recovered entries, got %d",
			len(recovered),
		)
	}

	if recovered[0].Term != 1 {
		t.Fatalf("expected term 1, got %d", recovered[0].Term)
	}

	if recovered[0].Command != "ENTER slot:001" {
		t.Fatalf(
			"unexpected command: %s",
			recovered[0].Command,
		)
	}

	if recovered[1].Command != "ENTER slot:002" {
		t.Fatalf(
			"unexpected command: %s",
			recovered[1].Command,
		)
	}
}
