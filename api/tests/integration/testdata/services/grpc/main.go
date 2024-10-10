//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative sample.proto
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Define a sample gRPC service
type sampleServer struct {
	UnimplementedSampleServer
}

// Define the service method
func (s *sampleServer) SayHello(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	return &HelloResponse{Message: "Hello " + req.Name}, nil
}

func (s *sampleServer) Ouch(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	// Return an error
	err := status.Error(codes.Internal, "An internal error occurred")
	return nil, err
}

func main() {
	// Create a TCP listener
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a gRPC server
	s := grpc.NewServer()

	// Register the service with the server
	RegisterSampleServer(s, &sampleServer{})

	// Start the server
	fmt.Println("Server started. Listening on port :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
