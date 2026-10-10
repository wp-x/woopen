# 联通云盘「我的同步盘」接口整理

> 本文档整理自当前项目 `main` 分支及其依赖的 `github.com/OpenListTeam/wopan-sdk-go v0.2.2`。
> 重点记录近期提交中用于“我的同步盘”的个人云接口，以及项目中已经验证过的 `fid` 使用规则。

## 1. 适用范围

本项目通过个人云列表读取接口返回的同步目录；名称保持返回的原值，不要求叫“我的同步盘”。目前没有单独按名称识别同步盘的接口或逻辑。能否列出某个账号的同步目录，仍取决于该账号的目录 ID、空间配置和上游返回数据。

| 项目 | 值 |
| --- | --- |
| 业务空间 | 个人云 / 我的同步盘 |
| `spaceType` | `"0"` |
| 元数据接口 | `POST https://panservice.mail.wo.cn/wohome/dispatcher` |
| 上传接口 | `POST https://tjupload.pan.wo.cn/openapi/client/upload2C`（地址由 `GetZoneInfo` 返回，不能永久写死） |
| `clientId` | `1001000021` |
| 认证 | 请求头 `Accesstoken: <access_token>` |
| 来源头 | `Origin: https://pan.wo.cn`、`Referer: https://pan.wo.cn/` |

元数据请求由联通协议封装为统一的 `dispatcher` 请求。`body.param` 通常是 SDK 加密后的 JSON，不能直接把下面的明文参数当作 `param` 发送；建议复用 SDK 的加密、签名和 token 刷新实现。

## 2. 重要 ID 规则

文件列表同时返回两个不同用途的标识：

| 字段 | 用途 |
| --- | --- |
| `id` | 文件/目录管理操作：删除、重命名、移动、复制等 |
| `fid` | 文件内容和下载直链：`GetDownloadUrlV2`、WebDAV 读取、流式下载 |

**下载必须使用 `fid`。** 最近提交 `aa7e888` 已验证：把列表返回的十六进制 `id` 传给 `GetDownloadUrlV2` 会得到 `rsp_code 1000`，同步盘文件会表现为 502。`fid` 为空时，项目封装才回退使用 `id`。

未配置根目录时使用字符串 `"0"`；配置 `root_folder_id` 时，WebDAV 从该目录 ID 开始解析路径。修改挂载根目录后需重启服务，以重建文件系统和目录缓存。上游不能直接传空字符串，项目会把未指定的目录归一化为配置根目录或 `"0"`。

## 3. 统一请求格式

```json
{
  "header": {
    "key": "QueryAllFiles",
    "channel": "wohome",
    "version": "",
    "resTime": 0,
    "reqSeq": 0,
    "sign": "<由 SDK 生成>"
  },
  "body": {
    "param": "<加密后的 JSON 字符串>",
    "secret": true
  }
}
```

成功响应的外层结构：

```json
{
  "STATUS": "200",
  "RSP": {
    "RSP_CODE": "0000",
    "RSP_DESC": "成功",
    "DATA": "<加密数据、JSON 或空字符串；由 SDK 解密并解析>"
  }
}
```

常见失败码：`8005`、`1001` 表示 token 无效或登录失败；请求失败后应刷新 token，再完整重试一次。不要把 token、签名或加密参数写入日志和文档。

## 4. 文件列表：QueryAllFiles

**底层接口**：`QueryAllFiles`
**项目封装**：`wopan.Client.ListFiles(dirID, page, pageSize)`

明文参数：

```json
{
  "spaceType": "0",
  "parentDirectoryId": "0",
  "pageNum": 0,
  "pageSize": 50,
  "sortRule": 0,
  "clientId": "1001000021"
}
```

注意项目对外页码从 `1` 开始，联通接口的 `pageNum` 从 `0` 开始，因此：`pageNum = max(page - 1, 0)`。

WebDAV 与 OpenList 的目录遍历采用每页 100 条，持续读取到不足一页为止。对支持最大每页 100 条的上游，按 200 条判断最后一页会提前停止，遗漏后续目录。这个返回限制需结合具体账号的实际响应验证。

支持取消的调用为 `ListFilesCtx(ctx, ListFilesOptions{DirID, Page, PageSize})`。内部 `FileInfo.parent_id` 来自请求的目录 ID；上游文件 `fid` 保留在 `FileInfo.fid`，用于下载。

解密后的 `DATA`：

```json
{
  "files": [
    {
      "familyId": 0,
      "fid": "<下载标识>",
      "creator": "186****5244",
      "size": 1599939,
      "createTime": "20220221190017",
      "name": "example.txt",
      "shootingTime": "",
      "id": "<管理标识>",
      "type": 1,
      "thumbUrl": "",
      "fileType": "1"
    }
  ]
}
```

`type=0` 表示目录，`type=1` 表示文件；`createTime` 格式为 `yyyyMMddHHmmss`。

## 5. 获取下载直链：GetDownloadUrlV2

**底层接口**：`GetDownloadUrlV2`
**项目封装**：`wopan.Client.GetDownloadURL(fileID)`

明文参数：

```json
{
  "type": "1",
  "fidList": ["<文件 fid>"],
  "clientId": "1001000021"
}
```

解密后的 `DATA`：

```json
{
  "type": 1,
  "list": [
    {
      "fid": "<文件 fid>",
      "downloadUrl": "https://.../openapi/download?fid=<签名值>"
    }
  ]
}
```

直链带签名和有效期，应按临时 URL 使用，不要持久化或手工拼接 `fid`。项目会对 URL 中的 `fid` 查询参数做编码修正，避免 `+` 等字符被错误解码。

## 6. 创建目录：CreateDirectory

**项目封装**：`wopan.Client.CreateDirectoryCtx(ctx, parentID, dirName)`

明文参数：

```json
{
  "spaceType": "0",
  "familyId": "<默认家庭空间 ID>",
  "parentDirectoryId": "0",
  "directoryName": "new-folder",
  "clientId": "1001000021"
}
```

虽然业务是个人云，当前接口仍使用初始化阶段通过 `FamilyUserCurrentEncode` 查询到的 `familyId`。返回数据：

```json
{ "id": "<新目录 id>" }
```

## 7. 文件和目录管理接口

以下接口均通过 `POST /wohome/dispatcher` 调用，`channel` 为 `wohome`、`spaceType` 为 `"0"`，请求体仍需 SDK 加密。

| 接口 key | 项目封装 | 关键明文参数 |
| --- | --- | --- |
| `RenameFileOrDirectory` | `Rename(ctx, isDir, id, name)` | `type`: 目录 `0` / 文件 `1`，`id` 使用列表 `id`，`name` |
| `MoveFile` | `MoveBatch(ctx, dirList, fileList, targetDirID)` | `dirList`、`fileList`、`targetDirId`、`sourceType:"0"`、`targetType:"0"` |
| `CopyFile` | `CopyBatch(ctx, dirList, fileList, targetDirID)` | 同 `MoveFile` |
| `DeleteFile` | `DeleteBatch(ctx, dirList, fileList)` | `spaceType:"0"`、`vipLevel:"0"`、`dirList`、`fileList` |

管理接口使用 `id`，不要把下载用的 `fid` 混入 `dirList` 或 `fileList`。目录和文件列表可分别传空数组，但至少应提供要操作的一类 ID。

## 8. 上传文件：upload2C

**项目封装**：`wopan.Client.UploadFile(ctx, parentID, fileName, contentType, size, content, onProgress)`

上传不是 `wohome/dispatcher`，而是 multipart 请求。当前 SDK 的关键字段如下：

| 字段 | 说明 |
| --- | --- |
| `uniqueId` | SDK 生成的上传标识，同一文件分片共用 |
| `accessToken` | 当前 access token |
| `psToken` | 当前个人云上传由 SDK 传字符串 `undefined` |
| `channel` | `wocloud` |
| `directoryId` | 目标目录 ID，根目录使用 `0` |
| `fileName` | 文件名 |
| `fileSize` | 文件总大小 |
| `totalPart` | 分片总数 |
| `partSize` | 当前分片大小 |
| `partIndex` | 分片序号，从 `1` 开始 |
| `fileInfo` | 加密后的文件元数据 |
| `file` | 当前分片二进制内容 |

multipart 的 `channel` 为 `wocloud`，但 `fileInfo` 使用 SDK 的 `wohome` 加密方式。`fileInfo` 的明文结构：

```json
{
  "spaceType": "0",
  "directoryId": "0",
  "batchNo": "20260101000000",
  "fileName": "example.bin",
  "fileSize": 123,
  "fileType": "5"
}
```

成功响应示例：

```json
{
  "code": "0000",
  "msg": "上传成功",
  "data": { "fid": "<新文件 fid>" }
}
```

上传返回的是文件 `fid`，项目会同时把它作为新文件的 `ID` 和 `FID` 保存；后续获取直链直接使用该值。

## 9. 项目调用关系

```text
ListFiles
  -> QueryAllFiles(spaceType=0)
  -> 返回 id + fid

GetDownloadURL
  -> GetDownloadUrlV2(fidList=[fid])
  -> 返回临时 downloadUrl

UploadFile
  -> upload2C(spaceType=0)
  -> 返回 fid

CreateDirectory / Rename / Move / Copy / Delete
  -> wohome dispatcher
  -> 管理标识使用 id
```

当前项目相关实现位置：

- `internal/wopan/client_list.go`
- `internal/wopan/client_dirs.go`
- `internal/wopan/client_ops.go`
- `internal/wopan/client_upload.go`
- `internal/wopan/client.go`

## 10. 接入建议

1. 优先复用 `wopan-sdk-go`，不要自行实现加密、签名和 token 刷新。
2. 把 `id`、`fid` 分开建模，字段名不要复用。
3. 下载、预览、WebDAV 读取一律传 `fid`。
4. 根目录使用账号配置的目录 ID；未配置时传 `"0"`，不要直接给上游传空字符串。
5. `GetZoneInfo` 返回的上传域名可能变化，启动时获取并缓存。
6. 不要记录 `Accesstoken`、`sign`、加密后的 `param` 和完整下载直链。
