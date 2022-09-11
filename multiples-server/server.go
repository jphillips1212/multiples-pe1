// Package main implements a server for Greeter service
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	pb "multiples-pe1/multiples"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	gRPCport = flag.Int("gRPC port", 50051, "The gRPC server port")
	restPort = flag.Int("REST port", 8080, "The REST API server port")
)

// gRPCserver is used to implement helloworld.GreeterServer
type gRPCserver struct {
	pb.UnimplementedMultiplesServer
}

// restServer is used as a receiver to make implementation more readable
type restServer struct{}

// restReply is used to marshal the JSON into when replying to a REST request
type restReply struct {
	Total uint64 `json:"total"`
}

// startGrpcServer registers the gRPCserver
func startGrpcServer(wg *sync.WaitGroup) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *gRPCport))
	if err != nil {
		log.Fatalf("GRPC port failed to listen: %v", err)
	}

	s := grpc.NewServer()

	pb.RegisterMultiplesServer(s, &gRPCserver{})

	reflection.Register(s)

	log.Printf("grpc server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	wg.Done()
}

// CalculateMultiplesOneLoop is called from the gRPC server
func (s *gRPCserver) CalculateMultiplesOneLoop(ctx context.Context, in *pb.MultiplesRequest) (*pb.MultiplesReply, error) {
	start := time.Now()
	log.Printf("Received calculate multiples in one loop request for: %d", in.GetTotal())

	multiplesTotal := calculateMultiplesOneLoop(in.GetTotal(), in.GetMultiples())

	elapsed := time.Since(start)
	log.Printf("One loop total answer is [%d] - Total time is [%s]", multiplesTotal, elapsed)

	return &pb.MultiplesReply{Total: uint64(multiplesTotal)}, nil
}

// CalculateMultiplesConcurrent is called from the gRPC server
func (s *gRPCserver) CalculateMultiplesConcurrent(ctx context.Context, in *pb.MultiplesRequest) (*pb.MultiplesReply, error) {
	start := time.Now()
	log.Printf("Received calculate multiples concurrently request for : %d", in.GetTotal())

	multiplesTotal := calculateMultiplesConcurrent(in.GetTotal(), in.GetMultiples())

	elapsed := time.Since(start)
	log.Printf("Concurrent total answer is: [%d] - Total time is [%s]", multiplesTotal, elapsed)

	return &pb.MultiplesReply{Total: uint64(multiplesTotal)}, nil
}

// startRestServers starts the rest server(s)
func startRestServer(wg *sync.WaitGroup) {
	router := mux.NewRouter()

	restServer := &restServer{}

	router.HandleFunc("/calculate-one-loop", restServer.CalculateMultiplesOneLoop).Methods("POST").Name("calculate-one-loop")
	router.HandleFunc("/calculate-concurrent", restServer.CalculateMultiplesConcurrent).Methods("POST").Name("calculate-concurrent")

	log.Printf("rest server listening at :%d", *restPort)

	http.ListenAndServe(fmt.Sprintf(":%d", *restPort), router)
}

func (s *restServer) CalculateMultiplesOneLoop(w http.ResponseWriter, req *http.Request) {

	// Decodes incoming request body
	requestDecoder := json.NewDecoder(req.Body)
	// Reuse protobuffer model
	var request pb.MultiplesRequest
	requestDecoder.Decode(&request)

	fmt.Println(fmt.Sprintf("Total value %d", request.Total))

	start := time.Now()
	log.Printf("Received calculate multiples in one loop request for : %d", request.Total)

	multiplesTotal := calculateMultiplesOneLoop(request.Total, request.Multiples)

	w.Header().Set("Content-Type", "application/json")
	resp := restReply{
		Total: uint64(multiplesTotal),
	}
	json.NewEncoder(w).Encode(resp)

	elapsed := time.Since(start)
	log.Printf("One loop total answer is: [%d] - Total time is [%s]", multiplesTotal, elapsed)
}

func (s *restServer) CalculateMultiplesConcurrent(w http.ResponseWriter, req *http.Request) {

	// Decodes incoming request body
	requestDecoder := json.NewDecoder(req.Body)
	// Reuse protobuffer model
	var request pb.MultiplesRequest
	requestDecoder.Decode(&request)

	fmt.Println(fmt.Sprintf("Total value %d", request.Total))

	start := time.Now()
	log.Printf("Received calculate multiples concurrently request for : %d", request.Total)

	multiplesTotal := calculateMultiplesConcurrent(request.Total, request.Multiples)

	w.Header().Set("Content-Type", "application/json")
	resp := restReply{
		Total: uint64(multiplesTotal),
	}
	json.NewEncoder(w).Encode(resp)

	elapsed := time.Since(start)
	log.Printf("Concurrent total answer is: [%d] - Total time is [%s]", multiplesTotal, elapsed)
}

func main() {
	flag.Parse()

	// Create a WaitGroup for both the gRPC server and the REST server
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go startGrpcServer(wg)
	go startRestServer(wg)

	wg.Wait()
}

// calculateMultiplesOneLoop calculates the total of the multiples by looping through the total one time
func calculateMultiplesOneLoop(total uint64, multiples []uint32) int {

	multiplesTotal := 0

	for i := 1; i < int(total); i++ {
		for _, m := range multiples {
			if i%int(m) == 0 {
				multiplesTotal += i
			}
		}
	}

	return multiplesTotal
}

// calculateMultiplesConcurrent calculates the total of the multiples by totalling each multiple up to the max concurrently
func calculateMultiplesConcurrent(total uint64, multiples []uint32) int {
	// Number of multiples that we need to loop through
	numMultiples := len(multiples)

	// Array to contain all of the channels, a channel will be assigned to each multiple total that needs to be calculated from sumMultiples
	chans := []chan int{}

	// Loop through each multiple and create a new go routine which calls sumMultiples
	// Add the newly created channel corresponding with this go routine to the chans
	for i := 0; i < numMultiples; i++ {
		tmp := make(chan int)
		chans = append(chans, tmp)
		go sumMultiples(int(multiples[i]), int(total), tmp)
	}

	wg := new(sync.WaitGroup)
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

	multiplesTotal := 0
	for i := range agg {
		multiplesTotal += i
	}

	return multiplesTotal
}

// sumMultiples calculates the total of that multiple up to the maximum number provided
func sumMultiples(multiple, max int, c chan<- int) {
	sum := 0

	for i := multiple; i <= max; i += multiple {
		sum += i
	}

	c <- sum
}
