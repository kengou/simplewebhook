# Dockerfile for simplewebhook
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26@sha256:fb612b7831d53a89cbc0aaa7855b69ad7b0caf603715860cf538df854d047b84 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/simplewebhook ./main.go


FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static:nonroot@sha256:f512d819b8f109f2375e8b51d8cfd8aafe81034bc3e319740128b7d7f70d5036

WORKDIR /
COPY --from=builder /workspace/bin/* .
USER 65532:65532

ENTRYPOINT ["/simplewebhook"]
