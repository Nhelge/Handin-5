package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"handin5/pb"
)

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

func runClient(op string, addr string, bidder string, amount int64) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("client: failed to connect to %s: %v", addr, err)
	}
	defer conn.Close()

	c := pb.NewAuctionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch strings.ToLower(op) {
	case "bid":
		if bidder == "" {
			log.Fatalf("client: bidder must be set for op=bid")
		}
		if amount <= 0 {
			log.Fatalf("client: amount must be > 0 for op=bid")
		}
		resp, err := c.Bid(ctx, &pb.BidRequest{
			Bidder: bidder,
			Amount: amount,
		})
		if err != nil {
			log.Fatalf("client: Bid RPC error: %v", err)
		}
		fmt.Printf("Bid response: status=%s, message=%q\n", resp.Status.String(), resp.Message)

	case "result":
		resp, err := c.Result(ctx, &pb.ResultRequest{})
		if err != nil {
			log.Fatalf("client: Result RPC error: %v", err)
		}
		fmt.Printf("Result:\n")
		fmt.Printf("  finished      = %v\n", resp.Finished)
		fmt.Printf("  highestBidder = %q\n", resp.HighestBidder)
		fmt.Printf("  highestAmount = %d\n", resp.HighestAmount)
		fmt.Printf("  message       = %q\n", resp.Message)

	default:
		log.Fatalf("client: unknown op=%q (use 'bid' or 'result')", op)
	}
}

func runServer(id int, addr string, peersStr string, leaderID int, auctionSeconds int) {
	peers, err := parsePeers(peersStr)
	if err != nil {
		log.Fatalf("failed to parse peers: %v", err)
	}

	log.Printf("[node %d] starting at %s, leader=%d, duration=%ds", id, addr, leaderID, auctionSeconds)

	node := NewNode(id, addr, leaderID, peers, time.Duration(auctionSeconds)*time.Second)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuctionServiceServer(grpcServer, node)

	log.Printf("[node %d] gRPC server listening on %s", id, addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}

func main() {
	clientMode := flag.Bool("client", false, "run in client mode instead of server mode")

	clientOp := flag.String("op", "result", "client operation: 'bid' or 'result'")
	clientAddr := flag.String("caddr", "127.0.0.1:5001", "server address to connect to (client mode)")
	clientBidder := flag.String("bidder", "", "bidder name (client mode, op=bid)")
	clientAmount := flag.Int64("amount", 0, "bid amount (client mode, op=bid)")

	id := flag.Int("id", 1, "node id")
	addr := flag.String("addr", "127.0.0.1:5001", "listen address (server mode)")
	peersStr := flag.String("peers", "", "peers in form '1=addr1,2=addr2,...' (server mode)")
	leaderID := flag.Int("leader", 1, "id of leader node (server mode)")
	auctionSeconds := flag.Int("duration", 100, "auction duration in seconds (server mode)")

	flag.Parse()

	if *clientMode {
		runClient(*clientOp, *clientAddr, *clientBidder, *clientAmount)
	} else {
		runServer(*id, *addr, *peersStr, *leaderID, *auctionSeconds)
	}
}
