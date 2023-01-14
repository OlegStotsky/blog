build-server:
	go build -o ./bin/server cmd/server/main.go

build-add-post:
	go build -o ./bin/add-post cmd/add-post/main.go

run: build-server
	./bin/server

add-post: build-add-post
	./bin/add-post