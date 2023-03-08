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

type Buffer struct {
	data []byte
}

func NewBuffer() *Buffer {
	obj := Buffer{}
	obj.data = make([]byte, 0)
	return &obj
}

func (obj *Buffer) AddChunk(chunk []byte) {
	obj.data = append(obj.data, chunk...)
}

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
	var buf *Buffer
	pl := playlist_module.NewPlaylist()
	defer pl.Destructor()
	MainWindow(&CFP, pl, buf)
}
