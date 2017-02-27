FROM alpine:3.5
MAINTAINER ianfoster@vorsk.com

COPY web /app/web
COPY templates /app/templates

WORKDIR /app/

USER nobody

EXPOSE 8082

ENTRYPOINT /app/web
