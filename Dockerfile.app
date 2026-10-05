FROM golang:1.27 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /parking-web ./cmd/parking-web

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /parking-web /parking-web
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/parking-web"]
