# Dockerfile
FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gateway .

# minimalan runtime image
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/gateway .
COPY .env .env

ENV GATEWAY_ADDRESS=:8000
CMD ["./gateway"]
