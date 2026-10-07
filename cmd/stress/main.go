package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	pb "smart-parking-kv/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type stressResult struct {
	clients          int
	operations       int
	requests         int64
	successful       int64
	failed           int64
	totalRequestTime time.Duration
	duration         time.Duration
}

type workerResult struct {
	requests         int64
	successful       int64
	failed           int64
	totalRequestTime time.Duration
}

func main() {
	address := flag.String("address", "localhost:50051", "parking gRPC server address")
	clients := flag.Int("clients", 100, "number of concurrent clients")
	operations := flag.Int("operations", 1000, "operations per client")
	flag.Parse()

	result, err := runStress(*address, *clients, *operations)
	if err != nil {
		log.Fatalf("stress test failed: %v", err)
	}

	requestsPerSecond := float64(result.successful) / result.duration.Seconds()
	averageLatency := time.Duration(0)
	if result.requests > 0 {
		averageLatency = result.totalRequestTime / time.Duration(result.requests)
	}

	fmt.Printf("Server address: %s\n", *address)
	fmt.Printf("Clients: %d\n", result.clients)
	fmt.Printf("Operations per client: %d\n", result.operations)
	fmt.Printf("Requests attempted: %d\n", result.requests)
	fmt.Printf("Successful requests: %d\n", result.successful)
	fmt.Printf("Failed requests: %d\n", result.failed)
	fmt.Printf("Duration: %s\n", result.duration)
	fmt.Printf("Successful throughput: %.2f requests/second\n", requestsPerSecond)
	fmt.Printf("Average request latency: %s\n", averageLatency)
}

func runStress(address string, clients, operations int) (stressResult, error) {
	if address == "" {
		return stressResult{}, errors.New("server address is required")
	}
	if clients < 1 {
		return stressResult{}, errors.New("client count must be positive")
	}
	if operations < 1 {
		return stressResult{}, errors.New("operation count must be positive")
	}

	started := time.Now()
	results := make(chan workerResult, clients)
	var workers sync.WaitGroup
	workers.Add(clients)

	for clientID := 0; clientID < clients; clientID++ {
		go func(clientID int) {
			defer workers.Done()
			results <- runWorker(address, clientID, operations)
		}(clientID)
	}

	workers.Wait()
	close(results)

	result := stressResult{
		clients:    clients,
		operations: operations,
		duration:   time.Since(started),
	}

	for worker := range results {
		result.requests += worker.requests
		result.successful += worker.successful
		result.failed += worker.failed
		result.totalRequestTime += worker.totalRequestTime
	}

	if result.requests == 0 {
		return result, errors.New("stress run completed without issuing requests")
	}

	return result, nil
}

func runWorker(address string, clientID, operations int) workerResult {
	connection, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return workerResult{
			requests: int64(operations),
			failed:   int64(operations),
		}
	}
	defer connection.Close()

	client := pb.NewParkingServiceClient(connection)
	result := workerResult{}

	for operation := 0; operation < operations; operation++ {
		slotID := fmt.Sprintf("stress-%d-%d", clientID, operation)
		vehicleID := fmt.Sprintf("vehicle-%d-%d", clientID, operation)

		response, requestErr, latency := enter(client, slotID, vehicleID)
		result.requests++
		result.totalRequestTime += latency
		if requestErr != nil || response == nil || !response.GetSuccess() {
			result.failed++
			continue
		}
		result.successful++

		exitResponse, exitErr, exitLatency := exit(client, slotID, vehicleID)
		result.requests++
		result.totalRequestTime += exitLatency
		if exitErr != nil || exitResponse == nil || !exitResponse.GetSuccess() {
			result.failed++
			continue
		}
		result.successful++
	}

	return result
}

func enter(
	client pb.ParkingServiceClient,
	slotID, vehicleID string,
) (*pb.ParkingResponse, error, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started := time.Now()
	response, err := client.Enter(ctx, &pb.EnterRequest{
		SlotId:    slotID,
		VehicleId: vehicleID,
	})
	return response, err, time.Since(started)
}

func exit(
	client pb.ParkingServiceClient,
	slotID, vehicleID string,
) (*pb.ParkingResponse, error, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started := time.Now()
	response, err := client.Exit(ctx, &pb.ExitRequest{
		SlotId:    slotID,
		VehicleId: vehicleID,
	})
	return response, err, time.Since(started)
}
