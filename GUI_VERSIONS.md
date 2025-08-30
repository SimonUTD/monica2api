# 多GUI版本说明

本项目提供了两种不同的GUI实现版本，以满足不同用户的需求和偏好。

## 版本概述

### 🖥️ Fyne版本 (main.go)
- **技术栈**: Go + Fyne v2
- **界面风格**: 原生桌面应用风格
- **特点**: 
  - 界面简洁，响应快速
  - 可执行文件体积小 (~15MB)
  - 资源占用少
  - 跨平台原生支持
  - 开发和部署简单

### 🌐 Wails版本 (main_wails.go)
- **技术栈**: Go + Wails v2 + Vue.js + Element Plus
- **界面风格**: 现代化Web界面
- **特点**:
  - 界面美观，交互体验好
  - 基于Web技术，定制性强
  - 组件库丰富，功能完整
  - 支持前端热重载开发
  - 可执行文件体积较大 (~50MB)

## 文件结构

```
monica-proxy/
├── main.go                  # Fyne版本入口文件
├── main_wails.go           # Wails版本入口文件
├── config.yaml             # 配置文件 (两个版本共享)
├── wails.json             # Wails项目配置
├── build-wails.sh         # Wails版本构建脚本
├── frontend/              # Wails前端代码
│   ├── src/
│   │   ├── views/         # Vue页面组件
│   │   ├── stores/        # Pinia状态管理
│   │   ├── router/        # Vue路由
│   │   └── App.vue       # 主应用组件
│   ├── package.json       # 前端依赖配置
│   └── vite.config.js     # Vite构建配置
├── internal/              # 内部业务逻辑 (共享)
└── README.md             # 项目说明文档
```

## 功能对比

| 功能 | Fyne版本 | Wails版本 | 说明 |
|------|-----------|-----------|------|
| Monica配置 | ✅ | ✅ | Cookie、Bot UID、Custom Bot模式 |
| 安全配置 | ✅ | ✅ | Bearer Token、限流、TLS等 |
| 服务器配置 | ✅ | ✅ | 主机、端口、超时等 |
| 代理配置 | ✅ | ✅ | HTTP/HTTPS代理设置 |
| 日志配置 | ✅ | ✅ | 日志级别、格式、输出 |
| 服务控制 | ✅ | ✅ | 启动/停止HTTP服务 |
| API测试 | ✅ | ✅ | 测试API配置是否正确 |
| 额度查询 | ✅ | ✅ | 查询Monica账号额度 |
| 配置保存 | ✅ | ✅ | 保存到config.yaml |
| 命令行模式 | ✅ | ❌ | Fyne版本支持-cli参数 |

## 构建和使用

### Fyne版本

```bash
# 编译
go build -o monica-proxy-fyne main.go

# 运行
./monica-proxy-fyne

# 命令行模式
./monica-proxy-fyne -cli
```

### Wails版本

```bash
# 前置条件
# 1. 安装Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. 确保安装了Node.js和npm

# 构建脚本方式 (推荐)
./build-wails.sh

# 手动构建
cd frontend && npm install && cd ..
wails build

# 运行
./build/bin/monica-proxy-wails

# 开发模式
wails dev
```

## 选择建议

### 选择Fyne版本，如果：
- 您需要轻量级的应用
- 对启动速度和资源占用有要求
- 偏好原生桌面应用的界面风格
- 需要命令行模式支持
- 希望构建和部署过程简单

### 选择Wails版本，如果：
- 您偏好现代化的Web界面风格
- 需要更丰富的交互体验
- 希望界面可定制性更强
- 需要前端开发的热重载功能
- 对可执行文件大小不敏感

## 配置兼容性

两个版本的配置文件格式完全兼容：
- 使用相同的`config.yaml`文件
- 配置参数和结构完全一致
- 可以在两个版本之间无缝切换
- 生成的配置文件可以互相使用

## 开发说明

### Fyne版本开发
- 直接修改`main.go`和相关的Go代码
- 界面使用Fyne的声明式UI构建
- 适合熟悉Go语言和传统GUI开发的开发者

### Wails版本开发
- 前端界面在`frontend/src/`目录下开发
- 使用Vue.js 3 + Element Plus + Pinia技术栈
- 后端逻辑在`main_wails.go`中
- 支持前端热重载，开发体验更好
- 适合熟悉Web技术栈的开发者

## 注意事项

1. **依赖管理**: 
   - Fyne版本只需要Go环境
   - Wails版本需要Go + Node.js环境

2. **构建时间**:
   - Fyne版本构建快速 (几秒钟)
   - Wails版本构建较慢 (需要安装前端依赖，1-2分钟)

3. **运行时性能**:
   - Fyne版本启动快，内存占用小
   - Wails版本有Webview开销，但界面更流畅

4. **更新维护**:
   - 两个版本会同步更新核心功能
   - 配置文件格式保持兼容
   - API接口完全一致

## 常见问题

**Q: 两个版本的功能有差异吗？**
A: 核心功能完全一致，只是界面实现不同。Wails版本界面更美观，Fyne版本更轻量。

**Q: 可以同时安装两个版本吗？**
A: 可以，两个版本的可执行文件名不同，可以共存。

**Q: 配置文件可以共享吗？**
A: 可以，两个版本使用相同的config.yaml文件，完全兼容。

**Q: 推荐使用哪个版本？**
A: 根据您的需求选择：
   - 追求轻量快速：选择Fyne版本
   - 追求界面体验：选择Wails版本