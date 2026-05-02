# lihan_web_music_player

[![OpenToolHub v3](https://img.shields.io/badge/OpenToolHub-v3-2563eb?logo=gitee&labelColor=111827)](https://gitee.com/lihan3238/OpenToolHub/blob/develop/docs/developer_tool_guide_zh.md)

一个基于 Go + Gin 的本地音乐播放器，支持本地曲库、歌词显示、音频 Range 请求，并可作为 OpenToolHub Tool Protocol v3 的真实 WEB_APP 验收工具运行。

## 功能

- 播放本地音乐文件：`mp3`、`m4a`、`mp4`、`ogg`、`wav`、`flac`
- 自动加载同名歌词文件：`.lrc`、`.txt`
- 支持顺序播放、单曲循环、随机播放
- 支持音量调节和当前歌曲高亮
- 支持多个音乐目录合并扫描：`music/`、`musics/`、`MUSIC_DIR`
- 使用 `http.ServeFile` 提供音频文件，支持浏览器 Range 请求
- 支持两种运行模式：
  - `standalone`：原始独立播放器模式，可直接访问
  - `toolbox`：OpenToolHub 接入模式，业务页面、API、静态资源和音乐资源需要平台 JWT 才能访问
- 已通过 `opentoolhub.tool.v3` WEB_APP 接入验证，可作为真实外部项目接入样例

## 快速开始

```bash
git clone https://github.com/lihan3238/lihan_web_music_player.git
cd lihan_web_music_player
go mod tidy
go run .
```

访问：

```text
http://localhost:8080
```

## 曲库目录

默认会扫描：

```text
music/
musics/
```

也可以通过 `MUSIC_DIR` 指定外部曲库：

```bash
MUSIC_DIR=/path/to/musics go run .
```

Windows PowerShell：

```powershell
$env:MUSIC_DIR="C:\lihan_work\github_repos\music\musics"
go run .
```

如果要配置多个目录，使用系统路径分隔符：

- Windows：`;`
- Linux/macOS：`:`

## 端口配置

默认端口是 `8080`。

命令行参数：

```bash
go run . -port 9000
```

环境变量：

```bash
PORT=9000 go run .
```

优先级：

```text
命令行参数 -port > 环境变量 PORT > 默认值 8080
```

## OpenToolHub Tool Protocol v3 接入

本项目可以作为 OpenToolHub 的真实 WEB_APP 工具运行，用来验证平台的 v3 协议能力。

上方 Badge 表示本项目支持 `opentoolhub.tool.v3`。它是静态展示标识，不代表实时运行状态；只有实际实现协议并通过 OpenToolHub 校验的项目才建议添加。

### 协议端点

公开端点：

```text
GET /health
GET /meta
```

`/meta` 返回：

- `protocol_version: opentoolhub.tool.v3`
- `tool_type: WEB_APP`
- `source.repository_url`
- `source.license`
- `source.deploy`
- `auth.type: platform_jwt`
- `validation.web_probe_paths`
- `web_entry_url: /`

toolbox 模式下受保护端点：

```text
GET /
GET /static/*
GET /api/songs
GET /api/lyrics/:name
GET /music/:file
```

这些端点必须通过 OpenToolHub 代理访问。直接无 token 访问会返回 `401` 或 `403`。

### toolbox 模式启动

Windows PowerShell 示例：

```powershell
$env:PORT="18080"
$env:OPENTOOLHUB_MODE="toolbox"
$env:OPENTOOLHUB_TOOL_ID="<OpenToolHub 中的工具 ID>"
$env:OPENTOOLHUB_ISSUER="opentoolhub"
$env:OPENTOOLHUB_AUDIENCE="opentoolhub-runtime"
$env:RUNTIME_JWT_SECRET="<与 OpenToolHub RUNTIME_JWT_SECRET 一致的密钥>"
$env:MUSIC_DIR="C:\lihan_work\github_repos\music\musics"
go run .
```

WSL/Linux 示例：

```bash
PORT=18080 \
OPENTOOLHUB_MODE=toolbox \
OPENTOOLHUB_TOOL_ID="<OpenToolHub 中的工具 ID>" \
OPENTOOLHUB_ISSUER=opentoolhub \
OPENTOOLHUB_AUDIENCE=opentoolhub-runtime \
RUNTIME_JWT_SECRET="<与 OpenToolHub RUNTIME_JWT_SECRET 一致的密钥>" \
MUSIC_DIR=/mnt/c/lihan_work/github_repos/music/musics \
go run .
```

兼容说明：代码仍支持旧变量 `OPENTOOLHUB_JWT_SECRET`，但推荐使用 OpenToolHub 文档中的标准变量名 `RUNTIME_JWT_SECRET`。

### 配置项

| 变量 | 说明 | 示例 |
|------|------|------|
| `PORT` | runtime 监听端口 | `18080` |
| `OPENTOOLHUB_MODE` | `standalone` 或 `toolbox` | `toolbox` |
| `OPENTOOLHUB_TOOL_ID` | OpenToolHub 创建出的工具 ID | `cmo...` |
| `OPENTOOLHUB_ISSUER` | runtime JWT issuer | `opentoolhub` |
| `OPENTOOLHUB_AUDIENCE` | runtime JWT audience | `opentoolhub-runtime` |
| `RUNTIME_JWT_SECRET` | 与 OpenToolHub 一致的 runtime JWT 密钥 | `<strong-secret>` |
| `MUSIC_DIR` | 外部曲库目录，可配置多个 | `C:\lihan_work\github_repos\music\musics` |

### OpenToolHub 登记信息

在 OpenToolHub 开发者中心登记：

```text
工具类型：WEB_APP
Runtime URL：http://localhost:18080
源码仓库：https://github.com/lihan3238/lihan_web_music_player
许可证：MIT
部署说明：使用 OPENTOOLHUB_MODE=toolbox 启动，并配置 MUSIC_DIR 与 RUNTIME_JWT_SECRET
```

如果 OpenToolHub 运行在 WSL，而本项目 runtime 运行在 Windows PowerShell，不要在 OpenToolHub 中填写 `localhost`。应填写 WSL 可访问的 Windows host IP，例如：

```text
http://172.23.80.1:18080
```

可在 WSL 中通过 `cat /etc/resolv.conf` 查看 nameserver 作为 Windows host IP 参考。

完成校验和激活后，从 OpenToolHub 市场详情页打开工具。浏览器 URL 应保持在 OpenToolHub 站内，例如：

```text
http://localhost:3000/tools/<tool-id>/app
```

播放器页面、歌曲列表、歌词和音频文件都应通过 OpenToolHub 反向代理访问。

## 本地验证

```bash
go test ./...
go build ./...
```

toolbox 模式下建议额外验证：

- 无 token 请求 `/api/songs` 应失败
- 带 OpenToolHub runtime JWT 请求 `/api/songs` 应成功
- 带 `Range: bytes=0-3` 请求 `/music/:file` 应返回 `206 Partial Content`

从 OpenToolHub 完整验证：

1. 启动 OpenToolHub。
2. 启动本项目 toolbox runtime。
3. 在 OpenToolHub 开发者中心创建 `WEB_APP` 工具并填写 runtime URL、源码仓库、许可证、部署说明。
4. 校验并激活工具。
5. 从工具市场详情页打开 `/tools/<tool-id>/app`。
6. 确认歌曲列表、歌词和音频播放都通过 OpenToolHub 站内 URL 加载。

## 项目结构

```text
lihan_web_music_player/
├── main.go
├── templates/
│   └── index.html
├── static/
│   ├── style.css
│   └── cover.png
├── music/
├── musics/
└── README.md
```

## License

MIT License
