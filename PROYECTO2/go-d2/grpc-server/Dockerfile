FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY proto ./proto
COPY go-d2/grpc-server ./go-d2/grpc-server
WORKDIR /src/go-d2/grpc-server
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o grpc-server .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /src/go-d2/grpc-server/grpc-server .
EXPOSE 50051
CMD ["./grpc-server"]
