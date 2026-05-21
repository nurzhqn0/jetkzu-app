# syntax=docker/dockerfile:1.7
# Shared backend Dockerfile. It compiles all Go binaries once into one runtime image.
FROM golang:1.23-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0
ENV GOFLAGS=-mod=readonly

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/user-service ./services/user/cmd && \
    go build -trimpath -ldflags="-s -w" -o /out/driver-service ./services/driver/cmd && \
    go build -trimpath -ldflags="-s -w" -o /out/ride-service ./services/ride/cmd && \
    go build -trimpath -ldflags="-s -w" -o /out/payment-service ./services/payment/cmd && \
    go build -trimpath -ldflags="-s -w" -o /out/notification-service ./services/notification/cmd && \
    go build -trimpath -ldflags="-s -w" -o /out/api-gateway ./gateway/cmd

# ----------- runtime -----------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/ /app/
USER nonroot:nonroot
