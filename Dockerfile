# syntax=docker/dockerfile:1

# ---- web: build the static export (Next.js output: export) ----
FROM oven/bun:1.4.2 AS web
WORKDIR /src
COPY package.json bun.lock ./
COPY packages ./packages
COPY apps/web ./apps/web
RUN bun install --frozen-lockfile
RUN bun run build:web

# ---- api: build the Go server that serves API + static files ----
FROM golang:1.24-bookworm AS api
WORKDIR /src
COPY apps/api/go.* ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# ---- runtime: one small image ----
FROM debian:bookworm-slim
WORKDIR /srv
COPY --from=api /out/server /usr/local/bin/server
COPY --from=web /src/apps/web/out /srv/out
ENV PORT=8080 \
    STATIC_DIR=/srv/out
EXPOSE 8080
CMD ["server"]
