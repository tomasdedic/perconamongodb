FROM golang:1.17-alpine AS builder

RUN mkdir /app
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN go build main.go

FROM alpine

RUN apk update && apk add ca-certificates

COPY entrypoint.sh /entrypoint.sh
COPY --from=builder /app/main /main

ENTRYPOINT /entrypoint.sh
CMD ["/main"]
