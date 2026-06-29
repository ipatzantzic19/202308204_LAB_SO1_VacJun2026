FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/go-metrics-exporter .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/go-metrics-exporter /go-metrics-exporter
EXPOSE 9100
ENTRYPOINT ["/go-metrics-exporter"]
