FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /threads-service ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /threads-service .

EXPOSE 8080
ENTRYPOINT ["./threads-service"]
