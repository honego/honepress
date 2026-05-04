# honepress

honepress 是一个用 Go 和 TypeScript 编写的轻量博客程序。Go 负责 Markdown 渲染、静态 HTML、RSS、sitemap、API 和静态文件服务；TypeScript 负责后台 Vue 页面和前台主题切换脚本。

运行期采用单二进制部署：前台模板、后台构建产物和主题脚本都会嵌入到 `app` 中，外部只需要挂载 `config.yaml` 和 `data`。

## 功能特性

- Markdown 文件外置存储，启动和保存后自动生成静态页面。
- 固定链接由 Front Matter 的 `url` 字段决定，标题变更不会影响链接。`icon` 可给文章标题加 emoji。
- 自动生成 `/rss.xml`、`/sitemap.xml`。
- 后台提供文章列表、新建、编辑、删除、保存、预览、站点 icon 上传、评论配置和站点设置；发布后自动生成公开页面，草稿不生成公开页面。
- 支持 Basic Auth、giscus 评论开关、auto/light/dark 主题和 Markdown emoji 短码。

## 目录结构

```text
cmd/honepress/main.go
adapter/httpserver/
common/filesystem/
common/validation/
constant/
option/
service/
renderer/
model/
web/admin/
web/theme/
template/
data/content/posts/
data/public/
```

## Docker 部署

复制配置文件：

```bash
cp config.example.yaml config.yaml
```

编辑配置文件：

```bash
vim config.yaml
```

启动容器：

```bash
docker compose up -d
```

默认访问：

- 首页：`http://127.0.0.1:8080/`
- 文章列表：`http://127.0.0.1:8080/blog.html`
- 后台：`http://127.0.0.1:8080/admin/`

Go 构建命令：

```bash
go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/honepress
```

手动运行：

```bash
./app -c config.yaml
```

Docker 内部启动命令：

```bash
/app/honepress -c /app/config.yaml
```

容器运行层只复制 `/app/honepress`。`config.yaml` 通过 compose 挂载到 `/app/config.yaml`；如果直接运行镜像且配置文件不存在，程序会自动生成默认配置。Markdown 内容和生成后的静态文件仍放在 `/app/data`，方便备份和迁移。

## 配置文件

站点标题、描述、后台认证和评论配置都在 `config.yaml` 中管理。配置文件路径优先级：

1. 命令行参数 `-c` 或 `--config`
2. 环境变量 `HONEPRESS_CONFIG`
3. 默认 `./config.yaml`

除 `HONEPRESS_CONFIG` 可用于指定配置文件路径外，不再通过其他环境变量配置站点信息。配置文件不存在时，程序会自动生成默认 `config.yaml` 并继续启动。

## Markdown 文章格式

文章放在 `data/content/posts/`：

```md
---
title: "Docker 搭建 xxxx"
icon: "✨"
date: "2026-05-04 12:00:00"
description: "这是一篇 Docker 部署笔记。"
draft: false
url: "1.html"
comments: true
aliases:
  - "docker-old.html"
tags:
  - Docker
  - 部署
---

这里是正文内容。
```

Front Matter 只给程序读取，不会出现在渲染后的正文中。`icon` 会显示在文章标题前；正文支持 `:sparkles:` 这类 Markdown emoji 短码。`tags` 会显示在文章列表、文章页，并写入 RSS category。

## 固定链接说明

`url` 决定文章最终 HTML 文件名，例如 `url: "1.html"` 生成 `/1.html`。没有 `url` 时才使用 Markdown 文件名兜底。禁止路径穿越、中文路径、空格、斜杠和保留文件名。

## RSS 说明

RSS 自动生成到 `/rss.xml`。草稿不会进入 RSS，也不会生成公开文章页。

## sitemap 说明

sitemap 自动生成到 `/sitemap.xml`。后台路径和 API 路径不会进入 sitemap。

## 评论系统说明

评论使用 giscus，评论数据保存在 GitHub Discussions。设置 `comment.enabled: false` 或文章 `comments: false` 时不会输出评论脚本。giscus 配置缺失不会阻止启动，只会输出中文警告。

## 明暗主题说明

前台主题源码位于 `web/theme/src/theme.ts`，构建后生成 `theme.js` 并复制到 `data/public/theme.js`。主题状态保存在 `localStorage` 的 `honepress-theme`，支持 `auto`、`light`、`dark`。

## 后台说明

后台路径是 `/admin/`，API 路径是 `/api/`，两者都受 Basic Auth 保护。Markdown 预览调用 Go 后端 `/api/preview`，不会在前端使用 Markdown 渲染库。后台的“站点设置”区域可以修改站点标题、描述、网站 icon、字体、giscus 评论配置和默认主题，保存后会写回配置并自动更新静态页面。

## 反代建议

建议在反向代理中保留 `Host`、`X-Forwarded-For`、`X-Forwarded-Proto`，并只把公网流量转发到容器的 `127.0.0.1:8080`。生产环境必须设置强密码。

## 备份建议

必须备份：

- `data/content/posts`

可选备份：

- `data/public`
