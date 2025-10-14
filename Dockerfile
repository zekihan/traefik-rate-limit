FROM --platform=$BUILDPLATFORM alpine:3.22.2@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 AS certs
RUN apk update && apk add ca-certificates

FROM --platform=$BUILDPLATFORM busybox:1.37.0@sha256:2f590fc602ce325cbff2ccfc39499014d039546dc400ef8bbf5c6ffb860632e7 AS builder

ARG TARGETPLATFORM

COPY ./dist ./dist

RUN case ${TARGETPLATFORM} in \
        "linux/amd64")    ARCH="amd64"    ;; \
        "linux/arm64")    ARCH="arm64"    ;; \
        "linux/arm/v7")   ARCH="armv7"   ;; \
    esac && \
    mv ./dist/traefik-rate-limit_linux_${ARCH}/traefik-rate-limit /usr/bin/traefik-rate-limit

FROM scratch AS final

COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/bin/traefik-rate-limit /usr/bin/traefik-rate-limit

CMD ["traefik-rate-limit", "server"]

HEALTHCHECK --interval=1s --timeout=3s --start-period=0s --retries=5 CMD [ "traefik-rate-limit", "healthcheck" ]

LABEL org.opencontainers.image.authors="zekihan@noqer.com" \
    org.opencontainers.image.url="https://noqer.com/" \
    org.opencontainers.image.documentation="https://docs.noqer.com/" \
    org.opencontainers.image.source="https://github.com/zekihan/traefik-rate-limit" \
    org.opencontainers.image.vendor="Zekihan AZMAN" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.title="traefik-rate-limit" \
    org.opencontainers.image.description="traefik-rate-limit is a rate limit middleware for traefik. It is a simple and lightweight solution to limit the number of requests to your backend services. It is designed to be used with traefik and can be easily integrated into your existing traefik setup."
