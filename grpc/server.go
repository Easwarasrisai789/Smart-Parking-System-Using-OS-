package grpc

import (
	"context"
	"fmt"

	parkinglogic "smart-parking-kv/parking"
	pb "smart-parking-kv/proto"
)

type ParkingServer struct {
	pb.UnimplementedParkingServiceServer

	parkingLot *parkinglogic.ParkingLot
}

func NewParkingServer() *ParkingServer {
	return &ParkingServer{
		parkingLot: parkinglogic.NewParkingLot(),
	}
}

func (s *ParkingServer) Enter(
	ctx context.Context,
	req *pb.EnterRequest,
) (*pb.ParkingResponse, error) {

	err := s.parkingLot.Enter(
		req.GetSlotId(),
		req.GetVehicleId(),
	)

	if err != nil {
		return &pb.ParkingResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ParkingResponse{
		Success: true,
		Message: fmt.Sprintf(
			"Vehicle %s entered slot %s",
			req.GetVehicleId(),
			req.GetSlotId(),
		),
	}, nil
}

func (s *ParkingServer) Exit(
	ctx context.Context,
	req *pb.ExitRequest,
) (*pb.ParkingResponse, error) {

	err := s.parkingLot.Exit(
		req.GetSlotId(),
		req.GetVehicleId(),
	)

	if err != nil {
		return &pb.ParkingResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.ParkingResponse{
		Success: true,
		Message: fmt.Sprintf(
			"Vehicle %s exited slot %s",
			req.GetVehicleId(),
			req.GetSlotId(),
		),
	}, nil
}

func (s *ParkingServer) Status(
	ctx context.Context,
	req *pb.StatusRequest,
) (*pb.ParkingResponse, error) {

	status := s.parkingLot.Status(req.GetSlotId())

	return &pb.ParkingResponse{
		Success: true,
		Message: fmt.Sprintf(
			"Slot %s is %s",
			req.GetSlotId(),
			status,
		),
	}, nil
}
