# WoOpen

> 联通云盘直链分享工具。生成一个链接，别人点开就能下载，完事。

<br>

## 这玩意怎么来的

我手上有个联通云盘会员，还有 11 年才到期，别问怎么来的，问就是一元一年买多。

平时根本用不上，放着吃灰又可惜。

想着能不能搞个工具分享文件给朋友？试了 Alist、Cloudreve 这些，感觉不太对。

我就一个联通云盘，用那些功能太多了，而且它们更像是「个人网盘站」，打开能看到文件列表和其他分享，我不想暴露这些，加密又太麻烦了。

我的需求很简单：发个链接，对方只能看到我分享的这一个文件，其他啥也看不到（类似于百度网盘）。

所以就自己写了一个。

<br>

## 功能

| 功能 | 说明 |
| --- | --- |
| 直链分享 | 文件/文件夹生成分享链接 |
| 访问控制 | 可以加密码、设过期时间、限制下载次数 |
| 自定义短链 | 比如 `/s/movie` |
| 访问统计 | 后台能看访问量 |
| 过期提醒 | Token 快过期会推送通知，支持 Bark、微信、Telegram 这些 |
| 文件上传 | 通过 API 上传文件到云盘 |
| WebDAV | 把云盘挂载成网络驱动器，像本地硬盘一样用 |

<br>

## 预览

**登录页面**

![登录页面](./docs/images/login.png)

**后台管理**

![后台管理](./docs/images/image.png)

**分享页面**

![分享页面](./docs/images/share.png)

<br>

## 部署

### 先拿到联通云盘的 Token

打开 https://pan.wo.cn/ 登录账号

按 `F12` 打开开发者工具，切到「网络」标签，刷新页面

找到带 `dispatcher` 的请求：

点进去看响应，找到 `token` 字段，复制出来：

![复制token](./docs/images/find-token.png)

> 最新页面通常只显示一个 UUID 形式的 Token，它是 Access Token。

<br>

### 方式一：拉镜像（推荐）

SSH 连上服务器：

```bash
mkdir -p /opt/woopen && cd /opt/woopen
```

创建 `docker-compose.yml`：

```bash
nano docker-compose.yml
```

粘贴以下内容，**密码记得改**：

```yaml
version: '3.8'

services:
  woopen:
    image: ghcr.io/wp-x/woopen:latest
    container_name: woopen
    ports:
      - "10010:10010"
    volumes:
      - ./data:/data
    environment:
      - WOOPEN_ADMIN_PASSWORD=改成你的密码
      - WOOPEN_JWT_SECRET=生成一个至少 32 位的随机密钥
      - WOOPEN_PORT=10010
      - TZ=Asia/Shanghai
    restart: unless-stopped
```

保存退出，启动：

```bash
docker-compose up -d
```

浏览器打开 `http://服务器IP:10010`

<br>

### 方式二：本地构建

适合想改代码的，把仓库拉下来构建：

```bash
cd /opt
git clone https://github.com/wp-x/woopen.git
cd woopen
```

改一下 `docker-compose.yml` 里的密码，然后：

```bash
docker-compose up -d --build
```

浏览器打开 `http://服务器IP:10010`

<br>

### 填 Token

登录后台，进「系统设置」，找到「刷新令牌」和「访问令牌」两个框。

如果只拿到一个 UUID Token，把它填入 **Access Token**，Refresh Token 留空。

如果拿到的是响应字段中明确标记的 `refresh_token`，则填入 **Refresh Token**；系统会自动刷新并保存新的 Access Token。为兼容旧配置，Refresh Token 刷新返回 `8005` 时也会自动按 Access Token 验证。

保存，去「文件管理」看看能不能加载出文件列表，能的话就成了。

<br>

## 环境变量

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| `WOOPEN_ADMIN_PASSWORD` | 登录密码（至少 12 位） | 必填 |
| `WOOPEN_JWT_SECRET` | JWT 密钥（至少 32 位随机字符） | 必填 |
| `WOOPEN_PORT` | 端口 | 10010 |
| `WOOPEN_SITE_URL` | 域名 | 空 |

<br>

## Nginx 反代

有域名的话可以配一下：

```nginx
server {
    listen 80;
    server_name pan.example.com;

    location / {
        proxy_pass http://127.0.0.1:10010;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

HTTPS 用 certbot 或者套 Cloudflare。

<br>

## 新功能使用说明

### 文件上传 API

可以通过 API 上传文件到云盘，适合自动化脚本或者第三方工具集成。

**上传文件**

```bash
curl -X POST http://localhost:10010/api/files/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@/path/to/file.txt" \
  -F "parent_id=0"
```

**创建目录**

```bash
curl -X POST http://localhost:10010/api/files/mkdir \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "新文件夹", "parent_id": "0"}'
```

> `parent_id` 填 `0` 表示根目录，填其他 ID 就是在那个目录下创建。

**获取 Token**

登录后台后，按 `F12` 打开开发者工具，在「网络」标签里找到任意请求，看请求头里的 `Authorization` 字段，复制 `Bearer` 后面的内容。

<br>

### WebDAV 挂载

可以把云盘挂载成网络驱动器，直接在文件管理器里操作文件。

**挂载地址**

```
http://你的服务器IP:10010/dav/
```

**认证方式**

WebDAV 使用 **Basic Auth**（账号固定为 `admin`，密码为你配置的后台管理员密码）。

**Windows 挂载**

打开「此电脑」，点击「映射网络驱动器」，输入：

```
\\你的服务器IP@10010\dav
```

或者用命令行：

```cmd
net use Z: http://你的服务器IP:10010/dav/ /user:admin 你的密码
```

**macOS 挂载**

打开 Finder，按 `Command + K`，输入：

```
http://你的服务器IP:10010/dav/
```

输入用户名 `admin` 和你设置的密码。

**Linux 挂载**

安装 davfs2：

```bash
sudo apt install davfs2
```

挂载：

```bash
sudo mount -t davfs http://你的服务器IP:10010/dav/ /mnt/webdav
```

输入用户名 `admin` 和密码。

> **注意**：WebDAV 的写入实现为“写入到临时文件，关闭时一次性上传”。大文件上传会占用磁盘临时空间。

<br>

## FAQ

**Token 多久过期？**

不固定，设置里可以开监控，过期了会推送通知。

**两个 token 为啥填一样的？**

联通云盘接口设计如此。

**支持其他网盘吗？**

不支持，只能联通云盘。

<br>

## 参考项目

| 项目 | 说明 |
| --- | --- |
| [Alist](https://github.com/alist-org/alist) | 支持多种网盘的文件列表程序，功能全面 |
| [OpenList](https://github.com/OpenListTeam/OpenList) | Alist 的活跃 Fork，持续维护中 |

<br>

## License

MIT
