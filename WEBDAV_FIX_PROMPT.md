# WebDAV 性能修复 Prompt

## 项目背景

这是一个 Go 语言项目，基于 `golang.org/x/net/webdav` 库实现了联通云盘的 WebDAV 挂载功能。当前实现存在严重性能问题：挂载极慢、文件拖拽无响应、视频无法播放。

## 问题诊断（共 5 个根因）

### 根因 1：GET 请求全量代理而非 **临时重定向**

`internal/webdav/file.go:71-104` 的 `Read()` 方法从联通云下载数据后再转发给客户端，数据流为「联通云 → 服务器 → 客户端」，服务器成为带宽瓶颈。

应在 `cmd/server/bootstrap.go:109-120` 的 GET/HEAD 拦截逻辑中，对**文件**（非目录）直接 **307 Temporary Redirect** 重定向到 `GetDownloadURL` 返回的直链，彻底绕过 `x/net/webdav` 的 `ServeContent` 流程。

> 不建议 301：直链通常是短期签名 URL，301 会被客户端/代理缓存，导致后续请求命中失效 URL。

### 根因 2：路径解析无缓存，每次操作都产生 N 次 API 调用

`internal/webdav/helpers.go:29-58` 的 `lookupPath()` 对路径的每一级都调用 `findChildByName()`，后者每次都执行完整的 `ListFiles` API 请求。深度为 4 的路径需要 4 次串行 API 调用。且完全没有缓存，相同路径反复请求。

应增加带 TTL 的缓存层（**只缓存命中结果，不缓存 NotFound**）：
- 目录列表缓存：`dirID → []*FileInfo`，TTL 30-60 秒
- 路径到 ID 映射缓存：`path → (id, parentID, *FileInfo)`，TTL 30-60 秒
- 写操作（Mkdir/RemoveAll/Rename/Close-with-upload）时清除相关缓存

### 根因 3：`Stat()` 方法有严重冗余调用

`internal/webdav/filesystem.go:138-179` 的 `Stat()` 方法：

```go
func (fs *WopanFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
    _, _, info, err := fs.lookupPath(ctx, name)    // ① 完整路径解析（已获得 info）
    // ...但 info 被忽略了...
    parentID, err := fs.getFileIDByPath(ctx, path.Dir(name))  // ② 再解析父路径
    f, err := fs.findChildByName(ctx, parentID, fileName)      // ③ 再列出父目录
}
```

`lookupPath` 返回的 `info` 已经包含完整的 `FileInfo`（Name/Size/IsDir/ModTime），但被丢弃了，后面又花了两次 API 调用重新获取。应直接使用 `lookupPath` 返回的 `info` 构造 `os.FileInfo`。

### 根因 4：`OpenFile()` 重复调用 `lookupPath`

`internal/webdav/filesystem.go:46-67` 的 `OpenFile()` 先调 `Stat()`（内部做 lookupPath），然后调 `openExistingFile()`（`helpers.go:126` 又做 lookupPath）。同一路径被解析了 2-3 遍。

应重构 `OpenFile` 流程：只做一次 `lookupPath`，将结果 `(id, parentID, info)` 直接传递给后续逻辑，避免重复解析。

### 根因 5：全局互斥锁阻塞所有并发操作

`internal/wopan/client.go` 的所有方法（ListFiles/GetDownloadURL/UploadFile/Delete/Move/Rename）都使用 `c.mu.Lock()` 互斥锁。问题：
- `ListFiles`（只读操作）用 `Lock` 而不是 `RLock`，多个 PROPFIND 请求串行执行
- `UploadFile`（`client.go:371`）在整个上传期间持锁，期间所有其他操作被阻塞，导致 Finder 心跳 PROPFIND 超时，表现为「拖入无反应」

⚠️ **注意**：`wopan-sdk-go` 的 `WoClient` 本身非线程安全（内部字段无锁且会在请求中刷新 token）。因此**不能简单改为 `RLock` 或在上传期间释放同一客户端锁**，否则会引入数据竞争和随机崩溃。

安全的做法是 **“双客户端 + 各自互斥”**：
- 元数据客户端（List/Get/Move/Rename/Delete）和上传客户端（Upload2C）各自串行，互不阻塞；
- 共享 token 存储，**每次操作前把最新 token 同步到对应客户端**；
- SDK 的 `OnRefreshToken` 回调只更新共享 token，不直接触碰另一客户端，避免死锁。

## 需要修改的文件清单

### 1. `cmd/server/bootstrap.go` — GET/HEAD 307 临时重定向

在第 109-120 行的 GET/HEAD 拦截处，增加对文件的处理：

```go
if r.Method == http.MethodHead || r.Method == http.MethodGet {
    reqPath := strings.TrimPrefix(r.URL.Path, "/dav")
    if reqPath == "" {
        reqPath = "/"
    }
    fi, err := davFS.Stat(r.Context(), reqPath)
    if err == nil {
        if fi.IsDir() {
            w.Header().Set("Content-Type", "httpd/unix-directory")
            w.Header().Set("Content-Length", "0")
            w.WriteHeader(http.StatusOK)
            return
        }
        // 文件：307 临时重定向到直链
        downloadURL, err := wopanClient.GetDownloadURL(/* 需要 fileID */)
        if err == nil {
            http.Redirect(w, r, downloadURL, http.StatusTemporaryRedirect)
            return
        }
    }
}
```

注意：当前 `Stat` 返回的是 `os.FileInfo`，不含 `fileID`。需要在 `WopanFS` 上增加一个方法（如 `ResolveFileID(path) (fileID string, err error)`）来获取文件 ID，或者直接调用 `lookupPath`（需要将其导出或在 `WopanFS` 上封装）。

### 2. `internal/webdav/helpers.go` — 增加缓存层

在 `WopanFS` 结构体中增加缓存字段：

```go
type WopanFS struct {
    client *wopan.Client

    // 缓存：目录内容
    dirCacheMu sync.RWMutex
    dirCache   map[string]*dirCacheEntry  // key: dirID

    // 缓存：路径→ID 映射
    pathCacheMu sync.RWMutex
    pathCache   map[string]*pathCacheEntry  // key: clean path
}

type dirCacheEntry struct {
    files   []*model.FileInfo
    expires time.Time
}

type pathCacheEntry struct {
    id       string
    parentID string
    info     *model.FileInfo
    expires  time.Time
}
```

所有对 `findChildByName` 和 `lookupPath` 的调用应先查缓存。写操作时清除受影响的缓存条目。

### 3. `internal/webdav/filesystem.go` — 修复 `Stat` 冗余

```go
func (fs *WopanFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
    name = path.Clean(name)
    if name == "/" || name == "." {
        return &fileInfo{name: "/", mode: os.ModeDir | 0755, modTime: time.Now(), isDir: true}, nil
    }

    _, _, info, err := fs.lookupPath(ctx, name)
    if err != nil {
        return nil, err
    }
    if info == nil {
        return nil, os.ErrNotExist
    }

    // 直接使用 lookupPath 返回的 info，不再重复查询
    modTime := info.ModTime
    if modTime.IsZero() {
        modTime = time.Now()
    }
    return &fileInfo{
        name:    info.Name,
        size:    info.Size,
        mode:    fs.getFileMode(info),
        modTime: modTime,
        isDir:   info.IsDir,
    }, nil
}
```

### 4. `internal/webdav/filesystem.go` — 重构 `OpenFile` 消除重复路径解析

```go
func (fs *WopanFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
    name = path.Clean(name)

    id, parentID, info, err := fs.lookupPath(ctx, name)
    if err != nil && !os.IsNotExist(err) {
        return nil, err
    }

    // 文件不存在且需要创建
    if info == nil {
        if (flag & os.O_CREATE) == 0 {
            return nil, os.ErrNotExist
        }
        // createFile 也不需要再次解析父目录，因为 parentID 可以从上面获取
        // 但 lookupPath 对于不存在的文件会返回 err，所以需要单独解析父目录
        return fs.createFile(ctx, name, flag, perm)
    }

    // 目录
    if info.IsDir {
        return fs.openDirectory(ctx, name)
    }

    // 已存在的文件 — 直接传入 id 和 parentID，不再重复 lookupPath
    return fs.openExistingFileWithIDs(ctx, name, id, parentID, info, flag)
}
```

新增 `openExistingFileWithIDs` 方法替代原 `openExistingFile`，接收已解析的 id/parentID，不再调用 `lookupPath`。

### 5. `internal/wopan/client.go` — 双客户端 + 共享 token

**目标**：避免上传阻塞元数据请求，同时不破坏 SDK 线程安全。

实现要点：
- `Client` 内维护 `metaClient` 与 `uploadClient` 两个 `WoClient`；
- `metaMu` 与 `uploadMu` 分别串行化各自客户端调用；
- `accessToken/refreshToken` 存在共享存储中；
- 每次操作前调用 `syncClientTokens(client)`，把共享 token 写回当前客户端；
- `OnRefreshToken` 只更新共享 token（不触碰另一客户端实例）。

## 修改优先级

1. **P0 - 修复 Stat 冗余**（文件：`filesystem.go`）— 改动最小，立竿见影，API 调用减少 60%+
2. **P0 - 修复 OpenFile 重复解析**（文件：`filesystem.go` + `helpers.go`）— 与 P0-1 一起做
3. **P1 - GET/HEAD 307 临时重定向**（文件：`bootstrap.go` + `filesystem.go`）— 解决下载慢和视频不能播放
4. **P1 - 增加缓存层**（文件：`helpers.go`）— 解决 PROPFIND 慢和整体挂载慢
5. **P2 - 双客户端并发**（文件：`client.go`）— 上传不再阻塞元数据请求

## 约束

- 保持 `golang.org/x/net/webdav` 库的 `FileSystem` 接口兼容
- 不修改 `internal/wopan/client.go` 的公开 API 签名
- 缓存 TTL 不宜过长（30-60 秒），避免数据不一致
- 写操作后必须清除相关缓存
