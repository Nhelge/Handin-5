package main

import (
	"errors"
	"sync"
	"time"
)

type AuctionState struct {
	mu sync.Mutex

	startedAt time.Time
	duration  time.Duration

	highestBid    int64
	highestBidder string

	bidderBids map[string]int64
}

func NewAuctionState(duration time.Duration) *AuctionState {
	return &AuctionState{
		startedAt:     time.Now(),
		duration:      duration,
		highestBid:    0,
		highestBidder: "",
		bidderBids:    make(map[string]int64),
	}
}

var (
	ErrAuctionFinished  = errors.New("auction has finished")
	ErrBidTooLow        = errors.New("bid is not higher than previous bid")
	ErrBidTooLowHighest = errors.New("bid is not higher than current highest bid")
)

func (s *AuctionState) isFinishedLocked() bool {
	return time.Since(s.startedAt) >= s.duration
}

func (s *AuctionState) applyBidLocked(bidder string, amount int64) error {
	if s.isFinishedLocked() {
		return ErrAuctionFinished
	}

	if prev, ok := s.bidderBids[bidder]; ok {
		if amount <= prev {
			return ErrBidTooLow
		}
	}

	if amount <= s.highestBid {
		return ErrBidTooLowHighest
	}

	s.bidderBids[bidder] = amount
	s.highestBid = amount
	s.highestBidder = bidder

	return nil
}

type Result struct {
	Finished      bool
	HighestBidder string
	HighestAmount int64
}

func (s *AuctionState) GetResult() Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	return Result{
		Finished:      s.isFinishedLocked(),
		HighestBidder: s.highestBidder,
		HighestAmount: s.highestBid,
	}
}
