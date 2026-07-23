FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/task-service ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/task-service ./task-service
COPY config/config.yaml ./config/config.yaml

EXPOSE 8080

ENTRYPOINT ["./task-service"]
