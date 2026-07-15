# 实施计划：直连功能（图床 / 视频床 / 文件床）

> 本文档是完整的执行计划，按阶段顺序实施。所有文件路径、字段名、函数名均与当前代码库核对过，
> 执行时不需要做任何架构决策。每个阶段末尾有验收标准，全部通过才能进入下一阶段。

## 背景与目标

WoOpen 现有分享链路：`/s/{code}` → 分享页 → 点下载 → `DownloadShare`（`internal/handler/handler.go:705`）
实时调联通 `GetDownloadUrlV2` 换临时签名 URL → `302` 重定向。

新增**直连模式**：一个稳定 URL `GET /f/{code}`，可直接嵌入 `<img>` / `<video>` / Markdown，
访问时服务端换取联通临时直链并 302。带防盗链（Referer 白名单 + Sec-Fetch-Dest 判断）、
IP 限速、直链缓存、占位图。

**范围外（明确不做）**：签名 URL、服务端代理流量模式、独立图床页面。

## 核心行为规则（防盗链判定，务必按此实现）

对 `GET /f/:code` 的每个请求，按顺序判定：

```
1. 分享不存在 / is_active=false / 已过期 / 达到次数上限 → 返回占位图（410/403 语义见下）
2. Sec-Fetch-Dest 头存在且值为 "document"            → 放行（用户主动打开，永远允许）
3. Referer 头为空：
   - settings.direct_allow_empty_referer = true（默认）→ 放行
   - false                                            → 拒绝
4. Referer 非空：
   - settings.direct_referer_whitelist 为空            → 放行（不限制）
   - Referer 的 host 匹配白名单任一条目（含子域后缀匹配，如 example.com 匹配 blog.example.com）→ 放行
   - 否则                                              → 拒绝
5. 拒绝时：settings.direct_reject_placeholder = true（默认）→ 返回 403 + 占位 SVG 图
             false → 返回纯 403 JSON
```

IP 限速独立于上述规则，作为 Gin 中间件套在 `/f/:code` 路由上：
每 IP 每分钟超过 `settings.direct_rate_limit`（默认 60，0 = 不限）次 → 429 + 占位图。

直连分享**不支持密码**（嵌入场景无法交互输密码），创建时若 `is_direct=true` 则忽略 password 字段。

---

## 阶段 0：实测联通直链有效期（先做，决定缓存 TTL）

1. 用现有分享创建一个文件的下载链接，取得 302 后的联通 CDN URL。
2. 写一个临时脚本（或手动 curl），在第 0/5/15/30/60/120 分钟各请求一次该 URL（HEAD 即可），记录何时开始返回 403/过期。
3. 同时验证两件事并记录在下方：
   - 联通 CDN URL 是否检查 Referer（带一个陌生 Referer 请求一次）
   - 是否支持 `Range: bytes=0-1023` 请求（决定视频拖进度条是否可用）
4. 将 `DirectURLCacheTTL` 设为实测有效期的 70%，向下取整到分钟。若实测有效期 < 3 分钟，TTL 设为 0（不缓存）。

**实测结果记录（执行时填写）**：
- 直链有效期：____ 分钟
- Referer 检查：有 / 无
- Range 支持：是 / 否
- 采用的缓存 TTL：____

> 若实测发现联通 CDN 检查 Referer 导致 `<img>` 内嵌失败，停止执行本计划，回报此结论（需要改为代理模式，属范围外）。

---

## 阶段 1：数据层

### 1.1 `internal/repository/database.go`

在现有 migration 列表末尾追加（照抄现有 `ALTER TABLE ... ADD COLUMN` 风格，忽略"duplicate column"错误的机制已存在）：

```sql
ALTER TABLE shares ADD COLUMN is_direct BOOLEAN DEFAULT 0
ALTER TABLE settings ADD COLUMN direct_referer_whitelist TEXT DEFAULT ''
ALTER TABLE settings ADD COLUMN direct_allow_empty_referer BOOLEAN DEFAULT 1
ALTER TABLE settings ADD COLUMN direct_rate_limit INTEGER DEFAULT 60
ALTER TABLE settings ADD COLUMN direct_reject_placeholder BOOLEAN DEFAULT 1
```

### 1.2 `internal/model/` 模型

`Share` 结构体（现有定义含 ID/ShareCode/.../MaxDownloads）追加：

```go
IsDirect bool `json:"is_direct"` // 直连模式（图床/外链），不支持密码
```

`Settings` 结构体追加：

```go
DirectRefererWhitelist  string `json:"direct_referer_whitelist"`   // 逗号分隔域名，空=不限制
DirectAllowEmptyReferer bool   `json:"direct_allow_empty_referer"` // 默认 true
DirectRateLimit         int    `json:"direct_rate_limit"`          // 次/分钟/IP，0=不限
DirectRejectPlaceholder bool   `json:"direct_reject_placeholder"`  // 拒绝时返回占位图
```

`CreateShareRequest` 追加 `IsDirect bool \`json:"is_direct"\``。

### 1.3 `internal/repository/share.go` 与 `settings.go`

- share 的 Create / GetByCode / List / Update 的 SQL 列清单中加入 `is_direct`。
- settings 的读写 SQL 加入四个新列。
- 保持现有函数签名不变，只扩列。

**验收**：`go build ./...` 通过；删除本地 `data/*.db` 后启动服务，表结构含新列（`sqlite3 data/woopen.db '.schema shares'` 验证）；旧库启动时 migration 不报错。

---

## 阶段 2：直链缓存

新建 `internal/handler/direct_cache.go`，完全仿照 `internal/handler/list_cache.go` 的 TTL 缓存模式：

```go
// key: 联通文件ID(string) → value: {downloadURL string, expires time.Time}
// 接口：Get(fileID string) (string, bool) / Set(fileID string, url string)
// TTL 常量 DefaultDirectURLCacheTTL 用阶段 0 的实测值；TTL<=0 时 Get 恒 miss
```

在 `Handler` 结构体（`internal/handler/handler.go`）加字段 `directCache`，
在 `NewHandler` 中初始化。

**验收**：单元测试 `internal/handler/direct_cache_test.go`——命中、过期失效、TTL=0 恒 miss 三个用例通过。

---

## 阶段 3：直连 Handler 与防盗链

### 3.1 新建 `internal/handler/direct.go`

实现 `func (h *Handler) DirectLink(c *gin.Context)`，逻辑：

1. 按 `code` 查 share（复用 `h.shareRepo.GetByCode`），要求 `IsDirect == true`。
   非直连分享访问 `/f/` 一律按不存在处理（避免绕过密码）。
2. 有效性检查：`is_active`、`expire_at`、`max_downloads`（判断逻辑照抄 `DownloadShare`，
   但失败时返回**占位图**而非 JSON，HTTP 状态：不存在 404、过期 410、超限 403）。
3. 防盗链判定：按文档开头的"核心行为规则"实现，抽成独立纯函数便于测试：
   ```go
   // internal/handler/direct.go
   type hotlinkPolicy struct {
       whitelist        []string // 已按逗号 split + trim + 小写
       allowEmptyReferer bool
   }
   func allowRequest(secFetchDest, referer string, p hotlinkPolicy) bool
   ```
   Referer 匹配：`url.Parse(referer)` 取 Hostname 小写，与白名单条目做
   `host == entry || strings.HasSuffix(host, "."+entry)`。Parse 失败按拒绝处理。
4. 取直链：先查 `directCache`，miss 则 `h.wopanClient.GetDownloadURL(share.TargetID)` 并写缓存。
5. 记录访问：`h.shareRepo.IncrementDownloadCount(share.ID)` 仅在**缓存 miss** 时调用
   （引用计数语义，避免同一页面多次加载灌水）；`accessLogRepo.Create` 的 Action 用 `"direct"`。
6. `c.Redirect(http.StatusFound, downloadURL)`。

### 3.2 占位图

`direct.go` 内定义一个函数返回内联 SVG（约 300 字节，灰底 + 文字），文字按场景：
`链接已失效` / `仅限授权站点引用` / `请求过于频繁`。响应头 `Content-Type: image/svg+xml`、
`Cache-Control: no-store`。不引入任何图片资源文件。

### 3.3 IP 限速中间件

新建 `internal/middleware/ratelimit.go`：固定窗口计数器（`map[ip]count`，每分钟整点重置，
`sync.Mutex` 保护），构造函数接收 `func() int` 动态读取当前设置值（settings 可在运行时被改）。
超限返回 429 + 占位图。

> ponytail: 固定窗口 + 全局锁足够自用规模；需要精确平滑再换令牌桶。

### 3.4 路由注册 `cmd/server/routes.go`

在 `registerShareRoutes` 附近新增：

```go
r.GET("/f/:code", middleware.DirectRateLimit(getLimit), h.DirectLink)
```

**验收**：
- 单元测试 `internal/handler/direct_test.go` 覆盖 `allowRequest` 全部分支
  （document 放行 / 空 Referer 开关两态 / 白名单命中·子域·未命中 / 白名单为空 / referer 解析失败）。
- 手工验证（服务跑起来后 curl）：
  ```
  curl -I localhost:8080/f/{code}                                    # 302
  curl -I -H 'Referer: https://evil.com' localhost:8080/f/{code}     # 白名单非空时 403+svg
  curl -I -H 'Referer: https://evil.com' -H 'Sec-Fetch-Dest: document' ... # 302
  连续 61 次请求                                                       # 第 61 次 429
  ```

---

## 阶段 4：创建/更新分享支持直连

`internal/handler/handler.go`：

- `CreateShare`：读取 `req.IsDirect` 写入 share；若 `IsDirect` 为 true，强制 `Password = ""`。
- `UpdateShare`：同上规则；允许把普通分享改为直连（同时清空密码），反向也允许。
- `GetSettings` / `UpdateSettings`：透传四个新设置字段，参考现有字段的处理方式
  （注意 `UpdateSettings` 里 token 有掩码逻辑，新字段不需要掩码）。

**验收**：`POST /api/shares` 带 `"is_direct": true, "password": "123"` → 落库 password 为空；
settings 四字段可读可写、重启后保留。

---

## 阶段 5：前端

前端 API 层：`web/src/api/index.ts` 中 share 与 settings 相关的类型/请求体加上新字段，与后端 JSON 字段名一致。

### 5.1 `web/src/views/admin/Files.vue`（创建分享弹窗，单个 + 批量两处表单）

- `shareForm` / `batchShareForm` 增加 `is_direct: false`。
- 表单加一个开关行「直连模式（图床/外链嵌入）」，风格沿用现有 brutalist 类名
  （`brutal-label-tag` 等，照抄相邻控件的结构）。
- `is_direct = true` 时：密码输入框 `disabled` 并显示提示"直连模式不支持密码"。
- 创建成功弹窗（`shareResult` 区域）：直连分享显示链接为 `{baseURL}/f/{code}`，
  并提供三个复制按钮：
  - URL：`https://host/f/code`
  - Markdown：`![文件名](https://host/f/code)`
  - HTML：`<img src="https://host/f/code" alt="文件名">`
  复制实现照抄页面现有的复制逻辑。
- **上传完成后的结果区域**：加「创建直连并复制」按钮 —— 用上传返回的文件信息直接调
  `POST /api/shares`（`is_direct: true`，其余默认），成功后把 URL 写入剪贴板并 toast。

### 5.2 `web/src/views/admin/Shares.vue`

- 列表行加徽标：`is_direct ? '直连' : '分享'`（沿用现有 tag 样式）。
- 直连行的"下载次数"列表头文案对直连语义显示为"引用次数"（若列是共享表头，
  则在数值旁加 tooltip 说明，二选一，以实现简单者为准）。
- 直连行的"复制链接"按钮复制 `/f/{code}` 而非 `/s/{code}`。
- 编辑弹窗同步支持 `is_direct` 开关（规则同 Files.vue）。

### 5.3 `web/src/views/admin/Settings.vue`

新增「直连设置」卡片（结构照抄现有监控设置卡片），四个控件：

| 控件 | 字段 | 默认 |
|---|---|---|
| 文本框：Referer 白名单（逗号分隔，空=不限制） | `direct_referer_whitelist` | 空 |
| 开关：允许空 Referer | `direct_allow_empty_referer` | 开 |
| 数字：每 IP 限速（次/分钟，0=不限） | `direct_rate_limit` | 60 |
| 开关：拒绝时返回占位图 | `direct_reject_placeholder` | 开 |

**验收**：`npm run build`（web 目录）无 TS 错误；手工过一遍：
创建直连 → 三格式复制内容正确 → 列表徽标正确 → 设置保存后刷新仍在 → 上传后一键直连可用。

---

## 阶段 6：端到端验证

1. `go test ./...` 全绿，`go build ./...` 通过，web `npm run build` 通过。
2. 真实文件验证：上传一张图片 → 创建直连 → 在本地写一个最小 HTML 页
   `<img src="http://localhost:8080/f/xxx">` 用浏览器打开，图片正常显示。
3. 视频文件同样验证 `<video>` 播放与进度条拖动（依赖阶段 0 的 Range 实测结论）。
4. 白名单设为 `example.com` 后，上述本地 HTML（Referer 是 localhost）应显示占位图；
   直接在地址栏打开 `/f/xxx` 仍正常（Sec-Fetch-Dest: document）。
5. 普通分享 `/s/{code}` 全流程回归：密码、下载、预览均不受影响。

## 注意事项

- **不要改动** `DownloadShare` / `PreviewShare` / WebDAV 的任何现有逻辑，直连是纯新增路径。
- 所有新增 Go 代码遵循项目现有风格（中文注释、gin 直接返回 `model.APIResponse`）。
- 占位图/错误信息中不得包含内部错误详情（如联通 API 报错原文）。
- commit 拆分建议：阶段 1+2 一个 commit（`feat: 直连数据层与直链缓存`），
  阶段 3+4 一个（`feat: 直连路由、防盗链与限速`），阶段 5 一个（`feat: 直连管理界面`）。
  不加任何 AI 署名。
