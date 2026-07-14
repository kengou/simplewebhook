# Dockerfile for simplewebhook
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26@sha256:d52df9c279840adf958d017ebb275651ed8338b953d39817bc3633a2e6b1bbcc AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/simplewebhook ./main.go


FROM --platform=${TARGETPLATFORM} gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240

WORKDIR /
COPY --from=builder /workspace/bin/simplewebhook .
USER 65532:65532

ENTRYPOINT ["/simplewebhook"]
