.PHONY: build test fmt vet check clean run

BINARY := web-crawler

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test -v -race ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

check: fmt vet test

clean:
	rm -f $(BINARY)
