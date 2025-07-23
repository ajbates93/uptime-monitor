# run the api
run:
	go run ./cmd/api

# build the api
build:
	go build -o bin/api cmd/api/main.go

# run the api
run-api: build
	./bin/api

# clean the bin directory
clean:
	rm -rf bin/*