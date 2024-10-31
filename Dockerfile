FROM golang:1.23.1-alpine3.19 AS builder

WORKDIR /feeder

RUN apk add \
  make \
  # used by make
  git \
  # to install GCC
  build-base

# ARG arch=x86_64
# ARG libwasmvm_sha=0421ad81247a46bbad6899c49d5081a5a080621ab9432e710152d85ba66c94bc
ARG arch=aarch64
ARG libwasmvm_sha=9429e9ab18f0b2519d9e3344b13fbb3ea339b7f1deedfaa2abc71522d190eaef
ARG libwasmvm_version=v1.5.5

# used to to link wasm
# ADD https://github.com/CosmWasm/wasmvm/releases/download/${libwasmvm_version}/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a
RUN wget -O /lib/libwasmvm_muslc.a https://github.com/CosmWasm/wasmvm/releases/download/${libwasmvm_version}/libwasmvm_muslc.${arch}.a
RUN sha256sum /lib/libwasmvm_muslc.a > /lib/libwasmvm_checksum
RUN if ! grep -q "${libwasmvm_sha} " /lib/libwasmvm_checksum ; then \
  echo "Expected libwasmvm signature: ${libwasmvm_sha}" && \
  echo "Actual   libwasmvm signature: $(cat /lib/libwasmvm_checksum)" && \
  exit 1; fi

COPY go.sum go.mod ./
RUN go mod download
COPY . .
RUN \
  # --mount=type=cache,target=/root/.cache/go-build \
  # --mount=type=cache,target=/go/pkg \
  CGO_ENABLED=1 make build

RUN file /feeder/build/pricefeeder

# ----------
FROM alpine:3.17

COPY --from=builder /feeder/build/pricefeeder /usr/bin/pricefeeder

ENTRYPOINT [ "/usr/bin/pricefeeder" ]
