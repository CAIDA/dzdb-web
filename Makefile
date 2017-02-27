.PHONY: all rall fmt tags test testv lc doc

all: web docker

docker: Dockerfile web
	docker build -t="lanrat/vdzweb" .

web: main.go config.go datastore.go model.go server.go app.go api.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o $@ $^

fmt:
	gofmt -s -w -l .

tags:
	gotags `find . -name "*.go"` > tags

test:
	go test ./...

testv:
	go test -v ./...
