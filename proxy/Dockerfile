FROM golang:latest as builder

WORKDIR /tmp/build
ADD . /tmp/build/

RUN go get ./... && \
    go build -o proxy proxy.go

FROM alpine

COPY --from=builder /tmp/build/proxy /app/
WORKDIR /app
CMD ["./proxy"]