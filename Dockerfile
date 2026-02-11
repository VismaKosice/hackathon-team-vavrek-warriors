FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -gcflags="all=-B" -ldflags="-s -w" -o /pension-engine .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /pension-engine /pension-engine
EXPOSE 8080
CMD ["/pension-engine"]
