package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
	"playlist/api"
) // main starts a gRPC server and waits for connection

type PlaylistServer struct {
	api.UnimplementedPlaylistServer
	Users map[string]string
}

func (s PlaylistServer) SignIn(ctx context.Context, request *api.AuthRequest) (*api.ResponseWithCode, error) {
	if s.Users[request.Login] == request.Password {
		return &api.ResponseWithCode{ReturnCode: 0}, nil
	}
	return &api.ResponseWithCode{ReturnCode: 1}, nil
}

func main() {
	// create a listener on TCP port 7777
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "localhost", 7777))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} // create a server instance
	s := PlaylistServer{}
	s.Users = make(map[string]string) // Create the TLS credentials
	s.Users["foo"] = "bar"
	creds, err := credentials.NewServerTLSFromFile("cert/server.crt", "cert/server.key")
	if err != nil {
		log.Fatalf("could not load TLS keys: %s", err)
	} // Create an array of gRPC options with the credentials
	opts := []grpc.ServerOption{grpc.Creds(creds)} // create a gRPC server object
	grpcServer := grpc.NewServer(opts...)          // attach the Ping service to the server
	api.RegisterPlaylistServer(grpcServer, &s)     // start the server
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
