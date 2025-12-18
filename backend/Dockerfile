# =========================================
# Build stage
# =========================================
FROM golang:1.23 AS build
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
ARG COMMIT
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.Commit=$COMMIT" -o /bin/server ./cmd/server

# =========================================
# Runtime stage
# =========================================
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /bin/server /app/server

# Non-root (distroless uses nonroot user)
USER nonroot:nonroot

EXPOSE 8080
ENV PORT=8080
ENTRYPOINT ["/app/server"]
