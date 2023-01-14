build:
	go build -o server cmd/main.go

run: build
	./server