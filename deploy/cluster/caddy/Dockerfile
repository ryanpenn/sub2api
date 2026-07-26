# syntax=docker/dockerfile:1

ARG CADDY_VERSION=2.11.4

FROM caddy:${CADDY_VERSION}-builder-alpine AS builder
ARG CADDY_VERSION
ARG CADDY_STORAGE_REDIS_VERSION=v1.8.1
RUN xcaddy build "v${CADDY_VERSION}" \
    --with "github.com/pberkel/caddy-storage-redis@${CADDY_STORAGE_REDIS_VERSION}"

FROM caddy:${CADDY_VERSION}-alpine
ARG CADDY_VERSION
ARG CADDY_SOURCE_REVISION
ARG CADDY_STORAGE_REDIS_VERSION
ARG CADDY_STORAGE_REDIS_SOURCE_REVISION
ARG SUB2API_FORK_REVISION
LABEL org.opencontainers.image.title="Sub2API Caddy" \
      org.opencontainers.image.version="v${CADDY_VERSION}-redis-${CADDY_STORAGE_REDIS_VERSION}" \
      org.opencontainers.image.source="https://github.com/ryanpenn/sub2api" \
      org.opencontainers.image.revision="${SUB2API_FORK_REVISION}" \
      io.sub2api.caddy.revision="${CADDY_SOURCE_REVISION}" \
      io.sub2api.caddy-storage-redis.version="${CADDY_STORAGE_REDIS_VERSION}" \
      io.sub2api.caddy-storage-redis.revision="${CADDY_STORAGE_REDIS_SOURCE_REVISION}"
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
COPY docker-entrypoint.sh /usr/local/bin/sub2api-caddy-entrypoint
RUN chmod 0755 /usr/local/bin/sub2api-caddy-entrypoint
ENTRYPOINT ["/usr/local/bin/sub2api-caddy-entrypoint"]
CMD []
