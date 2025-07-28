# run the server
run:
	go run ./cmd/server

# build the server
build:
	go build -o bin/server cmd/server/main.go

# run the server
run-server: build
	./bin/server

# clean the bin directory
clean:
	rm -rf bin/*