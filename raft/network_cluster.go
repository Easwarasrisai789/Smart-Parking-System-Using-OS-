package raft

import (
	"fmt"
	"sync"
)

type NetworkCluster struct {
	mu sync.Mutex

	nodes map[int]*Raft

	addresses map[int]string

	clients map[int]map[int]*RaftClient
}

func NewNetworkCluster(
	nodes map[int]*Raft,
	addresses map[int]string,
) *NetworkCluster {
	return &NetworkCluster{
		nodes:     nodes,
		addresses: addresses,
		clients:   make(map[int]map[int]*RaftClient),
	}
}

func (c *NetworkCluster) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for nodeID := range c.nodes {
		c.clients[nodeID] = make(map[int]*RaftClient)

		for peerID, address := range c.addresses {
			if peerID == nodeID {
				continue
			}

			client, err := NewRaftClient(address)
			if err != nil {
				return fmt.Errorf(
					"node %d failed to connect to node %d: %w",
					nodeID,
					peerID,
					err,
				)
			}

			c.clients[nodeID][peerID] = client
		}
	}

	return nil
}

func (c *NetworkCluster) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, peerClients := range c.clients {
		for _, client := range peerClients {
			_ = client.Close()
		}
	}

	c.clients = make(map[int]map[int]*RaftClient)
}

func (c *NetworkCluster) RequestVote(
	fromID int,
	toID int,
	args RequestVoteArgs,
) (RequestVoteReply, error) {

	c.mu.Lock()

	peerClients, ok := c.clients[fromID]
	if !ok {
		c.mu.Unlock()

		return RequestVoteReply{}, fmt.Errorf(
			"node %d has no network clients",
			fromID,
		)
	}

	client, ok := peerClients[toID]

	c.mu.Unlock()

	if !ok {
		return RequestVoteReply{}, fmt.Errorf(
			"node %d has no connection to node %d",
			fromID,
			toID,
		)
	}

	return client.RequestVote(args)
}

func (c *NetworkCluster) AppendEntries(
	fromID int,
	toID int,
	args AppendEntriesArgs,
) (AppendEntriesReply, error) {

	c.mu.Lock()

	peerClients, ok := c.clients[fromID]
	if !ok {
		c.mu.Unlock()

		return AppendEntriesReply{}, fmt.Errorf(
			"node %d has no network clients",
			fromID,
		)
	}

	client, ok := peerClients[toID]

	c.mu.Unlock()

	if !ok {
		return AppendEntriesReply{}, fmt.Errorf(
			"node %d has no connection to node %d",
			fromID,
			toID,
		)
	}

	return client.AppendEntries(args)
}
