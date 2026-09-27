.PHONY: build test vet run clean

build:
	go build -o hamlog .

test:
	go test ./...

vet:
	go vet ./...

run: build
	./hamlog

clean:
	rm -f hamlog *.adi
