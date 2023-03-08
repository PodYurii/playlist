all: dir cert_gen build
.PHONY: clean_bin clean_cert

build:
	go build -o server main.go
	go build -o client client.go client_interface.go

dir:
	mkdir cert

cert_gen:
	openssl req -x509 -newkey rsa:4096 -sha256 -days 7 -nodes \
      -keyout cert/server.key -out cert/server.crt -subj "/CN=localhost" \
      -addext "subjectAltName=DNS:localhost,DNS:localhost,IP:127.0.0.1"

clean_bin:
	rm -rf server client

clean_cert:
	rm -rf cert/server.crt cert/server.key