package main

import (
	"log"
	"net"

	parkinggrpc "smart-parking-kv/grpc"
	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
)

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()

	parkingServer := parkinggrpc.NewParkingServer()

	pb.RegisterParkingServiceServer(server, parkingServer)

	log.Println("Parking gRPC server started on port 50051")

	if err := server.Serve(listener); err != nil {
		log.Fatalf("gRPC server stopped: %v", err)
	}
}
