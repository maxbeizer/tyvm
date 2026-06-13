# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o tyvm .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
 && addgroup -S tyvm && adduser -S -G tyvm tyvm \
 && mkdir -p /app /data \
 && chown -R tyvm:tyvm /app /data
WORKDIR /app
COPY --from=builder /app/tyvm .
COPY templates/ templates/
COPY static/ static/
USER tyvm
ENV DB_PATH=/data/tyvm.db
VOLUME ["/data"]
EXPOSE 8080
CMD ["./tyvm"]
