all: build

build:
	go build main.go
	go build client.go

proto_gen:
	protoc --go_out=api/ --go_opt=paths=source_relative     --go-grpc_out=api/ --go-grpc_opt=paths=source_relative -I/home/yurii/playlist/module_git/api api.proto

cert_gen:
	openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
      -keyout cert/server.key -out cert/server.crt -subj "/CN=localhost" \
      -addext "subjectAltName=DNS:localhost,DNS:localhost,IP:127.0.0.1"