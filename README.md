# Monica Proxy GUI1

**Monica AI 代理服务**

将 Monica AI 转换为 ChatGPT 兼容的 API，支持完整的 OpenAI 接口兼容性，提供GUI界面进行整体管理。


## 项目来源
本项目基于 [https://github.com/ycvk/monica-proxy](https://github.com/ycvk/monica-proxy) 项目进行二次开发。

## ✨ **功能特性**

### 🔗 **API兼容性**

- ✅ **完整的System Prompt支持** - 通过Custom Bot Mode实现真正的系统提示词
- ✅ **ChatGPT API完全兼容** - 无缝替换OpenAI接口，支持所有标准参数
- ✅ **流式响应** - 完整的SSE流式对话体验，支持实时输出
- ✅ **Monica模型支持** - GPT-4o、Claude-4、Gemini等主流模型完整映射


## 🚀 **快速开始**

### 启动方式

```bash
# 编译
go build -o monica-proxy main.go

# 默认启动GUI配置界面
./monica-proxy

# 启动命令行模式
./monica-proxy -cli
```

### 测试API

```bash
curl -H "Authorization: Bearer your_bearer_token" \
     http://localhost:8080/v1/models
```


## 🏗️ **部署指南**

### 🔧 **源码编译**

```bash
# 克隆项目
git clone https://github.com/SimonUTD/monica-proxy-gui
cd monica-proxy-gui

# 编译
go build -o monica-proxy main.go

# 命令行模式运行
export MONICA_COOKIE="your_cookie"
export BEARER_TOKEN="your_token"
# export BOT_UID="your_bot_uid"  # 可选，用于Custom Bot模式
./monica-proxy -cli

# 或者启动GUI配置界面（默认）
./monica-proxy
```

## ⚙️ **配置参考**

### 🖥️ **GUI配置界面**

Monica Proxy 现在支持图形用户界面配置。程序默认启动GUI模式，可以方便地配置所有环境变量：

- **必填项**：Monica Cookie、Bearer Token（带有*标记）
- **选填项**：其他所有配置项都可以通过GUI界面进行配置
- **保存配置**：点击"保存配置"按钮将配置保存到 `config.yaml` 文件中
- **服务控制**：可以直接在GUI中启动和停止服务

#### 启动GUI模式

```bash
# 方法1：使用Makefile
make run-gui

# 方法2：直接运行（默认启动GUI）
./monica-proxy

# 方法3：编译后运行
go build -o monica-proxy main.go
./monica-proxy
```

#### GUI界面说明

1. **服务器配置**：配置HTTP服务器的主机地址、端口和超时时间
2. **Monica配置**：输入Monica Cookie和Bot UID（启用Custom Bot模式时必需）
3. **安全配置**：设置Bearer Token、TLS验证选项和限流配置
4. **日志配置**：配置日志级别、格式和输出方式
5. **服务控制**：启动/停止HTTP服务

#### 使用步骤

1. 启动程序（默认启动GUI界面）
2. 在相应字段中输入配置信息（必填项带有*标记）
3. 点击"保存配置"按钮将配置保存到config.yaml文件
4. 点击"启动服务"按钮启动HTTP服务
5. 使用"停止服务"按钮可以停止正在运行的服务

### 命令行模式

如果需要以命令行模式运行，可以使用 `-cli` 参数：

```bash
./monica-proxy -cli
```

命令行模式下，程序会直接读取配置文件和环境变量启动HTTP服务，不显示GUI界面。

## 🔌 **API使用**

### 支持的端点

- `POST /v1/chat/completions` - 聊天对话（兼容ChatGPT）
- `GET /v1/models` - 获取模型列表
- `POST /v1/images/generations` - 图片生成（兼容DALL-E）

### 认证方式

```http
Authorization: Bearer YOUR_BEARER_TOKEN
```

### 聊天API示例

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "system", "content": "你是一个有帮助的助手"},
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'
```

### 支持的模型

| 模型系列         | 模型名称                                                                                             | 说明                 |
|--------------|--------------------------------------------------------------------------------------------------|--------------------|
| **GPT系列**    | `gpt-5`, `gpt-4o`, `gpt-4o-mini`, `gpt-4.1`, `gpt-4.1-mini`, `gpt-4.1-nano`, `gpt-4-5`           | OpenAI GPT模型       |
| **Claude系列** | `claude-4-sonnet`, `claude-4-opus`, `claude-3-7-sonnet`, `claude-3-5-sonnet`, `claude-3-5-haiku` | Anthropic Claude模型 |  
| **Gemini系列** | `gemini-2.5-pro`, `gemini-2.5-flash`, `gemini-2.0-flash`, `gemini-1`                             | Google Gemini模型    |
| **O系列**      | `o1-preview`, `o3`, `o3-mini`, `o4-mini`                                                         | OpenAI O系列模型       |
| **其他**       | `deepseek-reasoner`, `deepseek-chat`, `grok-3-beta`, `grok-4`, `sonar`, `sonar-reasoning-pro`    | 专业模型               |

## 🛠️ **高级功能**

### Custom Bot Mode（系统提示词支持）

通过启用 Custom Bot Mode，可以让所有的聊天请求都支持系统提示词（system prompt）功能：
1、启用 Custom Bot Mode
2、设置BOT_UID （必须）
```bash

⬇️ 启动项目后 ⬇️

# 现在所有 /v1/chat/completions 请求都支持 system prompt
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {
        "role": "system",
        "content": "你是一个海盗船长，用海盗的口吻说话"
      },
      {
        "role": "user",
        "content": "介绍一下你自己"
      }
    ]
  }'
```


---

<div align="center">

**如果这个项目对你有帮助，请给个 ⭐️ Star！**

</div>

## 📄 **许可证**

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情


### 原项目许可证
原项目采用 MIT 许可证，其许可证内容如下：

```
MIT License

Copyright (c) 2024 Monica Proxy

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### 本项目许可证
本项目同样采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

### 致谢
感谢原作者 [ycvk](https://github.com/ycvk) 创建了优秀的 Monica Proxy 项目，为本项目提供了坚实的基础。
