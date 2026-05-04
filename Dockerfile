FROM node:22-alpine AS admin-builder
WORKDIR /src/web/admin
COPY web/admin/package.json web/admin/package-lock.json ./
RUN npm install
COPY web/admin/ ./
RUN npm run build

FROM node:22-alpine AS theme-builder
WORKDIR /src/web/theme
COPY web/theme/package.json web/theme/package-lock.json ./
RUN npm install
COPY web/theme/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=admin-builder /src/web/admin/dist ./web/admin/dist
COPY --from=theme-builder /src/web/theme/dist ./web/theme/dist
RUN go test ./...
RUN go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/blog

FROM alpine:latest
WORKDIR /app
COPY --from=go-build /out/app /app/app
VOLUME /app/data
EXPOSE 8080
CMD ["/app/app", "-c", "/app/config.yaml"]
