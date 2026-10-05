package grpc

import (
	"context"
	"testing"

	pb "smart-parking-kv/proto"
)

func TestParkingServerEnterAndStatus(t *testing.T) {
	server := NewParkingServer()
	ctx := context.Background()

	enterResponse, err := server.Enter(ctx, &pb.EnterRequest{
		SlotId:    "001",
		VehicleId: "AP39AB1234",
	})

	if err != nil {
		t.Fatalf("Enter failed: %v", err)
	}

	if !enterResponse.GetSuccess() {
		t.Fatalf("Enter should succeed: %s", enterResponse.GetMessage())
	}

	statusResponse, err := server.Status(ctx, &pb.StatusRequest{
		SlotId: "001",
	})

	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if !statusResponse.GetSuccess() {
		t.Fatalf("Status should succeed: %s", statusResponse.GetMessage())
	}

	expected := "Slot 001 is OCCUPIED"

	if statusResponse.GetMessage() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			statusResponse.GetMessage(),
		)
	}
}

func TestParkingServerExit(t *testing.T) {
	server := NewParkingServer()
	ctx := context.Background()

	_, err := server.Enter(ctx, &pb.EnterRequest{
		SlotId:    "002",
		VehicleId: "AP39AB5678",
	})

	if err != nil {
		t.Fatalf("Enter failed: %v", err)
	}

	exitResponse, err := server.Exit(ctx, &pb.ExitRequest{
		SlotId:    "002",
		VehicleId: "AP39AB5678",
	})

	if err != nil {
		t.Fatalf("Exit failed: %v", err)
	}

	if !exitResponse.GetSuccess() {
		t.Fatalf("Exit should succeed: %s", exitResponse.GetMessage())
	}

	statusResponse, err := server.Status(ctx, &pb.StatusRequest{
		SlotId: "002",
	})

	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	expected := "Slot 002 is AVAILABLE"

	if statusResponse.GetMessage() != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			statusResponse.GetMessage(),
		)
	}
}

func TestParkingServerOccupiedSlot(t *testing.T) {
	server := NewParkingServer()
	ctx := context.Background()

	_, err := server.Enter(ctx, &pb.EnterRequest{
		SlotId:    "003",
		VehicleId: "AP39AB1111",
	})

	if err != nil {
		t.Fatalf("first Enter failed: %v", err)
	}

	response, err := server.Enter(ctx, &pb.EnterRequest{
		SlotId:    "003",
		VehicleId: "AP39AB2222",
	})

	if err != nil {
		t.Fatalf("second Enter returned unexpected error: %v", err)
	}

	if response.GetSuccess() {
		t.Fatal("second Enter should fail because slot is occupied")
	}
}
