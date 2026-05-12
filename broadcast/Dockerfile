# syntax=docker/dockerfile:1
# check=error=true
FROM debian:trixie-slim AS builder
ARG YQ_VERSION=v4.53.2
ARG KUBECTL_VERSION=v1.36.0
ARG TARGETARCH

ENV OS_ARCH="${TARGETARCH:-amd64}"
ENV OS_NAME="linux"

WORKDIR /app

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get upgrade -y && apt-get --no-install-recommends install -y ca-certificates curl

RUN curl -SsLO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OS_NAME}/${OS_ARCH}/kubectl" \
 && curl -SsLO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OS_NAME}/${OS_ARCH}/kubectl.sha256" \
 && echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check \
 && chmod +x kubectl

RUN curl -SsL -o- "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_${OS_NAME}_${OS_ARCH}.tar.gz" | tar xz \
 && mv "yq_${OS_NAME}_${OS_ARCH}" yq \
 && chmod +x yq

FROM debian:trixie-slim AS final

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get upgrade -y && apt-get --no-install-recommends install -y ca-certificates
COPY --from=builder /app/kubectl /app/yq /usr/local/bin/
COPY broadcast.sh /usr/local/bin/

USER 1001
ENTRYPOINT ["broadcast.sh"]
