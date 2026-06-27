FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY proto/go.mod proto/go.sum ./proto/
COPY go-consumer/go.mod go-consumer/go.sum ./go-consumer/
WORKDIR /src/go-consumer
RUN go mod download
COPY proto/ /src/proto/
COPY go-consumer/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /consumer .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /consumer /consumer
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/consumer"]
