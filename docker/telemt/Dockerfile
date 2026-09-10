# syntax=docker/dockerfile:1
FROM debian:12-slim AS fetch
ARG TARGETARCH
ARG TELEMT_VERSION=3.5.7
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates curl tar; \
    rm -rf /var/lib/apt/lists/*; \
    case "$TARGETARCH" in \
      amd64) asset='telemt-x86_64-linux-musl.tar.gz'; sha='db26e363bb98f11a02a7fd6d0df455f4987af5cdb2a5897da7f6fb8d613fbf41' ;; \
      arm64) asset='telemt-aarch64-linux-musl.tar.gz'; sha='8730080863f8f8ed52ee11f9c51c4daa30b3044fc842538bf6dd8c91ac0572c1' ;; \
      *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    url="https://github.com/telemt/telemt/releases/download/${TELEMT_VERSION}/${asset}"; \
    curl -fL --retry 5 --retry-delay 2 --connect-timeout 10 --max-time 180 -o "/tmp/${asset}" "$url"; \
    printf '%s  %s\n' "$sha" "/tmp/${asset}" | sha256sum -c -; \
    tar -xzf "/tmp/${asset}" -C /tmp; \
    test -x /tmp/telemt; \
    install -m 0755 /tmp/telemt /telemt

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=fetch /telemt /app/telemt
USER nonroot:nonroot
EXPOSE 8443 9091
HEALTHCHECK --interval=20s --timeout=5s --start-period=15s --retries=5 CMD ["/app/telemt", "healthcheck", "/etc/telemt/config.toml", "--mode", "liveness"]
ENTRYPOINT ["/app/telemt"]
CMD ["/etc/telemt/config.toml"]
