# build stage
FROM golang:alpine AS build-env
RUN apk update && apk add --no-cache make git

WORKDIR /go/app/
COPY . .
RUN make

# final stage
FROM alpine

COPY --from=build-env /go/app/web /app/
COPY --from=build-env /go/app/templates /app/templates
COPY --from=build-env /go/app/static /app/static

WORKDIR /app/

USER nobody

EXPOSE 8082

ENTRYPOINT /app/web -listen $HTTP_LISTEN_ADDR
