# Builder stage
FROM golang:1.23 as builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o main cmd/zencloud/main.go

# Final stage
FROM alpine:latest
WORKDIR /app

# Create uploads directory if needed
RUN mkdir -p uploads

# Copy the binary from builder
COPY --from=builder /app/main .

COPY .env /app/.env

EXPOSE 8080
CMD ["./main"]

