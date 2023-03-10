all: build_server build_client dir cert_gen
.PHONY: clean_bin clean_cert

build_server:
	go build -o server main.go

build_client:
	go build -o client client.go client_interface.go

dir:
	mkdir cert

cert_gen:
	openssl req -x509 -newkey rsa:4096 -sha256 -days 7 -nodes \
      -keyout cert/server.key -out cert/server.crt -subj "/CN=localhost" \
      -addext "subjectAltName=DNS:localhost,DNS:localhost,IP:127.0.0.1"

docker_compose:
	docker compose up

clean_bin:
	rm -rf server client

clean_cert:
	rm -rf cert/server.crt cert/server.key