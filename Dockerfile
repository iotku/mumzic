# BUILD STAGE
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk update && apk upgrade --no-cache

RUN apk add --no-cache \
    opus-dev \
    opusfile-dev \
    pkgconfig \
    build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o mumzic


# RUN STAGE

FROM alpine:latest

WORKDIR /app

# I think yt-dlp pulls in ffmpeg
RUN apk add --no-cache \
    yt-dlp \
    yt-dlp-ejs \
    opusfile-dev

COPY --from=builder /app/mumzic .
COPY --from=builder /app/whitelist.txt /app/

ENTRYPOINT ["./mumzic"]
