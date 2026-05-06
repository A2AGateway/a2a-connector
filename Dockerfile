# Build stage
# Context must be the repo root (parent of a2a-connector/) so that the
# local a2a-protocol dependency is available:
#   docker build -t a2agateway/connector:latest -f a2a-connector/Dockerfile .
FROM golang:1.21-alpine AS builder

WORKDIR /workspace

RUN apk add --no-cache git ca-certificates tzdata

# Copy both modules so the replace directive resolves
COPY a2a-protocol/ ./a2a-protocol/
COPY a2a-connector/ ./a2a-connector/

WORKDIR /workspace/a2a-connector

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o connector ./cmd/connector

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /workspace/a2a-connector/connector .

# Runtime configuration via flags or environment variables.
# CONNECTOR_ID    — unique name registered with the gateway
# SAAS_ENDPOINT   — base URL of the A2A Gateway (e.g. http://gateway:8080)
# CONNECTOR_HOST  — public URL of this container (e.g. http://connector:8082)
# LEGACY_URL      — base URL of the legacy system to bridge
ENV CONNECTOR_ID=my-connector \
    SAAS_ENDPOINT="" \
    CONNECTOR_HOST="http://localhost:8082" \
    LEGACY_URL="http://localhost:8081" \
    PORT=8082

EXPOSE 8082

CMD ./connector \
    --connector-id="${CONNECTOR_ID}" \
    --saas-endpoint="${SAAS_ENDPOINT}" \
    --connector-host="${CONNECTOR_HOST}" \
    --legacy-url="${LEGACY_URL}" \
    --port="${PORT}"
