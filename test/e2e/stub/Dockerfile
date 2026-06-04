# Build image for the e2e readiness stub (test/e2e/stub/main.go).
#
# Used ONLY by the kind e2e smoke (`make test-e2e` / .github/workflows/e2e.yml).
# Built locally and `kind load`-ed into the cluster; never pushed to a registry.
# The ZelosPlatform e2e overlay pins every component image to this tag with
# imagePullPolicy: Never so the operator's hard-coded /healthz + /readyz probes
# pass and pods reach Ready without any GHCR credentials.
#
# Runs as UID 65532 (nonroot) with a read-only root filesystem to satisfy the
# pod SecurityContext the operator renders.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/stub ./test/e2e/stub

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/stub /stub
USER 65532:65532
ENTRYPOINT ["/stub"]
