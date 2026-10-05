package raft

import "time"

type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (r *Raft) AppendEntries(
	args *AppendEntriesArgs,
	reply *AppendEntriesReply,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reply.Term = r.CurrentTerm
	reply.Success = false

	// Reject stale leaders.
	if args.Term < r.CurrentTerm {
		return
	}

	// A newer term means this node must become a follower.
	if args.Term > r.CurrentTerm {
		r.CurrentTerm = args.Term
		r.State = Follower
		r.VotedFor = -1
	}

	// A valid AppendEntries RPC means the leader is alive.
	r.State = Follower
	r.lastHeartbeat = time.Now()
	reply.Term = r.CurrentTerm

	// Verify the leader's previous log index if present.
	if args.PrevLogIndex >= 0 {
		if args.PrevLogIndex >= len(r.Log) {
			return
		}

		if r.Log[args.PrevLogIndex].Term != args.PrevLogTerm {
			return
		}
	}

	// Append entries while preserving the shared prefix and truncating
	// conflicting suffixes if the leader has more up-to-date history.
	for i, entry := range args.Entries {
		index := args.PrevLogIndex + 1 + i

		if index < len(r.Log) {
			if r.Log[index].Term != entry.Term {
				r.Log = r.Log[:index]
			} else {
				continue
			}
		}

		// Append the remaining entries from the first conflicting or new index.
		if index >= len(r.Log) {
			r.Log = append(r.Log, args.Entries[i:]...)
			break
		}
	}

	// Update the commit index based on the leader's view of the log.
	if args.LeaderCommit > r.CommitIndex {
		lastIndex := len(r.Log) - 1

		if args.LeaderCommit < lastIndex {
			r.CommitIndex = args.LeaderCommit
		} else {
			r.CommitIndex = lastIndex
		}
	}

	reply.Success = true
}
