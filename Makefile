.PHONY: build install docs lint test clean

build:
	go build -o bin/gherkio ./cmd/gherkio

install:
	go install ./cmd/gherkio

docs:
	cd docs && npm install && npm run dev

lint:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf bin
	find . -name "*.out" -delete
