package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"smart-parking-kv/raft"
)

func main() {
	nodeID := getNodeID()

	peers := []int{0, 1, 2}

	node := raft.NewRaft(nodeID, peers)
	metrics, err := raft.NewMetrics(node)
	if err != nil {
		log.Fatalf("failed to initialize Raft metrics: %v", err)
	}

	log.Printf("====================================")
	log.Printf("STARTING RAFT NODE %d", nodeID)
	log.Printf("====================================")

	address := os.Getenv("RAFT_ADDRESS")

	if address == "" {
		address = ":6000"
	}

	server, _, err := raft.StartNetworkServerWithMetrics(
		node,
		address,
		metrics,
	)

	if err != nil {
		log.Fatalf(
			"failed to start Raft network server: %v",
			err,
		)
	}

	defer server.Stop()

	log.Printf(
		"NODE %d LISTENING ON %s",
		nodeID,
		address,
	)

	addressList := os.Getenv("NODE_ADDRESSES")

	if addressList == "" {
		log.Fatal("NODE_ADDRESSES is not configured")
	}

	rawAddresses := strings.Split(
		addressList,
		",",
	)

	if len(rawAddresses) != len(peers) {
		log.Fatalf(
			"NODE_ADDRESSES must contain %d addresses, got %d",
			len(peers),
			len(rawAddresses),
		)
	}

	peerAddresses := make(map[int]string)

	for _, peerID := range peers {
		if peerID == nodeID {
			continue
		}

		peerAddress := strings.TrimSpace(
			rawAddresses[peerID],
		)

		if peerAddress == "" {
			log.Fatalf(
				"missing address for peer %d",
				peerID,
			)
		}

		peerAddresses[peerID] = peerAddress

		log.Printf(
			"NODE %d PEER %d ADDRESS %s",
			nodeID,
			peerID,
			peerAddress,
		)
	}

	runtime := raft.NewNetworkRuntimeWithAddresses(
		node,
		peerAddresses,
	)

	runtime.Start()

	defer runtime.Stop()

	log.Printf(
		"NODE %d AUTOMATIC RAFT RUNTIME STARTED",
		nodeID,
	)

	metricsAddress := os.Getenv("METRICS_ADDRESS")
	if metricsAddress == "" {
		metricsAddress = ":8080"
	}

	metricsListener, err := net.Listen("tcp", metricsAddress)
	if err != nil {
		log.Fatalf("failed to start metrics listener: %v", err)
	}

	metricsServer := &http.Server{
		Handler: metrics.Handler(),
	}

	defer metricsServer.Close()

	go func() {
		if err := metricsServer.Serve(metricsListener); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("metrics HTTP server stopped: %v", err)
		}
	}()

	log.Printf(
		"NODE %d METRICS LISTENING ON %s",
		nodeID,
		metricsAddress,
	)

	select {}
}

func getNodeID() int {
	value := strings.TrimSpace(os.Getenv("NODE_ID"))

	if value != "" {
		id, err := strconv.Atoi(value)

		if err == nil {
			return id
		}
	}

	hostname, err := os.Hostname()
	if err == nil {
		if id, ok := nodeIDFromHostname(hostname); ok {
			return id
		}
	}

	log.Fatal("NODE_ID is not configured and could not be derived from hostname")

	return -1
}

func nodeIDFromHostname(hostname string) (int, bool) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return 0, false
	}

	if dotIndex := strings.Index(hostname, "."); dotIndex >= 0 {
		hostname = hostname[:dotIndex]
	}

	if dashIndex := strings.LastIndex(hostname, "-"); dashIndex >= 0 && dashIndex < len(hostname)-1 {
		id, err := strconv.Atoi(hostname[dashIndex+1:])
		if err == nil && id >= 0 {
			return id, true
		}
	}

	return 0, false
}
