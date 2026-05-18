FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend-builder

WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install
COPY web/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-builder

ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache gcc musl-dev \
    && if [ "$TARGETARCH" = "arm64" ] && [ "$BUILDPLATFORM" != "$TARGETPLATFORM" ]; then \
         apk add --no-cache aarch64-linux-musl-cross; \
       fi

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN if [ "$TARGETARCH" = "arm64" ] && [ "$BUILDPLATFORM" != "$TARGETPLATFORM" ]; then \
      CC=aarch64-linux-musl-gcc \
      CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go build -a -ldflags '-linkmode external -extldflags "-static"' -o margin-server ./cmd/server; \
    else \
      CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
      go build -a -ldflags '-linkmode external -extldflags "-static"' -o margin-server ./cmd/server; \
    fi

FROM node:20-alpine

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /app/margin-server ./margin-server

COPY --from=frontend-builder /app/web/dist ./dist
COPY --from=frontend-builder /app/web/node_modules ./node_modules

ENV PORT=8080
ENV API_PORT=8081
ENV DATABASE_URL=margin.db
ENV HOST=0.0.0.0
ENV API_URL=http://localhost:8081

EXPOSE 8080

COPY <<'EOF' /app/start.sh
#!/bin/sh
PORT=$API_PORT ./margin-server &
node ./dist/server/entry.mjs &
wait -n
EOF
RUN chmod +x /app/start.sh

CMD ["/app/start.sh"]
