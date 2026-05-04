# syntax=docker/dockerfile:1
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 honeok <i@honeok.com>

FROM node:22.22-alpine AS base

FROM base AS build-admin
WORKDIR /src/web/admin
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm install
COPY web/admin/ ./
RUN npm run build

FROM base AS build-theme
WORKDIR /src/web/theme
COPY web/theme/package.json web/theme/package-lock.json ./
RUN npm install
COPY web/theme/ ./
RUN npm run build

FROM golang:1.26-alpine AS build-backend
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=build-admin /src/web/admin/dist ./web/admin/dist
COPY --from=build-theme /src/web/theme/dist ./web/theme/dist
RUN go build -v -trimpath -ldflags="-s -w" \
    -o /go/bin/app ./cmd/blog

FROM alpine:3.23.4
LABEL org.opencontainers.image.authors="honeok <i@honeok.com>"
WORKDIR /app
COPY --from=build-backend /go/bin/app /app/honepress
RUN set -ex \
    && apk add --no-cache --update curl ca-certificates tzdata
VOLUME /app/data
EXPOSE 8080
ENV TZ=Asia/Shanghai
ENTRYPOINT ["/app/honepress", "-c", "/app/config.yaml"]
