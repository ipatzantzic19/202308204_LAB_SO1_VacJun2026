FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY proto ./proto
COPY go-d1/grpc-client ./go-d1/grpc-client
WORKDIR /src/go-d1/grpc-client
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o grpc-client .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /src/go-d1/grpc-client/grpc-client .
EXPOSE 9000
CMD ["./grpc-client"]
