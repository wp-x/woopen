# Project Context

- Type: Legacy Project
- Updated At: 2026-02-28T18:18:28.607Z

## Domain
联通云盘（pan.wo.cn/沃云盘）文件管理与分享：直链/短链分享、免登录下载、上传 API、WebDAV 挂载

## Architecture
Go 单体服务（Gin 路由：/api 管理后台 + /s/:code 分享访问；/dav WebDAV Basic Auth）+ SQLite 持久化（默认 ./data/woopen.db；Docker 环境 /data/woopen.db）+ 前端 SPA（web：Vue 3 + Vite + Element Plus，产物 web/dist 由后端静态托管）+ wopan-sdk-go 对接联通云盘 API；internal 分层：config/repository/service/handler/middleware/wopan/webdav。

## Constraints
N/A

## Key Commands
- go test ./...
- go run ./cmd/server
- go build -o woopen ./cmd/server
- ./start.sh
- cd web && npm run dev
- cd web && npm run build
- docker compose up -d --build
- docker build -t woopen .

## Owners
N/A

## Update History
- 2026-02-28T18:18:28.607Z WoOpen 是单用户自托管的联通云盘轻量分享系统：管理员登录后台配置 Token、管理文件与分享（短链、密码/过期/次数限制）、查看访问统计；访客通过 /s/:code 访问分享并以 302 获取直链下载/预览；同时提供上传 API 与 /dav WebDAV 挂载能力。
- 2026-02-28T18:15:31.916Z 一个以 Go 实现的云盘/文件分享服务端项目（含鉴权中间件与文件列表/分享/上传进度等处理），使用 SQLite 存储，并提供 Web 前端与 Docker 化部署，同时集成 OpenSpec/规格模板工作流用于产品/需求/设计文档管理。
