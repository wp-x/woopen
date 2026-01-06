# WoOpen - 联通云盘轻量网盘系统

基于联通云盘的轻量自托管网盘前端，支持免登录下载的文件分享。

## ✨ 功能特点

- 🚀 **轻量化**: 单用户设计，资源占用极低
- 🔗 **免登录下载**: 通过 302 直链重定向，访客无需登录即可下载
- 📁 **文件夹分享**: 支持分享整个文件夹，访客可浏览目录结构
- 🎯 **批量分享**: 一次选择多个文件批量创建分享
- 🔒 **密码保护**: 可选的访问密码保护
- ⏰ **过期时间**: 设置分享链接的有效期
- 📊 **访问统计**: 追踪分享链接的访问和下载次数
- 👁️ **在线预览**: 图片、视频、音频、PDF、文本在线预览
- 📱 **二维码生成**: 快速生成分享链接二维码
- 🌙 **暗黑模式**: 支持深色/浅色主题切换
- 🐳 **易于部署**: Docker 一键部署

## 🖼️ 界面预览

管理后台采用现代化设计，包括：
- Dashboard 统计概览
- 文件浏览器（支持多选）
- 分享管理中心
- 系统设置面板

分享页面支持：
- 单文件下载/预览
- 文件夹浏览与下载
- 密码验证页面
- 响应式布局

## 🚀 快速开始

### 开发模式

1. 启动后端：
```bash
# 编译后端
go build -o woopen ./cmd/server

# 运行
./woopen
```

2. 启动前端（另一个终端）：
```bash
cd web
npm install
npm run dev
```

3. 访问 http://localhost:3000
   - 默认密码：`admin123`

### 生产模式（Docker）

```bash
# 编辑 docker-compose.yml 设置密码
docker-compose up -d
```

访问 http://your-server:8080

## ⚙️ 配置说明

### 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `WOOPEN_PORT` | 否 | 8080 | 服务端口 |
| `WOOPEN_DATA_DIR` | 否 | /data | 数据目录 |
| `WOOPEN_ADMIN_PASSWORD` | **是** | admin123 | 管理员密码 |
| `WOOPEN_JWT_SECRET` | 否 | 随机 | JWT 密钥 |
| `WOOPEN_SITE_URL` | 否 | - | 站点 URL |

### 获取 Refresh Token

1. 登录 [联通云盘](https://pan.wo.cn)
2. 打开浏览器开发者工具（F12）
3. 在 Network > Cookie 或 Application > Local Storage 中找到 `refresh_token`
4. 在管理后台的「系统设置」中填入

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **数据库**: SQLite
- **SDK**: [wopan-sdk-go](https://github.com/OpenListTeam/wopan-sdk-go)

### 前端
- **框架**: Vue 3 + TypeScript
- **构建**: Vite
- **UI**: Element Plus
- **状态管理**: Pinia
- **路由**: Vue Router

## 📁 目录结构

```
woopen/
├── cmd/server/             # 后端入口
├── internal/               # 后端业务逻辑
│   ├── config/             # 配置管理
│   ├── handler/            # HTTP 处理器
│   ├── middleware/         # 中间件（JWT/CORS）
│   ├── model/              # 数据模型
│   ├── repository/         # 数据访问层
│   └── wopan/              # 云盘 SDK 封装
├── web/                    # Vue 3 前端
│   ├── src/
│   │   ├── api/            # API 封装
│   │   ├── views/          # 页面组件
│   │   │   ├── admin/      # 管理后台
│   │   │   └── share/      # 分享页面
│   │   ├── router/         # 路由配置
│   │   └── stores/         # 状态管理
│   └── ...
├── data/                   # SQLite 数据
├── Dockerfile
└── docker-compose.yml
```

## 📝 API 文档

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/s/:code` | 访问分享 |
| POST | `/s/:code/verify` | 验证密码 |
| GET | `/s/:code/download` | 下载文件 |
| GET | `/s/:code/preview` | 预览文件 |
| GET | `/s/:code/files` | 获取文件夹内容 |

### 管理接口（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 登录 |
| GET | `/api/files` | 获取文件列表 |
| POST | `/api/shares` | 创建分享 |
| POST | `/api/shares/batch` | 批量创建分享 |
| GET | `/api/shares/:id/qrcode` | 获取二维码 |
| GET | `/api/stats/overview` | 统计概览 |
| PUT | `/api/settings` | 更新设置 |

## 🔐 安全建议

1. **修改默认密码**: 首次部署后立即修改管理员密码
2. **使用 HTTPS**: 生产环境建议使用反向代理配置 HTTPS
3. **防火墙**: 限制管理端口的访问来源
4. **Token 保护**: `refresh_token` 是敏感信息，请妥善保管

## 📄 License

MIT
