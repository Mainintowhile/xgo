ARG BASE_IMAGE=ghcr.io/crazy-max/xgo:base
FROM ${BASE_IMAGE}

ENV GO_VERSION 11411

RUN \
  export ROOT_DIST=https://golang.org/dl/go1.14.11.linux-amd64.tar.gz    && \
  export ROOT_DIST_SHA=ef150041e1af0890ecdd98ebdd6c759096884052a584c09ce50b2b5bb9bab2cd && \
  \
$BOOTSTRAP_PURE
