FROM golang:latest as builder

WORKDIR /tmp/build
ADD . /tmp/build/

RUN go get ./... && \
    go build -o proxy proxy.go

FROM alpine

LABEL org.opencontainers.image.source = "https://https://github.com/cirruslabs/http-cache-action/"

COPY --from=builder /tmp/build/proxy /app/
WORKDIR /app
CMD ["./proxy"]