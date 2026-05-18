FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o /transport ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /transport /app/transport
EXPOSE 8081
ENTRYPOINT ["/app/transport"]
