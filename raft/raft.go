package raft

import (
	parkinglogic "smart-parking-kv/parking"
	"sync"
	"time"
)

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

type LogEntry struct {
	Term    int
	Command string
}

type Raft struct {
	mu sync.Mutex

	// Identity
	ID    int
	Peers []int

	// Current Raft state
	State State

	// Persistent state
	CurrentTerm int
	VotedFor    int
	Log         []LogEntry

	// Volatile state
	CommitIndex int
	LastApplied int

	// Leader state
	NextIndex  []int
	MatchIndex []int

	// Time of the most recent valid leader heartbeat.
	lastHeartbeat time.Time
}

func NewRaft(id int, peers []int) *Raft {
	return &Raft{
		ID:            id,
		Peers:         peers,
		State:         Follower,
		CurrentTerm:   0,
		VotedFor:      -1,
		Log:           make([]LogEntry, 0),
		CommitIndex:   0,
		LastApplied:   0,
		NextIndex:     make([]int, len(peers)),
		MatchIndex:    make([]int, len(peers)),
		lastHeartbeat: time.Now(),
	}
}

func (r *Raft) StartElection() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.State = Candidate
	r.CurrentTerm++
	r.VotedFor = r.ID

	// Starting an election does not count as a valid leader heartbeat.
	return r.CurrentTerm
}

func (r *Raft) BecomeLeader() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.State = Leader

	for i := range r.NextIndex {
		r.NextIndex[i] = len(r.Log)
		r.MatchIndex[i] = 0
	}
}

func (r *Raft) BecomeFollower(term int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.State = Follower
	r.CurrentTerm = term
	r.VotedFor = -1
	r.lastHeartbeat = time.Now()
}

func (r *Raft) ApplyCommitted(
	parkingLot *parkinglogic.ParkingLot,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for r.LastApplied <= r.CommitIndex {
		if r.LastApplied >= len(r.Log) {
			break
		}

		entry := r.Log[r.LastApplied]

		if err := ApplyCommand(
			parkingLot,
			entry.Command,
		); err != nil {
			return err
		}

		r.LastApplied++
	}

	return nil
}

func (r *Raft) GetState() State {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.State
}

func (r *Raft) GetCurrentTerm() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.CurrentTerm
}

func (r *Raft) GetLogInfo() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lastIndex := len(r.Log) - 1
	lastTerm := 0

	if lastIndex >= 0 {
		lastTerm = r.Log[lastIndex].Term
	}

	return lastIndex, lastTerm
}

// GetLogEntries returns a safe copy of the Raft log.
func (r *Raft) GetLogEntries() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries := make([]LogEntry, len(r.Log))
	copy(entries, r.Log)

	return entries
}

// GetCommitIndex returns the current committed log index.
func (r *Raft) GetCommitIndex() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.CommitIndex
}

// ResetElectionTimer records that this node has recently
// received a valid heartbeat from the leader.
func (r *Raft) ResetElectionTimer() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastHeartbeat = time.Now()
}

// LastHeartbeat returns the time when this node last
// received a valid leader heartbeat.
func (r *Raft) LastHeartbeat() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lastHeartbeat
}
