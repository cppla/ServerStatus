FROM golang:1.25-alpine AS builder

WORKDIR /src/server

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/*.go ./

ARG VERSION=2.0.0
ARG COMMIT=none
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /out/serverstatus .

FROM alpine:3.22

LABEL maintainer="cppla <https://cpp.la>"

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/config /app/data /app/web

COPY --from=builder /out/serverstatus /usr/local/bin/serverstatus
COPY server/config.json /app/config/config.json
COPY web /app/web/

ENV TZ=Asia/Shanghai \
    CONFIG_PATH=/app/config/config.json \
    STATS_PATH=/app/data/stats.json \
    WEB_DIR=/app/web \
    HTTP_ADDR=:80 \
    AGENT_ADDR=:35601

EXPOSE 80 35601

HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/serverstatus"]
