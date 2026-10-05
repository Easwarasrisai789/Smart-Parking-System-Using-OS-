package raft

import (
	parkinglogic "smart-parking-kv/parking"
	"testing"
	"time"
)

func TestNewRaft(t *testing.T) {
	peers := []int{0, 1, 2}

	r := NewRaft(0, peers)

	if r.ID != 0 {
		t.Fatalf("expected ID 0, got %d", r.ID)
	}

	if r.State != Follower {
		t.Fatalf("expected Follower state")
	}

	if r.CurrentTerm != 0 {
		t.Fatalf("expected term 0, got %d", r.CurrentTerm)
	}

	if r.VotedFor != -1 {
		t.Fatalf("expected VotedFor -1, got %d", r.VotedFor)
	}

	if len(r.Log) != 0 {
		t.Fatalf("expected empty log")
	}
}
func TestStartElection(t *testing.T) {
	r := NewRaft(0, []int{0, 1, 2})

	if r.State != Follower {
		t.Fatal("new Raft node should start as follower")
	}

	r.StartElection()

	if r.State != Candidate {
		t.Fatal("node should become candidate")
	}

	if r.CurrentTerm != 1 {
		t.Fatalf(
			"expected term 1, got %d",
			r.CurrentTerm,
		)
	}

	if r.VotedFor != 0 {
		t.Fatalf(
			"expected vote for self (0), got %d",
			r.VotedFor,
		)
	}
}

func TestBecomeLeader(t *testing.T) {
	r := NewRaft(0, []int{0, 1, 2})

	r.StartElection()
	r.BecomeLeader()

	if r.State != Leader {
		t.Fatal("node should become leader")
	}

	if r.NextIndex == nil {
		t.Fatal("NextIndex should be initialized")
	}

	if len(r.NextIndex) != 3 {
		t.Fatalf(
			"expected 3 NextIndex entries, got %d",
			len(r.NextIndex),
		)
	}
}

func TestBecomeFollower(t *testing.T) {
	r := NewRaft(0, []int{0, 1, 2})

	r.StartElection()
	r.BecomeLeader()

	r.BecomeFollower(5)

	if r.State != Follower {
		t.Fatal("node should become follower")
	}

	if r.CurrentTerm != 5 {
		t.Fatalf(
			"expected term 5, got %d",
			r.CurrentTerm,
		)
	}

	if r.VotedFor != -1 {
		t.Fatalf(
			"expected VotedFor -1, got %d",
			r.VotedFor,
		)
	}
}
func TestRequestVote(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	args := &RequestVoteArgs{
		Term:         1,
		CandidateID:  0,
		LastLogIndex: -1,
		LastLogTerm:  0,
	}

	reply := &RequestVoteReply{}

	r.RequestVote(args, reply)

	if !reply.VoteGranted {
		t.Fatal("vote should be granted")
	}

	if r.VotedFor != 0 {
		t.Fatalf(
			"expected vote for candidate 0, got %d",
			r.VotedFor,
		)
	}
}

func TestGrantedVoteResetsElectionTimer(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})
	oldHeartbeat := time.Now().Add(-time.Second)
	r.mu.Lock()
	r.lastHeartbeat = oldHeartbeat
	r.mu.Unlock()

	args := &RequestVoteArgs{
		Term:         1,
		CandidateID:  0,
		LastLogIndex: -1,
		LastLogTerm:  0,
	}
	reply := &RequestVoteReply{}

	r.RequestVote(args, reply)

	if !reply.VoteGranted {
		t.Fatal("vote should be granted")
	}
	if !r.LastHeartbeat().After(oldHeartbeat) {
		t.Fatal("granting a vote should reset the election timer")
	}
}

func TestRequestVoteRejectsOlderTerm(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	r.CurrentTerm = 2

	args := &RequestVoteArgs{
		Term:         1,
		CandidateID:  0,
		LastLogIndex: -1,
		LastLogTerm:  0,
	}

	reply := &RequestVoteReply{}

	r.RequestVote(args, reply)

	if reply.VoteGranted {
		t.Fatal("vote should not be granted for older term")
	}
}

func TestRequestVoteRejectsSecondCandidate(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	first := &RequestVoteArgs{
		Term:         1,
		CandidateID:  0,
		LastLogIndex: -1,
		LastLogTerm:  0,
	}

	firstReply := &RequestVoteReply{}

	r.RequestVote(first, firstReply)

	if !firstReply.VoteGranted {
		t.Fatal("first vote should be granted")
	}

	second := &RequestVoteArgs{
		Term:         1,
		CandidateID:  2,
		LastLogIndex: -1,
		LastLogTerm:  0,
	}

	secondReply := &RequestVoteReply{}

	r.RequestVote(second, secondReply)

	if secondReply.VoteGranted {
		t.Fatal("node should not vote for a second candidate")
	}
}
func TestAppendEntriesHeartbeat(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	r.CurrentTerm = 1

	args := &AppendEntriesArgs{
		Term:         1,
		LeaderID:     0,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	}

	reply := &AppendEntriesReply{}

	r.AppendEntries(args, reply)

	if !reply.Success {
		t.Fatal("heartbeat should succeed")
	}

	if r.State != Follower {
		t.Fatal("node should remain follower")
	}
}

func TestAppendEntriesReplicatesLog(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	r.CurrentTerm = 1

	args := &AppendEntriesArgs{
		Term:         1,
		LeaderID:     0,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{
				Term:    1,
				Command: "ENTER slot:001",
			},
		},
		LeaderCommit: 0,
	}

	reply := &AppendEntriesReply{}

	r.AppendEntries(args, reply)

	if !reply.Success {
		t.Fatal("AppendEntries should succeed")
	}

	if len(r.Log) != 1 {
		t.Fatalf(
			"expected 1 log entry, got %d",
			len(r.Log),
		)
	}

	if r.Log[0].Command != "ENTER slot:001" {
		t.Fatalf(
			"unexpected command: %s",
			r.Log[0].Command,
		)
	}
}

func TestAppendEntriesRejectsOldTerm(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	r.CurrentTerm = 2

	args := &AppendEntriesArgs{
		Term:         1,
		LeaderID:     0,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
	}

	reply := &AppendEntriesReply{}

	r.AppendEntries(args, reply)

	if reply.Success {
		t.Fatal("old leader should be rejected")
	}

	if reply.Term != 2 {
		t.Fatalf(
			"expected term 2, got %d",
			reply.Term,
		)
	}
}

func TestAppendEntriesUpdatesCommitIndex(t *testing.T) {
	r := NewRaft(1, []int{0, 1, 2})

	r.CurrentTerm = 1

	args := &AppendEntriesArgs{
		Term:         1,
		LeaderID:     0,
		PrevLogIndex: -1,
		PrevLogTerm:  0,
		Entries: []LogEntry{
			{
				Term:    1,
				Command: "ENTER slot:001",
			},
			{
				Term:    1,
				Command: "ENTER slot:002",
			},
		},
		LeaderCommit: 1,
	}

	reply := &AppendEntriesReply{}

	r.AppendEntries(args, reply)

	if !reply.Success {
		t.Fatal("AppendEntries should succeed")
	}

	if r.CommitIndex != 1 {
		t.Fatalf(
			"expected commit index 1, got %d",
			r.CommitIndex,
		)
	}
}
func TestApplyCommitted(t *testing.T) {
	r := NewRaft(0, []int{0, 1, 2})

	r.CurrentTerm = 1

	r.Log = append(
		r.Log,
		LogEntry{
			Term:    1,
			Command: "ENTER 001 AP39AB1234",
		},
	)

	r.CommitIndex = 0

	parkingLot := parkinglogic.NewParkingLot()

	err := r.ApplyCommitted(parkingLot)

	if err != nil {
		t.Fatalf(
			"ApplyCommitted failed: %v",
			err,
		)
	}

	status := parkingLot.Status("001")

	if status != "OCCUPIED" {
		t.Fatalf(
			"expected OCCUPIED, got %s",
			status,
		)
	}

	if r.LastApplied != 1 {
		t.Fatalf(
			"expected LastApplied 1, got %d",
			r.LastApplied,
		)
	}
}
