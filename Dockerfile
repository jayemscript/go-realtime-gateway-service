FROM golang:1.26.6-bookworm AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/gateway /gateway

USER nonroot:nonroot
ENTRYPOINT ["/gateway"]
