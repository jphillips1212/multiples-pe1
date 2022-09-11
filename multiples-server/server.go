// Package main implements a server for Greeter service
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "multiples-pe1/multiples"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

// server is used to implement helloworld.GreeterServer
type server struct {
	pb.UnimplementedMultiplesServer
}

// CalculateMultiplesOneLoop calculates the total of the multiples by looping through the total one time
func (s *server) CalculateMultiplesOneLoop(ctx context.Context, in *pb.MultiplesRequest) (*pb.MultiplesReply, error) {
	start := time.Now()
	log.Printf("Received calculate multiples in one loop request for: %d", in.GetTotal())

	multiplesTotal := 0

	for i := 1; i < int(in.GetTotal()); i++ {
		for _, m := range in.GetMultiples() {
			if i%int(m) == 0 {
				multiplesTotal += i
			}
		}
	}

	elapsed := time.Since(start)
	log.Printf("Answer is [%d]: Total time is [%s]", multiplesTotal, elapsed)

	return &pb.MultiplesReply{Total: uint64(multiplesTotal)}, nil
}

// CalculateMultiplesConcurrent calculates the total of the multiples by totalling each multiple up to the max concurrently
func (s *server) CalculateMultiplesConcurrent(ctx context.Context, in *pb.MultiplesRequest) (*pb.MultiplesReply, error) {
	start := time.Now()
	log.Printf("Received calculate multiples concurrently request for : %d", in.GetTotal())

	// Array of the multiples we're using
	mult := in.GetMultiples()

	// Number of multiples that we need to loop through
	numMultiples := len(mult)

	// Array to contain all of the channels, a channel will be assigned to each multiple total that needs to be calculated from sumMultiples
	chans := []chan int{}

	// Loop through each multiple and create a new go routine which calls sumMultiples
	// Add the newly created channel corresponding with this go routine to the chans
	for i := 0; i < numMultiples; i++ {
		tmp := make(chan int)
		chans = append(chans, tmp)
		go sumMultiples(int(mult[i]), int(in.GetTotal()), tmp)
	}

	var wg sync.WaitGroup
	wg.Add(len(chans))

	// Aggregate channel for receiving results back from the chans
	agg := make(chan int)

	// Loop through each channel created earlier and create a goRoutine to wait listening on each channel
	for _, ch := range chans {
		go func(c chan int) {
			for total := range c {
				// Send the total response to the aggregate channel
				agg <- total
				wg.Done()
			}
		}(ch)
	}

	// goRoutine to wait for all multiples to be added to the aggregate channel before closing it
	go func() {
		wg.Wait()
		close(agg)
	}()

	total := 0
	for i := range agg {
		total += i
	}

	elapsed := time.Since(start)
	log.Printf("Concurrent total answer is: [%d] - Total time is [%s]", total, elapsed)

	return &pb.MultiplesReply{Total: uint64(total)}, nil
}

// sumMultiples calculates the total of that multiple up to the maximum number provided
func sumMultiples(multiple, max int, c chan<- int) {
	sum := 0

	for i := multiple; i <= max; i += multiple {
		sum += i
	}

	c <- sum
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	pb.RegisterMultiplesServer(s, &server{})

	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
