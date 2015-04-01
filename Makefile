.PHONY: all rall fmt tags test testv lc doc

all: api

run: api
	./api

api: main.go config.go datastore.go model.go server.go app.go
	go build -o $@ $^

fmt:
	gofmt -s -w -l .

tags:
	gotags `find . -name "*.go"` > tags

test:
	go test ./...

testv:
	go test -v ./...

lc:
	wc -l `find . -name "*.go"`

doc:
	godoc -http=:8000

clean:
	rm `find . -maxdepth 1 -perm -111 -type f`
