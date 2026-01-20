# AI API Proxy

这是一个轻量级的 API 代理服务，旨在统一和简化对各种 AI 服务 API 的访问。它使用 Go 语言编写，支持 Docker 部署。


## 功能特性

- **多平台支持**: 代理了众多流行的 AI 和聊天服务 API。
- **简单路由**: 通过 URL 前缀将请求路由 to 相应的官方 API 端点。
- **Web 界面**: 提供一个简单的 HTML 首页，列出所有支持的 API 路由。
- **隐私保护**: 过滤掉某些可能泄露信息的请求头（如 host, referer, cf- 等）。
- **轻量级**: 基于 Go 语言，编译后体积小，运行资源占用低。

## 支持的 API 映射

该代理将以下本地路径映射到对应的官方 API 地址：

| 路由前缀 | 目标 API 地址 |
| :--- | :--- |
| `/discord` | `https://discord.com/api` |
| `/telegram` | `https://api.telegram.org` |
| `/openai` | `https://api.openai.com` |
| `/claude` | `https://api.anthropic.com` |
| `/gemini` | `https://generativelanguage.googleapis.com` |
| `/meta` | `https://www.meta.ai/api` |
| `/groq` | `https://api.groq.com/openai` |
| `/xai` | `https://api.x.ai` |
| `/cohere` | `https://api.cohere.ai` |
| `/huggingface` | `https://api-inference.huggingface.co` |
| `/together` | `https://api.together.xyz` |
| `/novita` | `https://api.novita.ai` |
| `/portkey` | `https://api.portkey.ai` |
| `/fireworks` | `https://api.fireworks.ai` |
| `/openrouter` | `https://openrouter.ai/api` |
| `/cerebras` | `https://api.cerebras.ai` |

## 快速开始

### 使用 Docker Compose (推荐)

1. 确保已安装 Docker 和 Docker Compose。
2. 克隆本仓库或下载文件。
3. 在项目根目录下运行：

```bash
docker-compose up -d --build
```

服务将在 `http://localhost:7890` 启动。

### 使用预构建镜像部署 (GitHub Packages)

如果不想自己构建，可以直接拉取 GitHub Container Registry 上的镜像：

```bash
# 拉取镜像
docker pull ghcr.io/handsomezhuzhu/api-proxy:latest

# 运行容器
docker run -d -p 7890:7890 --name api-proxy ghcr.io/handsomezhuzhu/api-proxy:latest
```

### 使用 Docker

1. 构建镜像：

```bash
docker build -t api-proxy .
```

2. 运行容器：

```bash
docker run -d -p 7890:7890 --name api-proxy api-proxy
```

### 从源码运行

1. 确保已安装 Go 环境 (推荐 1.21+)。
2. 运行：

```bash
go run main.go
```

或者编译后运行：

```bash
go build -o api-proxy main.go
./api-proxy
```

默认端口为 `7890`。你可以通过命令行参数指定端口：

```bash
./api-proxy 8080
```

## 使用示例

假设你的服务运行在 `http://localhost:7890`。

**访问 OpenAI API:**

你可以将原本发往 `https://api.openai.com/v1/chat/completions` 的请求改为发往：

`http://localhost:7890/openai/v1/chat/completions`

**访问 Claude (Anthropic) API:**

将请求发往：

`http://localhost:7890/claude/v1/messages`

## 🚀 阿里云 ESA (边缘安全加速) 配置指南 (非常重要)

如果你使用了阿里 ESA 加速本服务，**必须**在 ESA 控制台中进行以下设置，否则会出现 `Origin Time-out` (源站超时) 或 AI 回复卡顿（打字机效果失效）的问题。

### 1. 缓存配置 (Cache)
请进入 **站点管理** -> **缓存配置**，添加以下规则：

| 配置项 | 推荐设置 | 说明 |
| :--- | :--- | :--- |
| **边缘缓存过期时间** <br> (Edge Cache TTL) | **不缓存** <br> (或设置为 0秒) | **核心设置**。必须禁止 CDN 节点缓存 AI 的接口响应，否则第二个用户会看到上一个用户的对话，或者直接报错。建议针对 API 目录（如 `/openai/*`）设置。 |
| **浏览器缓存过期时间** <br> (Browser Cache TTL) | **不缓存** | 禁止客户端浏览器缓存接口结果。 |
| **查询字符串** | **保留** (或 遵循源站) | 某些 AI API 使用 URL 参数传递版本号或签名，不可忽略。 |

### 2. 回源配置 (Origin) - 解决超时问题的关键
ESA 默认的连接超时时间较短（通常 30秒），而 AI 模型（特别是推理模型）可能需要 60秒+ 才能生成第一个字。

请进入 **站点管理** -> **回源配置**：

*   **读超时时间 (Read Timeout)**: 修改为 **120秒** 或 **300秒**。
    *   *说明*: 如果不改这个，AI 思考超过 30秒时，ESA 会认为源站挂了，直接切断连接并报 `Origin Time-out`。

### 3. 开发模式 (Debug)
如果配置后仍然有问题，可以暂时开启 **“开发模式”**。这会强制所有请求绕过缓存节点直接回源，用于排查是否是缓存规则导致的问题。

---

## 注意事项

- 本项目仅作为 API 请求的转发代理，请确保你拥有对应服务的有效 API Key。
- 代理会透传你的 Authorization 头（API Key）。
- 首页 (`/`) 提供了一个简单的状态页面，列出所有可用路由。
- **隐私**: 本服务会自动过滤 `X-Forwarded-For` 和 `Host` 头，保护你的源站 IP。日志中会尝试解析 `Ali-Cdn-Real-Ip` 以记录真实访问者 IP。
