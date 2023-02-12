db-dir-path := /var/lib/blog

db-dir:
	echo "creating db dir " $(db-dir-path)
	mkdir -p $(db-dir-path)

build-server:
	go build -o ./bin/server cmd/server/main.go

build-add-post:
	go build -o ./bin/add-post cmd/add-post/main.go

run: build-server db-dir
	./bin/server --db-dir=$(db-dir-path)

add-post: build-add-post
	./bin/add-post