package main

import (
	"bufio"
	"crypto/tls"
	"github.com/PodYurii/playlist_module"
	"github.com/PodYurii/playlist_module/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"os"
)

func main() {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := grpc.Dial("localhost:7777", grpc.WithTransportCredentials(credentials.NewTLS(config)))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(conn)
	c := api.NewPlaylistClient(conn)
	in := bufio.NewReader(os.Stdin)
	var token uint64
	CFP := CommonFuncParams{in, &token, &c}
	LoginWindow(&CFP)
	pl := playlist_module.NewPlaylist()
	defer pl.Destructor()
	if err != nil {
		panic("oto.NewContext failed: " + err.Error())
	}
	MainWindow(&CFP, pl)
}
