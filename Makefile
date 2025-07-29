# run the server
run:
	go run ./cmd/main.go

# build the server
build:
	go build -o bin/uptime-monitor cmd/main.go

# run the server
run-server: build
	./bin/uptime-monitor

# generate templ files
generate:
	templ generate

# clean the bin directory
clean:
	rm -rf bin/*