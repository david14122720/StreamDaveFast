# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ffmpeg

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o streamdavefast .

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ffmpeg ca-certificates

WORKDIR /app

RUN mkdir -p Videos processed css js font reproductor && chmod 755 .

COPY --from=builder /app/streamdavefast .
COPY --from=builder /app/index.html .
COPY --from=builder /app/css ./css
COPY --from=builder /app/js ./js
COPY --from=builder /app/font ./font
COPY --from=builder /app/reproductor ./reproductor

EXPOSE 8080

CMD ["./streamdavefast"]
