GIT_DATE := $(shell git log -1 --pretty='%aI')
GIT_HASH := $(shell git rev-parse HEAD)
GIT_BRANCH := $(shell git symbolic-ref --short HEAD)

# creates static binaries
LD_FLAGS := -ldflags "-w -s \
	-X 'dnscoffee/version.GitDate=$(GIT_DATE)' \
	-X 'dnscoffee/version.GitHash=$(GIT_HASH)' \
	-X 'dnscoffee/version.GitBranch=$(GIT_BRANCH)'"
CC := CGO_ENABLED=0 go build -trimpath -a -installsuffix cgo $(LD_FLAGS)

MODULE_SOURCES := $(shell find */ -type f -name '*.go' )
SOURCES := $(shell find . -maxdepth 1 -type f -name '*.go')
BIN := dnscoffee

.PHONY: all fmt docker clean check

all: $(BIN)

docker: Dockerfile
	docker build -t="lanrat/dnscoffee" .

$(BIN): $(SOURCES) $(MODULE_SOURCES) go.mod go.sum
	$(CC) -o $@ $(SOURCES)

clean:
	rm $(BIN)

fmt:
	gofmt -s -w -l .

check: | check1 check2

check1:
	golangci-lint run --enable-all

check2:
	staticcheck -unused.whole-program -checks all ./...