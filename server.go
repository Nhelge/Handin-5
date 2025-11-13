package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"handin5/pb"
)

type Node struct {
	pb.UnimplementedAuctionServiceServer

	id       int
	address  string
	leaderID int // we hard-code leader for now (e.g., 1)

	peersMu sync.Mutex
	peers   map[int]pb.AuctionServiceClient // id -> client

	state *AuctionState
}

// NewNode creates a new node with a given duration for the auction.
func NewNode(id int, addr string, leaderID int, peerAddrs map[int]string, auctionDuration time.Duration) *Node {
	n := &Node{
		id:       id,
		address:  addr,
		leaderID: leaderID,
		state:    NewAuctionState(auctionDuration),
		peers:    make(map[int]pb.AuctionServiceClient),
	}

	// Connect to peers (except self)
	for pid, paddr := range peerAddrs {
		if pid == id {
			continue
		}
		conn, err := grpc.Dial(paddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("[node %d] failed to connect to peer %d (%s): %v", id, pid, paddr, err)
			continue
		}
		client := pb.NewAuctionServiceClient(conn)
		n.peers[pid] = client
	}

	return n
}

func (n *Node) isLeader() bool {
	return n.id == n.leaderID
}

// Bid handles client bids.
func (n *Node) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidResponse, error) {
	if !n.isLeader() {
		// Forward to leader
		n.peersMu.Lock()
		leaderClient, ok := n.peers[n.leaderID]
		n.peersMu.Unlock()
		if !ok {
			return &pb.BidResponse{
				Status:  pb.BidResponse_EXCEPTION,
				Message: "this node is not leader and cannot reach leader",
			}, nil
		}
		return leaderClient.Bid(ctx, req)
	}

	// Leader logic
	n.state.mu.Lock()
	defer n.state.mu.Unlock()

	// First, check locally if bid would be accepted (to avoid useless replication)
	err := n.state.applyBidLocked(req.Bidder, req.Amount)
	if err != nil {
		return &pb.BidResponse{
			Status:  pb.BidResponse_FAIL,
			Message: err.Error(),
		}, nil
	}

	// Now replicate to followers.
	ok := n.replicateBidToFollowers(ctx, req.Bidder, req.Amount)
	if !ok {
		log.Printf("[node %d] replication failed for bid %s:%d", n.id, req.Bidder, req.Amount)
		return &pb.BidResponse{
			Status:  pb.BidResponse_EXCEPTION,
			Message: "replication failed (not enough acks)",
		}, nil
	}

	return &pb.BidResponse{
		Status:  pb.BidResponse_SUCCESS,
		Message: "bid accepted",
	}, nil
}

// replicateBidToFollowers sends ReplicateBid to all peers and waits for majority acks.
func (n *Node) replicateBidToFollowers(ctx context.Context, bidder string, amount int64) bool {
	n.peersMu.Lock()
	peersCopy := make(map[int]pb.AuctionServiceClient, len(n.peers))
	for id, c := range n.peers {
		peersCopy[id] = c
	}
	n.peersMu.Unlock()

	totalNodes := len(peersCopy) + 1 // peers + self
	needed := totalNodes/2 + 1       // majority
	acks := 1                        // self already applied

	var wg sync.WaitGroup
	resCh := make(chan bool, len(peersCopy))

	for pid, client := range peersCopy {
		if pid == n.id {
			continue
		}
		wg.Add(1)
		go func(pid int, c pb.AuctionServiceClient) {
			defer wg.Done()

			cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			_, err := c.ReplicateBid(cctx, &pb.ReplicateBidRequest{
				Bidder: bidder,
				Amount: amount,
			})
			if err != nil {
				log.Printf("[node %d] error replicating to %d: %v", n.id, pid, err)
				resCh <- false
				return
			}
			resCh <- true
		}(pid, client)
	}

	wg.Wait()
	close(resCh)

	for ok := range resCh {
		if ok {
			acks++
		}
	}

	return acks >= needed
}

// ReplicateBid is called by the leader on all followers.
func (n *Node) ReplicateBid(ctx context.Context, req *pb.ReplicateBidRequest) (*pb.ReplicateBidResponse, error) {
	n.state.mu.Lock()
	defer n.state.mu.Unlock()

	// Apply bid using same logic
	err := n.state.applyBidLocked(req.Bidder, req.Amount)
	if err != nil {
		log.Printf("[node %d] ReplicateBid error: %v", n.id, err)
		return &pb.ReplicateBidResponse{Ok: false}, nil
	}

	return &pb.ReplicateBidResponse{Ok: true}, nil
}

// Result returns the current result or highest bid.
func (n *Node) Result(ctx context.Context, _ *pb.ResultRequest) (*pb.ResultResponse, error) {
	res := n.state.GetResult()

	if res.Finished {
		if res.HighestBidder == "" {
			return &pb.ResultResponse{
				Finished:      true,
				HighestBidder: "",
				HighestAmount: 0,
				Message:       "auction finished, no bids were placed",
			}, nil
		}
		return &pb.ResultResponse{
			Finished:      true,
			HighestBidder: res.HighestBidder,
			HighestAmount: res.HighestAmount,
			Message:       fmt.Sprintf("auction finished, winner is %s with bid %d", res.HighestBidder, res.HighestAmount),
		}, nil
	}

	return &pb.ResultResponse{
		Finished:      false,
		HighestBidder: res.HighestBidder,
		HighestAmount: res.HighestAmount,
		Message:       "auction still running",
	}, nil
}
