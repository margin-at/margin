FROM oven/bun:1 AS frontend-builder

WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install
COPY web/ ./
# Lexicons are the source of truth for schema docs rendered by the web build.
COPY lexicons/ /app/lexicons/
RUN bun run build

FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o margin-server ./cmd/server


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
set -e

term() {
	kill -TERM "$api_pid" "$web_pid" 2>/dev/null
	exit 0
}
trap term TERM INT

PORT=$API_PORT ./margin-server &
api_pid=$!

until wget -qO- "http://localhost:$API_PORT/health" >/dev/null 2>&1; do
	if ! kill -0 "$api_pid" 2>/dev/null; then
		echo "API exited before becoming ready"
		exit 1
	fi
	echo "Waiting for API..."
	sleep 0.5
done

node ./dist/server/entry.mjs &
web_pid=$!

while kill -0 "$api_pid" 2>/dev/null && kill -0 "$web_pid" 2>/dev/null; do
	sleep 1
done

kill -TERM "$api_pid" "$web_pid" 2>/dev/null
exit 1
EOF
RUN chmod +x /app/start.sh

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- "http://localhost:$API_PORT/health" >/dev/null 2>&1 && wget -qO- "http://localhost:$PORT/" >/dev/null 2>&1 || exit 1

CMD ["/app/start.sh"]
