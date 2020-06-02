# build stage
FROM golang:alpine AS build-env
RUN apk update && apk add --no-cache make git

WORKDIR /go/app/
COPY . .
RUN make

# final stage
FROM alpine

COPY --from=build-env /go/app/dnscoffee /app/
COPY --from=build-env /go/app/templates /app/templates
COPY --from=build-env /go/app/static /app/static

WORKDIR /app/

USER nobody

ENTRYPOINT /app/dnscoffee -listen $HTTP_LISTEN_ADDR
