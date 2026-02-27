FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /aevitas ./cmd/aevitas

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /aevitas /usr/local/bin/aevitas

RUN mkdir -p /root/.aevitas/workspace

VOLUME ["/root/.aevitas"]

EXPOSE 18790 9876 9886

ENTRYPOINT ["aevitas"]
CMD ["gateway"]
