# Stage 1: Build
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ffmpeg

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o streamdavefast .

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app

COPY --from=builder /app/streamdavefast .
COPY --from=builder /app/index.html .
COPY --from=builder /app/css ./css
COPY --from=builder /app/js ./js
COPY --from=builder /app/font ./font
COPY --from=builder /app/reproductor ./reproductor
COPY --from=builder /app/Videos ./Videos
COPY --from=builder /app/processed ./processed

RUN mkdir -p /app/Videos /app/processed && \
    chmod 755 /app

EXPOSE 8080

CMD ["./streamdavefast"]
