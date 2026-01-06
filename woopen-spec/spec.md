# WoOpen - 联通云盘轻量网盘系统

> **项目名称**: WoOpen  
> **创建日期**: 2026-01-06  
> **版本**: v1.0.0-spec

---

## 1. 项目概述

### 1.1 项目背景
联通云盘（沃云盘）原生客户端存在下载必须登录的限制，无法直接生成公开分享链接。WoOpen 旨在解决这一痛点，提供一个轻量级的自托管网盘前端，以联通云盘为后端存储，实现无需登录的文件分享与下载功能。

### 1.2 核心目标
- **轻量化**: 单用户设计，资源占用极低
- **免登录下载**: 通过 302 直链重定向，访客无需登录即可下载
- **易于部署**: Docker 一键部署
- **AI 友好开发**: 使用主流技术栈，便于 AI 辅助开发与维护

### 1.3 技术栈选型

| 层面 | 技术选择 | 理由 |
|------|----------|------|
| **后端语言** | Go | 与 wopan-sdk-go 原生兼容，性能优秀，静态编译便于部署 |
| **后端框架** | Gin / Fiber | 轻量高性能，社区活跃 |
| **前端框架** | Vue 3 + Vite | 现代化、AI 友好、生态丰富 |
| **UI 组件库** | Element Plus / Naive UI | 与参考设计风格匹配 |
| **数据库** | SQLite | 轻量单文件，无需额外服务 |
| **部署方式** | Docker | 用户指定需求 |

---

## 2. 系统架构

### 2.1 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        WoOpen                                │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   前端 SPA  │◄──►│  后端 API   │◄──►│   SQLite    │     │
│  │  (Vue 3)    │    │   (Go/Gin)  │    │   数据库    │     │
│  └─────────────┘    └──────┬──────┘    └─────────────┘     │
│                            │                                 │
│                            ▼                                 │
│                   ┌─────────────────┐                       │
│                   │  wopan-sdk-go   │                       │
│                   └────────┬────────┘                       │
│                            │                                 │
└────────────────────────────┼────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   联通云盘 API   │
                    └─────────────────┘
```

### 2.2 核心模块

1. **认证模块**: Token 管理与自动刷新
2. **文件模块**: 云盘文件浏览与操作
3. **分享模块**: 分享链接的创建、管理、访问
4. **统计模块**: 访问日志与统计分析
5. **设置模块**: 系统配置管理

---

## 3. 功能规范

### 3.1 认证与 Token 管理

#### Token 获取方式
- **手动获取**: 用户从浏览器开发者工具中抓取 `refresh_token`
- **后台配置**: 在管理后台的设置页面填入 Token

#### Token 持久化策略
```
Token 刷新流程:
1. SDK 使用 refresh_token 请求新的 access_token
2. 联通云盘返回新的 access_token 和 refresh_token
3. 系统自动将新 Token 保存到 SQLite 数据库
4. 程序重启后从数据库读取最新 Token
```

#### 数据库 Token 表结构
```sql
CREATE TABLE tokens (
    id INTEGER PRIMARY KEY,
    refresh_token TEXT NOT NULL,
    access_token TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2 文件浏览与管理

#### 根文件夹配置
- 支持配置 `root_folder_id`（根文件夹 ID）
- 可设置为全盘根目录，也可设置为特定文件夹
- 后台只展示该文件夹及其子内容

#### 云盘空间支持
- **个人云**: `space_type = 1`
- **家庭云**: `space_type = 2`，需配置 `family_id`

#### 文件列表功能
- 分页加载文件列表
- 支持按名称/大小/修改时间排序
- 面包屑导航显示当前路径
- 搜索功能（在当前目录及子目录中搜索）

### 3.3 分享功能

#### 分享链接格式
```
https://your-domain.com/s/{share_code}
```
- `share_code`: 随机字符串或用户自定义字符串
- 示例: `https://your-domain.com/s/my-movie`

#### 分享类型
| 类型 | 说明 |
|------|------|
| 单文件分享 | 直接下载或预览单个文件 |
| 文件夹分享 | 访客可浏览文件夹内容，逐个下载 |

#### 分享链接属性
| 属性 | 类型 | 必填 | 说明 |
|------|------|------|------|
| share_code | string | 是 | 短链接标识，可自定义 |
| target_type | enum | 是 | file / folder |
| target_id | string | 是 | 联通云盘文件/文件夹 ID |
| target_path | string | 是 | 文件/文件夹路径（用于显示） |
| password | string | 否 | 访问密码 |
| expire_at | datetime | 否 | 过期时间，为空则永不过期 |
| description | string | 否 | 备注说明（仅管理员可见） |
| created_at | datetime | 是 | 创建时间 |
| is_active | boolean | 是 | 是否启用 |

#### 分享数据库表结构
```sql
CREATE TABLE shares (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    share_code TEXT UNIQUE NOT NULL,
    target_type TEXT NOT NULL CHECK(target_type IN ('file', 'folder')),
    target_id TEXT NOT NULL,
    target_path TEXT NOT NULL,
    target_name TEXT NOT NULL,
    password TEXT,
    expire_at DATETIME,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1
);
```

#### 批量分享
- 支持选择多个文件/文件夹
- 一次性生成多个独立的分享链接

### 3.4 下载与预览

#### 下载模式
- **302 重定向**: 服务端获取直链后，302 重定向到联通云盘真实下载地址
- **不支持流量中转**: 避免服务器带宽瓶颈

```
下载流程:
1. 访客点击下载
2. 后端调用 GetDownloadUrlV2 获取直链
3. 返回 HTTP 302 重定向到直链
4. 浏览器直接从联通云盘下载
```

#### 预览支持格式

| 类型 | 支持格式 | 预览方式 |
|------|----------|----------|
| 图片 | jpg, jpeg, png, gif, webp, svg, ico | 302 直链 + `<img>` |
| 视频 | mp4, webm | 302 直链 + HTML5 `<video>` |
| 音频 | mp3, wav, ogg, flac | 302 直链 + HTML5 `<audio>` |
| PDF | pdf | 302 直链 + 浏览器内置预览 |
| 文本 | txt, md, json, xml, yaml, log | 302 直链 + 前端 fetch 渲染 |

> **注意**: mkv、avi 等格式需要转码才能在浏览器播放，本项目不支持，仅提供下载

### 3.5 访问统计

#### 统计数据表结构
```sql
CREATE TABLE access_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    share_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK(action IN ('view', 'download')),
    ip_address TEXT,
    user_agent TEXT,
    referer TEXT,
    accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (share_id) REFERENCES shares(id)
);
```

#### 统计展示
- **Dashboard 统计卡片**:
  - 总分享数
  - 总访问量
  - 总下载量
  - 今日访问量
- **流量趋势图**: 最近 7 天的访问/下载趋势
- **单个分享统计**: 每个分享链接的访问详情

### 3.6 管理后台

#### 登录认证
- 简单密码登录（单用户系统）
- 密码存储使用 bcrypt 哈希
- JWT Token 维持会话

#### 后台功能模块

| 模块 | 功能 |
|------|------|
| 概览 (Dashboard) | 统计数据、流量趋势图 |
| 文件浏览 | 浏览云盘文件、创建分享 |
| 分享管理 | 查看、编辑、删除分享链接 |
| 访问日志 | 查看详细访问记录 |
| 系统设置 | Token 配置、根目录设置、密码修改 |

#### UI 设计风格
- **主题**: 简洁现代，蓝色主调
- **布局**: 左侧固定导航栏 + 右侧内容区域
- **卡片设计**: 圆角卡片 + 微阴影
- **暗黑模式**: 支持切换

参考设计:
![后台参考](admin_console_reference.png)

---

## 4. 分享页面设计

### 4.1 页面结构

```
┌──────────────────────────────────────────────────────────┐
│  [Logo]  WoOpen                          [暗黑模式切换]   │
├──────────────────────────────────────────────────────────┤
│                                                          │
│   ┌─────────────────────────────────────────────────┐   │
│   │                                                 │   │
│   │     ┌─────────────────────────────────────┐    │   │
│   │     │                                     │    │   │
│   │     │   文件/文件夹名称                    │    │   │
│   │     │   文件大小 / 文件数量                │    │   │
│   │     │                                     │    │   │
│   │     │   [列表视图] [网格视图]              │    │   │
│   │     │                                     │    │   │
│   │     │   ┌─────┐ ┌─────┐ ┌─────┐          │    │   │
│   │     │   │文件1│ │文件2│ │文件3│          │    │   │
│   │     │   └─────┘ └─────┘ └─────┘          │    │   │
│   │     │                                     │    │   │
│   │     │         [下载] [预览]               │    │   │
│   │     │                                     │    │   │
│   │     └─────────────────────────────────────┘    │   │
│   │                                                 │   │
│   └─────────────────────────────────────────────────┘   │
│                                                          │
│                    Powered by WoOpen                     │
└──────────────────────────────────────────────────────────┘

背景: 渐变/动态背景
```

### 4.2 密码保护页面

当分享设置了密码时，先显示密码输入页:

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│           ┌────────────────────────────┐                │
│           │                            │                │
│           │     🔒 该分享需要密码      │                │
│           │                            │                │
│           │   ┌──────────────────┐     │                │
│           │   │   请输入密码     │     │                │
│           │   └──────────────────┘     │                │
│           │                            │                │
│           │        [确认访问]          │                │
│           │                            │                │
│           └────────────────────────────┘                │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### 4.3 响应式设计
- **桌面端**: 居中卡片布局，最大宽度限制
- **移动端**: 全宽卡片，触摸友好的按钮尺寸
- **视图切换**: 列表视图 / 网格视图

### 4.4 文件图标
- 使用通用文件类型图标（非缩略图）
- 图标库推荐: Material Design Icons / Lucide Icons

### 4.5 二维码功能
- 每个分享链接可生成二维码
- 支持下载二维码图片

---

## 5. API 设计

### 5.1 认证相关

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 管理员登录 |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/me` | 获取当前用户信息 |

### 5.2 文件操作

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/files` | 获取文件列表 |
| GET | `/api/files/:id` | 获取文件详情 |
| GET | `/api/files/search` | 搜索文件 |

### 5.3 分享管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/shares` | 获取分享列表 |
| POST | `/api/shares` | 创建分享 |
| POST | `/api/shares/batch` | 批量创建分享 |
| PUT | `/api/shares/:id` | 更新分享 |
| DELETE | `/api/shares/:id` | 删除分享 |
| GET | `/api/shares/:id/qrcode` | 获取分享二维码 |

### 5.4 公开访问

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/s/:code` | 访问分享页面 |
| POST | `/s/:code/verify` | 验证分享密码 |
| GET | `/s/:code/files` | 获取分享文件列表（文件夹分享） |
| GET | `/s/:code/download/:fileId` | 下载文件（302 重定向） |
| GET | `/s/:code/preview/:fileId` | 预览文件（302 重定向） |

### 5.5 统计相关

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/stats/overview` | 获取概览统计 |
| GET | `/api/stats/trend` | 获取趋势数据 |
| GET | `/api/stats/shares/:id` | 获取单个分享统计 |

### 5.6 系统设置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/settings` | 获取设置 |
| PUT | `/api/settings` | 更新设置 |
| PUT | `/api/settings/password` | 修改密码 |
| POST | `/api/settings/token/test` | 测试 Token 有效性 |

---

## 6. 配置项

### 6.1 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `WOOPEN_PORT` | 否 | 8080 | 服务端口 |
| `WOOPEN_DATA_DIR` | 否 | /data | 数据目录（SQLite 存放位置） |
| `WOOPEN_ADMIN_PASSWORD` | 是 | - | 管理员密码（首次启动时设置） |
| `WOOPEN_JWT_SECRET` | 否 | 随机生成 | JWT 签名密钥 |
| `WOOPEN_SITE_URL` | 否 | - | 站点 URL（用于生成分享链接） |

### 6.2 系统设置（后台可配置）

| 设置项 | 说明 |
|--------|------|
| `refresh_token` | 联通云盘刷新令牌 |
| `root_folder_id` | 根文件夹 ID |
| `family_id` | 家庭云 ID（可选，留空使用个人云） |
| `site_title` | 站点标题 |
| `site_logo` | 站点 Logo URL |

---

## 7. Docker 部署

### 7.1 docker-compose.yml

```yaml
version: '3.8'

services:
  woopen:
    image: woopen:latest
    container_name: woopen
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - WOOPEN_ADMIN_PASSWORD=your_secure_password
      - WOOPEN_SITE_URL=https://your-domain.com
    restart: unless-stopped
```

### 7.2 Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o woopen ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/woopen .
COPY --from=builder /app/dist ./dist
EXPOSE 8080
CMD ["./woopen"]
```

---

## 8. 项目结构

```
woopen/
├── cmd/
│   └── server/
│       └── main.go           # 入口
├── internal/
│   ├── config/               # 配置管理
│   ├── handler/              # HTTP 处理器
│   ├── middleware/           # 中间件（认证、日志等）
│   ├── model/                # 数据模型
│   ├── repository/           # 数据访问层
│   ├── service/              # 业务逻辑层
│   └── wopan/                # 联通云盘 SDK 封装
├── web/                      # Vue 3 前端
│   ├── src/
│   │   ├── views/
│   │   │   ├── admin/        # 管理后台页面
│   │   │   └── share/        # 分享页面
│   │   ├── components/
│   │   ├── api/
│   │   └── store/
│   └── ...
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

---

## 9. 开发里程碑

### Phase 1: 核心基础 (MVP) ✅
- [x] 项目初始化与架构搭建
- [x] Token 管理与持久化
- [x] 文件列表浏览
- [x] 单文件分享与下载（302）
- [x] 简单管理后台

### Phase 2: 完整分享功能 ✅
- [x] 文件夹分享
- [x] 密码保护
- [x] 过期时间
- [x] 自定义短链接
- [x] 批量分享

### Phase 3: 体验优化 ✅
- [x] 在线预览（图片、视频、音频、PDF、文本）
- [x] 访问统计与日志
- [x] Dashboard 图表
- [x] 二维码生成
- [x] 暗黑模式

### Phase 4: 分享页面美化 ✅
- [x] 响应式布局
- [x] 列表/网格视图切换
- [x] 动态背景
- [x] 文件排序与搜索

---

## 10. 注意事项与风险

### 10.1 安全风险
- **302 重定向风险**: 联通云盘官方不推荐将直链用于公开分享，可能导致账号被限制
- **缓解措施**: 
  - 设置分享链接访问频率限制
  - 记录访问日志便于排查异常
  - 建议仅用于小范围分享

### 10.2 技术限制
- 直链可能有时效性，需要每次请求时实时获取
- 部分视频格式无法在浏览器直接播放
- 不支持打包下载文件夹（需要服务器中转）

### 10.3 依赖项
- **wopan-sdk-go**: `github.com/OpenListTeam/wopan-sdk-go v0.1.5`
- 该 SDK 基于 AList 项目开发，需关注其更新

---

## 附录

### A. 联通云盘 API 速查

| 功能 | SDK 方法 | 说明 |
|------|----------|------|
| 获取文件列表 | `QueryAllFiles()` | 分页获取 |
| 获取下载链接 | `GetDownloadUrlV2()` | 返回直链 |
| 创建目录 | `CreateDirectory()` | - |
| 移动文件 | `MoveFile()` | - |
| 复制文件 | `CopyFile()` | - |
| 删除文件 | `DeleteFile()` | - |
| 重命名 | `RenameFileOrDirectory()` | - |
| 上传文件 | `Upload2C()` | - |

### B. 参考资料
- AList 项目: https://github.com/OpenListTeam/OpenList
- wopan-sdk-go: https://github.com/OpenListTeam/wopan-sdk-go
- 联通云盘驱动源码: `drivers/wopan/driver.go`

---

*文档版本: 1.0.0*  
*最后更新: 2026-01-06*
