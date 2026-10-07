package main

import (
	"errors"
	"net"
	"testing"
	"time"

	parkinggrpc "smart-parking-kv/grpc"
	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
)

func TestRunStressAgainstLocalParkingServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen returned an error: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterParkingServiceServer(server, parkinggrpc.NewParkingServer())
	go func() {
		if err := server.Serve(listener); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("gRPC server stopped unexpectedly: %v", err)
		}
	}()
	defer server.Stop()

	result, err := runStress(listener.Addr().String(), 8, 50)
	if err != nil {
		t.Fatalf("runStress returned an error: %v", err)
	}

	want := int64(8 * 50 * 2)
	if result.requests != want {
		t.Errorf("request count = %d, want %d", result.requests, want)
	}
	if result.successful != want {
		t.Errorf("successful request count = %d, want %d", result.successful, want)
	}
	if result.failed != 0 {
		t.Errorf("failed request count = %d, want 0", result.failed)
	}
	if result.duration <= 0 {
		t.Fatal("stress run duration must be positive")
	}
	throughput := float64(result.successful) / result.duration.Seconds()
	averageLatency := result.totalRequestTime / time.Duration(result.requests)
	t.Logf(
		"clients=%d operations_per_client=%d requests=%d failed=%d duration=%s throughput=%.2f requests/second mean_latency=%s",
		result.clients,
		result.operations,
		result.successful,
		result.failed,
		result.duration,
		throughput,
		averageLatency,
	)
}

func TestRunStressRejectsInvalidConfiguration(t *testing.T) {
	_, err := runStress("", 1, 1)
	if err == nil {
		t.Fatal("runStress accepted an empty server address")
	}

	_, err = runStress("localhost:50051", 0, 1)
	if err == nil {
		t.Fatal("runStress accepted a non-positive client count")
	}

	_, err = runStress("localhost:50051", 1, 0)
	if err == nil {
		t.Fatal("runStress accepted a non-positive operation count")
	}
}
