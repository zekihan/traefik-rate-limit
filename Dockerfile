FROM --platform=$BUILDPLATFORM alpine:3.21.3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS certs
RUN apk update && apk add ca-certificates

FROM golang:alpine AS builder

WORKDIR /go/src/github.com/zekihan/traefik-rate-limit

COPY go.mod go.sum ./

COPY . .

RUN go mod tidy

RUN go build -tags server -o /usr/bin/traefik-rate-limit ./cmd

FROM scratch AS final

COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/bin/traefik-rate-limit /usr/bin/traefik-rate-limit

CMD ["traefik-rate-limit", "server"]

HEALTHCHECK --interval=1s --timeout=3s --start-period=0s --retries=5 CMD [ "traefik-rate-limit", "healthcheck" ]
