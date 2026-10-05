package raft

import "time"

type RequestVoteArgs struct {
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (r *Raft) RequestVote(
	args *RequestVoteArgs,
	reply *RequestVoteReply,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reply.Term = r.CurrentTerm
	reply.VoteGranted = false

	// Candidate has an older term.
	if args.Term < r.CurrentTerm {
		return
	}

	// Candidate has a newer term.
	if args.Term > r.CurrentTerm {
		r.CurrentTerm = args.Term
		r.State = Follower
		r.VotedFor = -1
	}

	reply.Term = r.CurrentTerm

	// Already voted for another candidate.
	if r.VotedFor != -1 && r.VotedFor != args.CandidateID {
		return
	}

	// Check whether candidate's log is at least as up-to-date.
	lastLogIndex := len(r.Log) - 1

	lastLogTerm := 0

	if lastLogIndex >= 0 {
		lastLogTerm = r.Log[lastLogIndex].Term
	}

	if args.LastLogTerm < lastLogTerm {
		return
	}

	if args.LastLogTerm == lastLogTerm &&
		args.LastLogIndex < lastLogIndex {
		return
	}

	r.VotedFor = args.CandidateID
	r.lastHeartbeat = time.Now()
	reply.VoteGranted = true
}
