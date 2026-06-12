FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download && go build -o tyvm .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/tyvm .
COPY templates/ templates/
COPY static/ static/
EXPOSE 8080
CMD ["./tyvm"]
