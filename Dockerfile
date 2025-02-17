# Build stage
FROM golang:1.22.5 AS builder
WORKDIR /app
COPY . .
RUN go build -o TradeEngine cmd/main.go

# Run stage
FROM ubuntu:latest
LABEL authors="can.zanat"

# download the CA certificates
RUN apt-get update && apt-get install -y ca-certificates
RUN update-ca-certificates

WORKDIR /app/
COPY --from=builder /app/TradeEngine ./
COPY ./.config ./.config
RUN chmod +x TradeEngine
CMD ["./TradeEngine"]
