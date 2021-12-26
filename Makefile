deps:
	go mod vendor
	go mod tidy
.PHONY: deps

build:
	go build -o bin/groceries .
.PHONY: build

clean:
	rm -f bin/groceries
.PHONY: clean

run: clean build
	./bin/groceries -env=dev
.PHONY: run

