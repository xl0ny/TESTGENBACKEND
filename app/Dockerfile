FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server .
ENV APP_PORT=8080
EXPOSE 8080
CMD ["./server"]
