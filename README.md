# [DNS Coffee Frontend](https://dns.coffee)

## Building

Requires go compiler >= go1.13

```sh
$ make
```

## Running

Database connection information is set via [libpq environment variables](https://www.postgresql.org/docs/current/libpq-envars.html).

The listen address and port can be set with a flag.

To run all thats needed is the compiled `web` binary and the `static/` and `templates/` directories. The `static/` and `templates/` directories should be in the cwd when calling `web`.

### Flags

```sh
$ ./web -h
Usage of ./web:
  -listen string
        ip:port to listen on (default "127.0.0.1:8080")
```

### Example

```sh
$ export PGHOST=localhost
$ export PGDATABASE=DATABASE_NAME
$ export PGUSER=DB_USER
$ export PGPASSWORD=DB_PASSWORD
$ ./web
2020/04/29 21:45:22 Server starting on 127.0.0.1:8080
```

## Docker

The docker build used a 2-stage build. The first stage compiles the go program to a static binary, and the second stage copies the resulting binary and static files to a fresh image to run the web server.

### Build

```sh
$ make docker
```

### Run

docker-compose example:

```yaml
version: '3'

services:
    web:
        container_name: dnszone_web
        image: lanrat/dnscoffee
        restart: unless-stopped
        environment:
            - TZ=America/Los_Angeles
            - PGHOST=DATABASE_HOST
            - PGUSER=DATABASE_USER
            - PGPASSWORD=DATABASE_PASSWORD
            - PGDATABASE=DATABASE_NAME
            - HTTP_LISTEN_ADDR=0.0.0.0:8082
        healthcheck:
            test: ["CMD-SHELL", "wget --quiet --tries=1 --spider http://localhost:8082/ || exit 1"]
            interval: 30s
            timeout: 10s
            retries: 3
```
