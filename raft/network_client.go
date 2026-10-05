package raft

import (
	"context"
	"fmt"
	"time"

	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RaftClient struct {
	conn   *grpc.ClientConn
	client pb.RaftServiceClient
}

func NewRaftClient(address string) (*RaftClient, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create connection to %s: %w",
			address,
			err,
		)
	}

	return &RaftClient{
		conn:   conn,
		client: pb.NewRaftServiceClient(conn),
	}, nil
}

func (c *RaftClient) RequestVote(
	args RequestVoteArgs,
) (RequestVoteReply, error) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	response, err := c.client.RequestVote(
		ctx,
		&pb.RequestVoteRequest{
			Term:         int32(args.Term),
			CandidateId:  int32(args.CandidateID),
			LastLogIndex: int32(args.LastLogIndex),
			LastLogTerm:  int32(args.LastLogTerm),
		},
	)

	if err != nil {
		return RequestVoteReply{}, err
	}

	return RequestVoteReply{
		Term:        int(response.Term),
		VoteGranted: response.VoteGranted,
	}, nil
}

func (c *RaftClient) AppendEntries(
	args AppendEntriesArgs,
) (AppendEntriesReply, error) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	entries := make([]*pb.RaftLogEntry, len(args.Entries))

	for i, entry := range args.Entries {
		entries[i] = &pb.RaftLogEntry{
			Term:    int32(entry.Term),
			Command: entry.Command,
		}
	}

	response, err := c.client.AppendEntries(
		ctx,
		&pb.AppendEntriesRequest{
			Term:         int32(args.Term),
			LeaderId:     int32(args.LeaderID),
			PrevLogIndex: int32(args.PrevLogIndex),
			PrevLogTerm:  int32(args.PrevLogTerm),
			Entries:      entries,
			LeaderCommit: int32(args.LeaderCommit),
		},
	)

	if err != nil {
		return AppendEntriesReply{}, err
	}

	return AppendEntriesReply{
		Term:    int(response.Term),
		Success: response.Success,
	}, nil
}

func (c *RaftClient) Close() error {
	return c.conn.Close()
}
