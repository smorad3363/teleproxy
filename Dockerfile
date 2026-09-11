FROM golang:1.27.1-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/teleproxy-control ./cmd/control

FROM debian:bookworm-slim
RUN groupadd --gid 10001 teleproxy \
    && useradd --uid 10001 --gid 10001 --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin teleproxy \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/teleproxy-control /usr/local/bin/teleproxy-control
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=5 CMD ["/usr/local/bin/teleproxy-control", "healthcheck", "http://127.0.0.1:8080/readyz"]
ENTRYPOINT ["/usr/local/bin/teleproxy-control"]
