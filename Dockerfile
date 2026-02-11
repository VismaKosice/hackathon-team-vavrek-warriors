FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pension-engine .

FROM scratch
COPY --from=builder /pension-engine /pension-engine
EXPOSE 8080
CMD ["/pension-engine"]
