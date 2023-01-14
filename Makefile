build-server:
	go build -o server cmd/server/main.go

build-add-post:
	go build -o add-post cmd/add-post/main.go

run: build-server
	./server

add-post: build-add-post
	./add-post