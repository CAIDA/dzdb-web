# creates static binaries
CC := CGO_ENABLED=0 go build -ldflags "-w -s" -trimpath -a -installsuffix cgo

MODULE_SOURCES := $(shell find */ -type f -name '*.go' )
SOURCES := $(shell find . -maxdepth 1 -type f -name '*.go')
BIN := web

.PHONY: all fmt docker clean

all: web

docker: Dockerfile
	docker build -t="lanrat/vdz-web" .

$(BIN): $(SOURCES) $(MODULE_SOURCES) go.mod go.sum
	$(CC) -o $@ $(SOURCES)

clean:
	rm $(BIN)

fmt:
	gofmt -s -w -l .