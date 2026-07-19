# MetalTracker middleman - price cache API
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/middleman ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget su-exec \
	&& adduser -D -u 1000 -H -s /sbin/nologin middleman \
	&& mkdir -p /data \
	&& chown middleman:middleman /data

COPY --from=build /out/middleman /usr/local/bin/middleman
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN sed -i 's/\r$//' /docker-entrypoint.sh \
	&& chmod +x /docker-entrypoint.sh

ENV LISTEN_ADDR=:8080
ENV DB_PATH=/data/prices.db
ENV POLL_INTERVAL=1h

WORKDIR /data
EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
	CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/docker-entrypoint.sh"]
