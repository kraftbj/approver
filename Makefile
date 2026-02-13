.PHONY: build run clean test vet

build:
	go build -o approver .

run: build
	./approver

clean:
	rm -f approver

test:
	go test ./...

vet:
	go vet ./...
