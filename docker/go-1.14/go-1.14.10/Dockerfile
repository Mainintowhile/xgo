ARG BASE_IMAGE=ghcr.io/crazy-max/xgo:base
FROM ${BASE_IMAGE}

ENV GO_VERSION 11410

RUN \
  export ROOT_DIST=https://golang.org/dl/go1.14.10.linux-amd64.tar.gz    && \
  export ROOT_DIST_SHA=66eb6858f375731ba07b0b33f5c813b141a81253e7e74071eec3ae85e9b37098 && \
  \
$BOOTSTRAP_PURE
