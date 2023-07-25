FROM golang:1.20-buster as builder
WORKDIR /app

COPY go.* ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN go build -o exporter ./main.go

# The Real container
FROM ubuntu:20.04
RUN apt-get update && apt-get install -y openssl ca-certificates
RUN update-ca-certificates
RUN apt-get install -y libssl-dev
RUN rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/api /app/api
CMD ["/app/exporter"]
