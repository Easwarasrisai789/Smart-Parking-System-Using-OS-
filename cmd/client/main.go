package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  client enter <slot> <vehicle>")
		fmt.Println("  client exit <slot> <vehicle>")
		fmt.Println("  client status <slot>")
		return
	}

	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewParkingServiceClient(conn)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	switch os.Args[1] {

	case "enter":
		if len(os.Args) != 4 {
			fmt.Println("Usage: client enter <slot> <vehicle>")
			return
		}

		response, err := client.Enter(ctx, &pb.EnterRequest{
			SlotId:    os.Args[2],
			VehicleId: os.Args[3],
		})

		if err != nil {
			log.Fatalf("ENTER failed: %v", err)
		}

		fmt.Println(response.GetMessage())

	case "exit":
		if len(os.Args) != 4 {
			fmt.Println("Usage: client exit <slot> <vehicle>")
			return
		}

		response, err := client.Exit(ctx, &pb.ExitRequest{
			SlotId:    os.Args[2],
			VehicleId: os.Args[3],
		})

		if err != nil {
			log.Fatalf("EXIT failed: %v", err)
		}

		fmt.Println(response.GetMessage())

	case "status":
		if len(os.Args) != 3 {
			fmt.Println("Usage: client status <slot>")
			return
		}

		response, err := client.Status(ctx, &pb.StatusRequest{
			SlotId: os.Args[2],
		})

		if err != nil {
			log.Fatalf("STATUS failed: %v", err)
		}

		fmt.Println(response.GetMessage())

	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}
