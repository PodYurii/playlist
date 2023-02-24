package main

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"playlist/api"
)

func main() {
	var conn *grpc.ClientConn // Create the client TLS credentials
	creds, err := credentials.NewClientTLSFromFile("cert/server.crt", "")
	if err != nil {
		log.Fatalf("could not load tls cert: %s", err)
	} // Initiate a connection with the server
	conn, err = grpc.Dial("localhost:7777", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	c := api.NewPlaylistClient(conn)
	response, err := c.SignIn(context.Background(), &api.AuthRequest{Login: "foo", Password: "br"})
	if err != nil {
		log.Fatalf("error when calling SignIn: %s", err)
	}
	log.Printf("Response from server: %d", response.ReturnCode)
}
