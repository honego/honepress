# HonePress

HonePress 是一个用 Go 和 Vue 3 / TypeScript 编写的轻量博客程序。Go 负责 Markdown 渲染、静态 HTML、RSS、sitemap、API 和静态文件服务；前端分为后台管理界面和前台主题脚本。

项目现在按 `backend`、`frontend`、`dist`、`data` 拆分：Go module 位于 `backend` 目录，运行时从文件系统读取前端构建产物，不依赖 Go embed。

## 功能特性

- Markdown 文件外置存储，启动和保存后自动生成静态页面。
- 固定链接由 Front Matter 的 `url` 字段决定，标题变更不会影响链接。`icon` 可给文章页生成 emoji favicon。
- 自动生成 `/rss.xml`、`/sitemap.xml`。
- 后台提供文章列表、新建、编辑、删除、保存、预览、站点 icon、评论配置和站点设置；发布后自动生成公开页面，草稿不生成公开页面。
- 支持 Basic Auth、giscus 全局评论开关、auto/light/dark 主题和 Markdown emoji 短码。

## 目录结构

```text
backend/
  go.mod
  go.sum
  main.go
  internal/
    server/
    filesystem/
    validation/
    core/
    model/
    config/
    renderer/
    service/
frontend/
  public/
    favicon.ico
    favicon.svg
    apple-touch-icon.png
    logo.svg
  admin/
  theme/
    templates/
dist/
  admin/
  theme/
data/
  content/posts/
  public/
```

说明：

- `frontend/theme/templates` 是运行时渲染静态站点所需的模板和前台 CSS。
- `frontend/admin` 是后台 Vue 3 + Vite 源码，构建产物输出到 `dist/admin`。
- `frontend/theme` 是前台主题脚本源码，构建产物输出到 `dist/theme`。
- `frontend/public` 是项目默认 favicon/logo 的唯一维护位置，admin 和 theme 构建时共用。
- `data/content/posts` 是文章目录，路径保持不变。
- `data/public` 是运行时生成的公开站点和用户上传/配置资源目录，不等同于 `frontend/public`。

## 构建

构建后台：

```bash
cd frontend/admin
npm install
npm run build
```

构建前台主题：

```bash
cd frontend/theme
npm install
npm run build
```

Go 构建命令：

```bash
cd backend
go build -trimpath -ldflags="-s -w" -o /out/honepress .
```

手动运行：

```bash
./honepress -c config.yaml
```

启动时会检查：

- `dist/admin/index.html`
- `dist/theme/.vite/manifest.json` 和 `dist/theme/assets/*`
- `frontend/theme/templates/index.html`
- `frontend/theme/templates/blog.html`
- `frontend/theme/templates/post.html`
- `frontend/theme/src/style.css`

如果缺少构建产物或模板，会输出清晰的中文错误并停止启动。

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
- 归档页：`http://127.0.0.1:8080/archive.html`
- 后台：`http://127.0.0.1:8080/admin/`

Docker 内部启动命令保持不变：

```bash
/app/honepress -c /app/config.yaml
```

推荐运行时目录：

```text
/app/
  honepress
  config.yaml
  config.example.yaml
  frontend/theme/templates/
  dist/
    admin/
    theme/
  data/
    content/
    public/
```

## 配置文件

站点标题、描述、后台认证和评论配置都在 `config.yaml` 中管理。配置文件路径优先级：

1. 命令行参数 `-c` 或 `--config`
2. 环境变量 `HONEPRESS_CONFIG`
3. 默认 `./config.yaml`

除 `HONEPRESS_CONFIG` 可用于指定配置文件路径外，不再通过其他环境变量配置站点信息。配置文件不存在时，程序会自动生成默认 `config.yaml` 并继续启动。

## Icon 说明

项目默认 favicon/logo 维护在 `frontend/public`：

- `frontend/public/favicon.ico`
- `frontend/public/favicon.svg`
- `frontend/public/apple-touch-icon.png`
- `frontend/public/logo.svg`

后台“网站 Icon”是运行时站点配置，会写入 `config.yaml` 或运行时资源目录。渲染页面时优先使用配置中的网站 icon；没有配置时回退到 `/favicon.svg`。

## Markdown 文章格式

文章放在 `data/content/posts/`，文件名会由后台按标题自动维护，例如 `世界你好.md`：

```md
---
title: "世界你好"
icon: "☘️"
date: "2026-05-05 00:00:00"
description: "欢迎使用 HonePress。"
draft: false
url: "hello.html"
aliases: []
tags:
  - HonePress
---

欢迎使用 HonePress 。这是您的第一篇文章，编辑或删除它，然后开始写作吧！
```

Front Matter 只给程序读取，不会出现在渲染后的正文中。站点 icon 是全站默认 favicon；单篇文章的 `icon` 会生成该文章自己的 emoji favicon，没有设置时回退到站点 icon 或默认 `/favicon.svg`。

## 固定链接说明

`url` 决定文章最终 HTML 文件名，例如 `url: "hello.html"` 生成 `/hello.html`。后台会按标题自动维护 Markdown 文件名，例如标题“世界你好”保存为 `世界你好.md`。固定链接禁止路径穿越、中文路径、空格、斜杠和保留文件名。

## RSS 和 sitemap

RSS 自动生成到 `/rss.xml`。sitemap 自动生成到 `/sitemap.xml`。草稿不会进入 RSS，也不会生成公开文章页。后台路径 `/admin/` 和 API 路径 `/api/` 不会进入 sitemap。

## 评论系统说明

评论使用 giscus，评论数据保存在 GitHub Discussions。评论只由全局 `comment.enabled` 控制；关闭时不会输出评论脚本。giscus 配置缺失不会阻止启动，只会输出中文警告。

## 明暗主题说明

前台主题源码位于 `frontend/theme/src/theme.ts` 和 `frontend/theme/src/style.css`，构建后生成 `dist/theme/assets/*.js`、`dist/theme/assets/*.css` 以及 `dist/theme/.vite/manifest.json`。运行时会读取 manifest，并把资源复制到 `data/public/assets/`。主题状态保存在 `localStorage` 的 `honepress-theme`，支持 `auto`、`light`、`dark`。

## 后台说明

后台路径保持 `/admin/`，API 路径保持 `/api/`。Markdown 预览调用 Go 后端 `/api/preview`，不会在前端使用 Markdown 渲染库。后台的“站点设置”区域可以修改站点标题、描述、网站 icon、字体、giscus 评论配置和默认主题，保存后会写回配置并自动更新静态页面。

## 备份建议

必须备份：

- `data/content/posts`
- `config.yaml`

可选备份：

- `data/public`
