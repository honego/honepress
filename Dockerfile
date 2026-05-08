# syntax=docker/dockerfile:1

FROM node:22.22-alpine AS base

FROM base AS build-admin
WORKDIR /src/frontend/admin
COPY frontend/public ./public
COPY frontend/admin/package.json ./
RUN npm install
COPY frontend/admin/ ./
RUN npm run build
RUN mkdir -p /src/dist/admin \
    && if [ -d out/admin ]; then cp -R out/admin/. /src/dist/admin/; else cp -R out/. /src/dist/admin/; fi

FROM base AS build-theme
WORKDIR /src/frontend/theme
COPY frontend/public ./public
COPY frontend/theme/package.json ./
RUN npm install
COPY frontend/theme/ ./
RUN npm run build
RUN mkdir -p /src/dist/theme \
    && cp -R out/. /src/dist/theme/

FROM golang:1.26-alpine AS build-backend
WORKDIR /go/src/github.com/honeok/honepress
ENV CGO_ENABLED=0
COPY backend .
RUN set -ex \
    && go build -v -trimpath -ldflags="-s -w -buildid=" \
        -o /go/bin/honepress main.go

FROM alpine:3.23.4
LABEL org.opencontainers.image.authors="honeok <i@honeok.com>"
WORKDIR /app
COPY --from=build-backend /go/bin/honepress /app/honepress
COPY --from=build-admin /src/dist/admin /app/dist/admin
COPY --from=build-theme /src/dist/theme /app/dist/theme
RUN set -ex \
    && apk add --no-cache --update curl ca-certificates tzdata \
    && mkdir -p /app/data/content/posts /app/data/public
VOLUME /app/data
EXPOSE 8080
ENTRYPOINT ["./honepress", "-c", "config.yaml"]
