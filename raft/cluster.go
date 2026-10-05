package raft

import (
	"strings"
	"sync"

	parkinglogic "smart-parking-kv/parking"
)

type Cluster struct {
	mu sync.Mutex

	nodes map[int]*Raft
	alive map[int]bool

	// Each Raft node has its own state machine.
	stateMachines map[int]*parkinglogic.ParkingLot
}

func NewCluster(size int) *Cluster {
	cluster := &Cluster{
		nodes:         make(map[int]*Raft),
		alive:         make(map[int]bool),
		stateMachines: make(map[int]*parkinglogic.ParkingLot),
	}

	peers := make([]int, size)

	for i := 0; i < size; i++ {
		peers[i] = i
	}

	for i := 0; i < size; i++ {
		cluster.nodes[i] = NewRaft(i, peers)
		cluster.alive[i] = true

		cluster.stateMachines[i] = parkinglogic.NewParkingLot()
	}

	return cluster
}

func (c *Cluster) Node(id int) *Raft {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.nodes[id]
}

func (c *Cluster) StateMachine(id int) *parkinglogic.ParkingLot {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.stateMachines[id]
}

func (c *Cluster) FailNode(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.alive[id] = false
}

func (c *Cluster) RecoverNode(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.alive[id] = true
}

func (c *Cluster) IsAlive(id int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.alive[id]
}

func (c *Cluster) RequestVotes(candidateID int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.alive[candidateID] {
		return 0
	}

	candidate := c.nodes[candidateID]

	candidate.StartElection()

	votes := 1

	lastLogIndex := len(candidate.Log) - 1
	lastLogTerm := 0

	if lastLogIndex >= 0 {
		lastLogTerm = candidate.Log[lastLogIndex].Term
	}

	for id, node := range c.nodes {
		if id == candidateID {
			continue
		}

		if !c.alive[id] {
			continue
		}

		args := &RequestVoteArgs{
			Term:         candidate.CurrentTerm,
			CandidateID:  candidateID,
			LastLogIndex: lastLogIndex,
			LastLogTerm:  lastLogTerm,
		}

		reply := &RequestVoteReply{}

		node.RequestVote(args, reply)

		if reply.VoteGranted {
			votes++
		}
	}

	if votes > len(c.nodes)/2 {
		candidate.BecomeLeader()
	}

	return votes
}

func (c *Cluster) ReplicateCommand(
	leaderID int,
	command string,
	parkingLot *parkinglogic.ParkingLot,
) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.alive[leaderID] {
		return false
	}

	leader := c.nodes[leaderID]

	if leader.State != Leader {
		return false
	}

	entry := LogEntry{
		Term:    leader.CurrentTerm,
		Command: command,
	}

	leader.Log = append(leader.Log, entry)

	successes := 1

	for id, follower := range c.nodes {
		if id == leaderID {
			continue
		}

		if !c.alive[id] {
			continue
		}

		args := &AppendEntriesArgs{
			Term:         leader.CurrentTerm,
			LeaderID:     leaderID,
			PrevLogIndex: len(leader.Log) - 2,
			PrevLogTerm:  0,
			Entries:      []LogEntry{entry},
			LeaderCommit: leader.CommitIndex,
		}

		if args.PrevLogIndex >= 0 {
			args.PrevLogTerm = leader.Log[args.PrevLogIndex].Term
		}

		reply := &AppendEntriesReply{}

		follower.AppendEntries(args, reply)

		if reply.Success {
			successes++
		}
	}

	// Majority achieved.
	if successes > len(c.nodes)/2 {
		leader.CommitIndex = len(leader.Log) - 1

		// Update commit index for replicated followers.
		for id, follower := range c.nodes {
			if id == leaderID {
				continue
			}

			if !c.alive[id] {
				continue
			}

			if follower.CommitIndex < leader.CommitIndex {
				follower.CommitIndex = leader.CommitIndex
			}
		}

		// Apply committed commands on leader.
		if err := leader.ApplyCommitted(parkingLot); err != nil {
			return false
		}

		// Apply committed commands on every follower.
		for id, follower := range c.nodes {
			if id == leaderID {
				continue
			}

			if !c.alive[id] {
				continue
			}

			followerLot := c.stateMachines[id]

			if err := follower.ApplyCommitted(followerLot); err != nil {
				return false
			}
		}

		return true
	}

	return false
}

func (c *Cluster) CatchUpNode(nodeID int, leaderID int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.alive[nodeID] {
		return false
	}

	if !c.alive[leaderID] {
		return false
	}

	leader := c.nodes[leaderID]
	follower := c.nodes[nodeID]

	if leader.State != Leader {
		return false
	}

	// Copy leader log.
	follower.Log = make([]LogEntry, len(leader.Log))
	copy(follower.Log, leader.Log)

	follower.CurrentTerm = leader.CurrentTerm
	follower.CommitIndex = leader.CommitIndex

	// Apply recovered commands to follower state machine.
	followerLot := c.stateMachines[nodeID]

	if err := follower.ApplyCommitted(followerLot); err != nil {
		return false
	}

	return true
}

func ApplyCommand(
	parkingLot *parkinglogic.ParkingLot,
	command string,
) error {
	parts := strings.Fields(command)

	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "ENTER":
		if len(parts) != 3 {
			return nil
		}

		return parkingLot.Enter(
			parts[1],
			parts[2],
		)

	case "EXIT":
		if len(parts) != 3 {
			return nil
		}

		return parkingLot.Exit(
			parts[1],
			parts[2],
		)

	default:
		return nil
	}
}
