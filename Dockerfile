# syntax=docker/dockerfile:1
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 honeok <i@honeok.com>

FROM node:22.22-alpine AS base

FROM base AS build-admin
WORKDIR /src/frontend/admin
COPY frontend/public ../public
COPY frontend/admin/package.json frontend/admin/package-lock.json ./
RUN npm install
COPY frontend/admin/ ./
RUN npm run build

FROM base AS build-theme
WORKDIR /src/frontend/theme
COPY frontend/public ../public
COPY frontend/theme/package.json frontend/theme/package-lock.json ./
RUN npm install
COPY frontend/theme/ ./
RUN npm run build

FROM golang:1.26-alpine AS build-backend
WORKDIR /src/backend
ENV CGO_ENABLED=0
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -v -trimpath -ldflags="-s -w" \
    -o /go/bin/honepress .

FROM alpine:3.23.4
LABEL org.opencontainers.image.authors="honeok <i@honeok.com>"
WORKDIR /app
COPY --from=build-backend /go/bin/honepress /app/honepress
COPY config.example.yaml /app/config.example.yaml
COPY --from=build-admin /src/dist/admin /app/dist/admin
COPY --from=build-theme /src/dist/theme /app/dist/theme
COPY --from=build-theme /src/frontend/theme/templates /app/frontend/theme/templates
RUN set -ex \
    && apk add --no-cache --update curl ca-certificates tzdata \
    && mkdir -p /app/data/content/posts /app/data/public
VOLUME /app/data
EXPOSE 8080
ENV TZ=Asia/Shanghai
ENTRYPOINT ["/app/honepress", "-c", "/app/config.yaml"]
