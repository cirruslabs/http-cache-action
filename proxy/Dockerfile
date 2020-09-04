FROM golang:latest as builder

WORKDIR /tmp/build
ADD . /tmp/build/

RUN go get ./... && \
    CGO_ENABLED=0 GOOS=linux go build -o proxy proxy.go

FROM alpine

LABEL org.opencontainers.image.source = "https://github.com/cirruslabs/http-cache-action/"

RUN apk --no-cache add ca-certificates
COPY --from=builder /tmp/build/proxy /app/
WORKDIR /app
CMD ["./proxy"]