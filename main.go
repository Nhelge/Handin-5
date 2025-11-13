package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"

	"handin5/pb"
)

// peers flag format: "1=127.0.0.1:5001,2=127.0.0.1:5002,3=127.0.0.1:5003"
func parsePeers(peersStr string) (map[int]string, error) {
	res := make(map[int]string)
	if peersStr == "" {
		return res, nil
	}
	entries := strings.Split(peersStr, ",")
	for _, e := range entries {
		parts := strings.Split(e, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid peer format: %s", e)
		}
		var id int
		_, err := fmt.Sscanf(parts[0], "%d", &id)
		if err != nil {
			return nil, fmt.Errorf("invalid peer id %s: %w", parts[0], err)
		}
		res[id] = parts[1]
	}
	return res, nil
}

func main() {
	id := flag.Int("id", 1, "node id")
	addr := flag.String("addr", "127.0.0.1:5001", "listen address")
	peersStr := flag.String("peers", "", "peers in form '1=addr1,2=addr2,...'")
	leaderID := flag.Int("leader", 1, "id of leader node")
	auctionSeconds := flag.Int("duration", 100, "auction duration in seconds")
	flag.Parse()

	peers, err := parsePeers(*peersStr)
	if err != nil {
		log.Fatalf("failed to parse peers: %v", err)
	}

	log.Printf("[node %d] starting at %s, leader=%d, duration=%ds", *id, *addr, *leaderID, *auctionSeconds)

	node := NewNode(*id, *addr, *leaderID, peers, time.Duration(*auctionSeconds)*time.Second)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuctionServiceServer(grpcServer, node)

	log.Printf("[node %d] gRPC server listening on %s", *id, *addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}
