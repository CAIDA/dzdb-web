.PHONY: all rall fmt tags test testv lc doc

all: web

web: main.go config.go datastore.go model.go server.go app.go api.go
	go build -o $@ $^

fmt:
	gofmt -s -w -l .

tags:
	gotags `find . -name "*.go"` > tags

test:
	go test ./...

testv:
	go test -v ./...
