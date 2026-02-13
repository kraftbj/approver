.PHONY: build run clean

build:
	go build -o approver .

run: build
	./approver

clean:
	rm -f approver
