# Dockerfile for simplewebhook
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26@sha256:b54cbf583d390341599d7bcbc062425c081105cc5ef6d170ced98ef9d047c716 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/simplewebhook ./main.go


FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39

WORKDIR /
COPY --from=builder /workspace/bin/simplewebhook .
USER 65532:65532

ENTRYPOINT ["/simplewebhook"]
