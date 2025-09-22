# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Monica Proxy 是一个将 Monica AI 转换为 ChatGPT 兼容 API 的代理服务，支持完整的 OpenAI 接口兼容性。该项目提供两种 GUI 版本：Fyne 版本（原生桌面应用）和 Wails 版本（Web 风格界面）。

## Architecture

### 核心组件
- **API Server** (`internal/apiserver/`): 基于 Echo 框架的 HTTP 服务器，提供 OpenAI 兼容的 API 端点
- **Monica Client** (`internal/monica/`): 处理与 Monica AI 的通信和请求转换
- **Service Layer** (`internal/service/`): 业务逻辑层，包含聊天、模型、图像、文件等服务
- **Configuration** (`internal/config/`): 配置管理，支持 YAML/JSON 文件和环境变量
- **GUI Frontend** (`frontend/`): 基于 Vue.js + Element Plus 的 Web 界面（Wails 版本）

### 技术栈
- **Backend**: Go 1.23+ with Echo framework
- **Frontend**: Vue.js 3 + Element Plus + Vite
- **GUI Framework**: Wails v2（当前版本）或 Fyne v2（原版）
- **HTTP Client**: Resty for API requests
- **Configuration**: YAML/JSON with environment variable override

## Common Commands

### Wails 版本开发（当前）
```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式（前端热重载）
wails dev

# 构建应用
./build-wails.sh
# 或手动构建
wails build

# 运行构建后的应用
./build/bin/monica-proxy-wails
```

### 依赖管理
```bash
# 整理 Go 依赖
go mod tidy

# 安装前端依赖
cd frontend && npm install
```

### 测试和验证
```bash
# 启动命令行模式
./build/bin/monica-proxy-wails -cli

# 测试 API 端点
curl -H "Authorization: Bearer your_token" http://localhost:8080/v1/models
```

## Configuration

### 配置文件位置
项目按以下优先级加载配置：
1. 用户配置目录：`~/.monica-proxy/config.yaml`
2. 当前目录：`config.yaml`
3. 环境变量
4. 默认值

### 关键配置项
- **Monica.Cookie**: Monica 服务的认证 cookie（必需）
- **Security.BearerToken**: API 认证令牌（必需）
- **Monica.EnableCustomBotMode**: 是否启用自定义机器人模式
- **Monica.BotUID**: 自定义机器人 UID（启用 Custom Bot 模式时必需）

### 环境变量
```bash
export MONICA_COOKIE="your_monica_cookie"
export BEARER_TOKEN="your_api_token"
export BOT_UID="your_bot_uid"  # 可选
export ENABLE_CUSTOM_BOT_MODE=true  # 可选
```

## API Endpoints

### OpenAI 兼容接口
- `POST /v1/chat/completions` - 聊天对话
- `GET /v1/models` - 获取模型列表  
- `POST /v1/images/generations` - 图像生成
- `POST /v1/files` - 文件上传
- `GET /v1/files/:file_id` - 获取文件信息
- `GET /v1/files` - 文件列表
- `DELETE /v1/files/:file_id` - 删除文件

### Custom Bot 接口
- `POST /v1/chat/custom-bot/:bot_uid` - 自定义机器人聊天
- `POST /v1/chat/custom-bot` - 使用配置的 Bot UID 聊天

## Development Notes

### 项目结构特点
- 采用分层架构：API 层 → 服务层 → 数据层
- 支持流式响应（SSE）和非流式响应
- 完整的错误处理和日志记录
- 两种 GUI 版本可切换：Wails（现代化）和 Fyne（轻量级）

### 关键文件
- `main_wails.go`: Wails 版本主程序入口
- `internal/apiserver/router.go`: API 路由定义
- `internal/config/config.go`: 配置管理逻辑
- `internal/service/chat_service.go`: 聊天服务核心逻辑
- `internal/monica/client.go`: Monica API 客户端

### 前端开发
- 使用 Vue 3 Composition API
- Element Plus 组件库
- Pinia 状态管理
- Vue Router 路由管理

### 构建输出
- Wails 版本：`./build/bin/monica-proxy-wails`
- 支持跨平台构建（macOS、Linux、Windows）

## Security Considerations

- 所有 API 请求需要 Bearer Token 认证
- 支持请求限流配置
- 敏感信息日志脱敏
- 支持 TLS 证书验证配置