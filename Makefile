# creates static binaries
CC := CGO_ENABLED=0 go build -a -installsuffix cgo

MODULE_SOURCES := $(shell find */ -type f -name '*.go' )
SOURCES := $(shell find . -type f -name '*.go' -maxdepth 1)
BIN := web

.PHONY: all fmt docker clean

all: web

docker: Dockerfile web
	docker build -t="lanrat/vdzweb" .

$(BIN): $(SOURCES) $(MODULE_SOURCES) go.mod
	$(CC) -o $@ $(SOURCES)

clean:
	rm $(BIN)

fmt:
	gofmt -s -w -l .
