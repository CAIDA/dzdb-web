.PHONY: all fmt docker

all: web

docker: Dockerfile web
	docker build -t="lanrat/vdzweb" .

web: main.go config.go datastore.go model.go server.go app.go api.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -o $@ $^

fmt:
	gofmt -s -w -l .
