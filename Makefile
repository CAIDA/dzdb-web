# creates static binaries
CC := CGO_ENABLED=0 go build -ldflags "-w -s" -trimpath -a -installsuffix cgo

MODULE_SOURCES := $(shell find */ -type f -name '*.go' )
SOURCES := $(shell find . -maxdepth 1 -type f -name '*.go')
BIN := web

.PHONY: all fmt docker clean check

all: web

docker: Dockerfile
	docker build -t="lanrat/dnscoffee" .

$(BIN): $(SOURCES) $(MODULE_SOURCES) go.mod go.sum
	$(CC) -o $@ $(SOURCES)

clean:
	rm $(BIN)

fmt:
	gofmt -s -w -l .

check:
	golangci-lint run || true
	staticcheck -unused.whole-program -checks all ./...