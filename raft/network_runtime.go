package raft

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

type PeerConnection struct {
	Address string
	Client  *RaftClient
}

type NetworkRuntime struct {
	node *Raft

	peers map[int]*PeerConnection

	mu sync.Mutex

	stopCh chan struct{}
	wg     sync.WaitGroup

	running bool
}

// Existing constructor.
// Keeps all existing tests compatible.
func NewNetworkRuntime(
	node *Raft,
	clients map[int]*RaftClient,
) *NetworkRuntime {
	peers := make(map[int]*PeerConnection)

	for peerID, client := range clients {
		peers[peerID] = &PeerConnection{
			Client: client,
		}
	}

	return &NetworkRuntime{
		node:   node,
		peers:  peers,
		stopCh: make(chan struct{}),
	}
}

// New constructor for Docker / dynamic peer connections.
func NewNetworkRuntimeWithAddresses(
	node *Raft,
	addresses map[int]string,
) *NetworkRuntime {
	peers := make(map[int]*PeerConnection)

	for peerID, address := range addresses {
		peers[peerID] = &PeerConnection{
			Address: address,
		}
	}

	return &NetworkRuntime{
		node:   node,
		peers:  peers,
		stopCh: make(chan struct{}),
	}
}

func (r *NetworkRuntime) Start() {
	r.mu.Lock()

	if r.running {
		r.mu.Unlock()
		return
	}

	r.running = true
	r.mu.Unlock()

	r.wg.Add(1)
	go r.electionLoop()

	r.wg.Add(1)
	go r.replicationLoop()

	log.Printf(
		"Network runtime started for node %d",
		r.node.ID,
	)
}

func (r *NetworkRuntime) Stop() {
	r.mu.Lock()

	if !r.running {
		r.mu.Unlock()
		return
	}

	r.running = false
	close(r.stopCh)

	r.mu.Unlock()

	r.wg.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, peer := range r.peers {
		if peer.Client != nil {
			_ = peer.Client.Close()
			peer.Client = nil
		}
	}

	log.Printf(
		"Network runtime stopped for node %d",
		r.node.ID,
	)
}

// ----------------------------------------------------
// Peer connection handling
// ----------------------------------------------------

func (r *NetworkRuntime) getPeerClient(peerID int) *RaftClient {
	r.mu.Lock()
	defer r.mu.Unlock()

	peer, ok := r.peers[peerID]
	if !ok {
		return nil
	}

	return peer.Client
}

func (r *NetworkRuntime) connectPeer(peerID int) *RaftClient {
	r.mu.Lock()

	peer, ok := r.peers[peerID]
	if !ok || peer.Address == "" {
		r.mu.Unlock()
		return nil
	}

	if peer.Client != nil {
		client := peer.Client
		r.mu.Unlock()
		return client
	}

	address := peer.Address
	r.mu.Unlock()

	client, err := NewRaftClient(address)
	if err != nil {
		log.Printf(
			"NODE %d FAILED TO CONNECT TO PEER %d ADDRESS %s ERROR: %v",
			r.node.ID,
			peerID,
			address,
			err,
		)
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	peer, ok = r.peers[peerID]
	if !ok {
		_ = client.Close()
		return nil
	}

	if peer.Client != nil {
		_ = client.Close()
		return peer.Client
	}

	peer.Client = client

	log.Printf(
		"NODE %d CONNECTED TO PEER %d AT %s",
		r.node.ID,
		peerID,
		address,
	)

	return client
}

func (r *NetworkRuntime) reconnectPeer(peerID int) {
	r.mu.Lock()

	peer, ok := r.peers[peerID]
	if !ok {
		r.mu.Unlock()
		return
	}

	oldClient := peer.Client
	peer.Client = nil

	r.mu.Unlock()

	if oldClient != nil {
		_ = oldClient.Close()
	}

	log.Printf(
		"NODE %d WILL RETRY CONNECTION TO PEER %d",
		r.node.ID,
		peerID,
	)
}

// ----------------------------------------------------
// Election
// ----------------------------------------------------

func (r *NetworkRuntime) electionLoop() {
	defer r.wg.Done()

	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	timeout := func() time.Duration {
		return time.Duration(
			700+random.Intn(500),
		) * time.Millisecond
	}

	lastElection := time.Now()
	electionTimeout := timeout()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return

		case <-ticker.C:
			switch r.node.GetState() {
			case Leader:
				lastElection = time.Now()
				electionTimeout = timeout()
				continue
			case Candidate:
				// A candidate should not keep extending its election timer based on
				// stale leader heartbeats. It will retry after its current timeout.
			default:
				lastHeartbeat := r.node.LastHeartbeat()

				if lastHeartbeat.After(lastElection) {
					lastElection = lastHeartbeat
					electionTimeout = timeout()
					continue
				}
			}

			if time.Since(lastElection) < electionTimeout {
				continue
			}

			log.Printf(
				"NODE %d ELECTION TIMEOUT",
				r.node.ID,
			)

			r.runElection()

			lastElection = time.Now()
			electionTimeout = timeout()
		}
	}
}

func (r *NetworkRuntime) runElection() {
	term := r.node.StartElection()

	log.Printf(
		"NODE %d STARTING ELECTION TERM %d",
		r.node.ID,
		term,
	)

	votes := 1

	lastIndex, lastTerm := r.node.GetLogInfo()

	for _, peerID := range r.node.Peers {
		if peerID == r.node.ID {
			continue
		}

		client := r.getPeerClient(peerID)

		if client == nil {
			client = r.connectPeer(peerID)
		}

		if client == nil {
			log.Printf(
				"NODE %d VOTE REQUEST FAILED TO PEER %d: NO CONNECTION",
				r.node.ID,
				peerID,
			)
			continue
		}

		reply, err := client.RequestVote(
			RequestVoteArgs{
				Term:         term,
				CandidateID:  r.node.ID,
				LastLogIndex: lastIndex,
				LastLogTerm:  lastTerm,
			},
		)

		if err != nil {
			log.Printf(
				"NODE %d VOTE REQUEST FAILED TO PEER %d ERROR: %v",
				r.node.ID,
				peerID,
				err,
			)

			r.reconnectPeer(peerID)
			continue
		}

		if reply.Term > term {
			r.node.BecomeFollower(reply.Term)

			log.Printf(
				"NODE %d DISCOVERED HIGHER TERM FROM PEER %d TERM: %d",
				r.node.ID,
				peerID,
				reply.Term,
			)

			return
		}

		if reply.VoteGranted {
			votes++

			log.Printf(
				"NODE %d RECEIVED VOTE FROM %d",
				r.node.ID,
				peerID,
			)
		}
	}

	if votes > len(r.node.Peers)/2 {
		r.node.BecomeLeader()

		log.Printf(
			"NODE %d BECAME LEADER TERM %d VOTES %d",
			r.node.ID,
			term,
			votes,
		)
	} else {
		log.Printf(
			"NODE %d ELECTION FAILED TERM %d VOTES %d",
			r.node.ID,
			term,
			votes,
		)
	}
}

// ----------------------------------------------------
// Replication / heartbeat
// ----------------------------------------------------

func (r *NetworkRuntime) replicationLoop() {
	defer r.wg.Done()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return

		case <-ticker.C:
			if r.node.GetState() != Leader {
				continue
			}

			r.replicate()
		}
	}
}

func (r *NetworkRuntime) replicate() {
	lastIndex, _ := r.node.GetLogInfo()

	for _, peerID := range r.node.Peers {
		if peerID == r.node.ID {
			continue
		}

		client := r.getPeerClient(peerID)

		if client == nil {
			client = r.connectPeer(peerID)
		}

		if client == nil {
			continue
		}

		entries := r.node.GetLogEntries()

		prevLogIndex := -1
		prevLogTerm := 0

		if len(entries) > 0 {
			prevLogIndex = -1
			prevLogTerm = 0
		}

		args := AppendEntriesArgs{
			Term:         r.node.GetCurrentTerm(),
			LeaderID:     r.node.ID,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      entries,
			LeaderCommit: r.node.GetCommitIndex(),
		}

		reply, err := client.AppendEntries(args)

		if err != nil {
			log.Printf(
				"NODE %d APPEND ENTRIES FAILED TO PEER %d ERROR: %v",
				r.node.ID,
				peerID,
				err,
			)

			r.reconnectPeer(peerID)
			continue
		}

		if reply.Term > args.Term {
			r.node.BecomeFollower(reply.Term)

			log.Printf(
				"NODE %d DISCOVERED HIGHER TERM FROM PEER %d TERM: %d",
				r.node.ID,
				peerID,
				reply.Term,
			)

			return
		}

		if reply.Success {
			log.Printf(
				"NODE %d HEARTBEAT/REPLICATION SUCCESS PEER %d LOG_INDEX %d",
				r.node.ID,
				peerID,
				lastIndex,
			)
		}
	}
}
