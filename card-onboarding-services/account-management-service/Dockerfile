# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy the go.mod and go.sum from the root of services
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the account-management-service
RUN CGO_ENABLED=0 GOOS=linux go build -o account-service-bin ./account-management-service/main.go

# Stage 2: Create a minimal production image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/account-service-bin ./main

EXPOSE 8082

CMD ["./main"]
