FROM golang:1.22-alpine AS builder

WORKDIR /build

# Download deps first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Build — CGO_ENABLED=0 produces a fully static binary (required for scratch)
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gtkai ./cmd/gtkai/

# Final image — scratch: no shell, no OS, just the binary
FROM scratch

COPY --from=builder /build/gtkai /gtkai

ENTRYPOINT ["/gtkai"]
