# syntax=docker/dockerfile:1

# 1) Painel web (React) -> internal/dashboard/dist
FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# 2) Gateway em Go puro (sem CGO: o SQLite e o modernc), com o painel embutido
FROM golang:1.26-alpine AS builder
WORKDIR /src
# Dependencias primeiro (cache) e exatamente as do go.sum: build reprodutivel.
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
COPY --from=web /src/internal/dashboard/dist ./internal/dashboard/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/rafaeldourado9/arcanum/internal/api.Version=${VERSION}" \
    -o /out/arcanum ./cmd/gateway \
 && mkdir -p /out/data

# 3) Imagem final distroless: so o binario, certificados e fuso horario. Sem shell, sem
#    gerenciador de pacotes, usuario sem privilegio (nonroot, UID 65532).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/arcanum /usr/local/bin/arcanum
COPY --from=builder --chown=65532:65532 /out/data /data

USER nonroot:nonroot
VOLUME /data
ENV GATEWAY_DB_PATH=/data
EXPOSE 3100

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/arcanum", "healthcheck"]

ENTRYPOINT ["/usr/local/bin/arcanum"]
