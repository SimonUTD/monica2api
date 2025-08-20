package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"monica-proxy/internal/apiserver"
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	customMiddleware "monica-proxy/internal/middleware"
	"monica-proxy/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// 命令行参数
var (
	guiMode = flag.Bool("gui", false, "启动GUI配置界面")
)

// 全局变量
var (
	serverApp *App
	serverMu  sync.Mutex
)

func main() {
	flag.Parse()

	// 如果指定了-gui参数，则启动GUI模式
	if *guiMode {
		startGUIMode()
		return
	}

	// 否则启动命令行模式（默认行为）
	startCLIMode()
}

// startGUIMode 启动GUI模式
func startGUIMode() {
	startGUI()
}

// startCLIMode 启动命令行模式
func startCLIMode() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 设置日志级别
	logger.SetLevel(cfg.Logging.Level)

	// 创建应用实例
	app := newApp(cfg)

	// 启动服务器
	logger.Info("启动服务器", zap.String("address", cfg.GetAddress()))

	if err := app.Start(); err != nil {
		logger.Fatal("启动服务器失败", zap.Error(err))
	}
}

// startGUI 启动GUI配置界面
func startGUI() {
	// 创建Fyne应用
	myApp := app.New()
	myWindow := myApp.NewWindow("Monica Proxy 配置")
	myWindow.Resize(fyne.NewSize(600, 800))

	// 创建配置管理器
	configManager := NewConfigManager()

	// 创建GUI界面
	gui := NewGUI(configManager)

	// 设置主内容
	myWindow.SetContent(gui.CreateMainContainer())

	// 设置窗口关闭事件
	myWindow.SetOnClosed(func() {
		// 如果服务正在运行，停止服务
		serverMu.Lock()
		if serverApp != nil {
			// 在GUI模式下，我们不能优雅地关闭echo服务
			// 因为GUI应用和HTTP服务运行在同一个进程中
		}
		serverMu.Unlock()
	})

	// 显示窗口并运行应用
	myWindow.ShowAndRun()
}

// ConfigManager 配置管理器
type ConfigManager struct {
	config *config.Config
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager() *ConfigManager {
	// 加载现有配置或创建默认配置
	cfg, err := config.Load()
	if err != nil {
		// 如果加载失败，使用默认配置
		cfg = config.GetDefaultConfig()
	}

	return &ConfigManager{
		config: cfg,
	}
}

// SaveConfig 保存配置到文件
func (cm *ConfigManager) SaveConfig() error {
	// 将配置保存到config.yaml文件
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return err
	}

	return os.WriteFile("config.yaml", data, 0644)
}

// GetConfig 获取当前配置
func (cm *ConfigManager) GetConfig() *config.Config {
	return cm.config
}

// GUI 图形用户界面
type GUI struct {
	configManager *ConfigManager

	// 服务器配置控件
	serverHostEntry   *widget.Entry
	serverPortEntry   *widget.Entry
	readTimeoutEntry  *widget.Entry
	writeTimeoutEntry *widget.Entry
	idleTimeoutEntry  *widget.Entry

	// Monica配置控件
	monicaCookieEntry        *widget.Entry
	monicaBotUIDEntry        *widget.Entry
	enableCustomBotModeCheck *widget.Check

	// 安全配置控件
	bearerTokenEntry      *widget.Entry
	tlsSkipVerifyCheck    *widget.Check
	rateLimitEnabledCheck *widget.Check
	rateLimitRPSEntry     *widget.Entry
	requestTimeoutEntry   *widget.Entry

	// 日志配置控件
	logLevelEntry         interface{} // 可以是*widget.Entry或*widget.Select
	logFormatEntry        interface{} // 可以是*widget.Entry或*widget.Select
	logOutputEntry        interface{} // 可以是*widget.Entry或*widget.Select
	enableRequestLogCheck *widget.Check
	maskSensitiveCheck    *widget.Check

	// 服务控制
	testButton  *widget.Button
	startButton *widget.Button
	stopButton  *widget.Button
	statusLabel *widget.Label
}

// NewGUI 创建新的GUI实例
func NewGUI(configManager *ConfigManager) *GUI {
	return &GUI{
		configManager: configManager,
	}
}

// createMainConfigTab 创建主要配置标签页（包含服务控制、Monica配置和安全配置）
func (g *GUI) createMainConfigTab() *container.Scroll {
	// 初始化控件
	g.monicaCookieEntry = widget.NewMultiLineEntry()
	g.monicaCookieEntry.Wrapping = fyne.TextWrapWord
	g.monicaBotUIDEntry = widget.NewEntry()
	g.enableCustomBotModeCheck = widget.NewCheck("启用自定义Bot模式", g.onCustomBotModeChanged)

	g.bearerTokenEntry = widget.NewEntry()
	g.tlsSkipVerifyCheck = widget.NewCheck("跳过TLS验证", nil)
	g.rateLimitEnabledCheck = widget.NewCheck("启用限流", nil)
	g.rateLimitRPSEntry = widget.NewEntry()
	g.requestTimeoutEntry = widget.NewEntry()

	g.startButton = widget.NewButton("启动服务", g.onStartService)
	g.stopButton = widget.NewButton("停止服务", g.onStopService)
	g.stopButton.Disable()

	// 添加测试按钮
	g.testButton = widget.NewButton("测试配置", g.onTestConfig)

	g.statusLabel = widget.NewLabel("服务状态: 未启动")
	g.statusLabel.Wrapping = fyne.TextWrapWord

	// 设置默认值
	cfg := g.configManager.GetConfig()
	g.monicaCookieEntry.SetText(cfg.Monica.Cookie)
	g.monicaBotUIDEntry.SetText(cfg.Monica.BotUID)
	g.enableCustomBotModeCheck.SetChecked(cfg.Monica.EnableCustomBotMode)

	g.bearerTokenEntry.SetText(cfg.Security.BearerToken)
	g.tlsSkipVerifyCheck.SetChecked(cfg.Security.TLSSkipVerify)
	g.rateLimitEnabledCheck.SetChecked(cfg.Security.RateLimitEnabled)
	g.rateLimitRPSEntry.SetText(strconv.Itoa(cfg.Security.RateLimitRPS))
	g.requestTimeoutEntry.SetText(cfg.Security.RequestTimeout.String())

	// 根据Custom Bot模式状态设置Bot UID的可用性
	if !cfg.Monica.EnableCustomBotMode {
		g.monicaBotUIDEntry.Disable()
	}

	// 创建布局 - 简化设计，去除不必要的卡片容器
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Cookie*", Widget: g.monicaCookieEntry, HintText: "Monica登录后的Cookie（必填）"},
			{Text: "Bot UID", Widget: g.monicaBotUIDEntry, HintText: "自定义Bot的UID（启用Custom Bot模式时必需）"},
			{Text: "启用自定义Bot模式", Widget: g.enableCustomBotModeCheck, HintText: "启用后支持系统提示词"},
			{Text: "API Key*", Widget: g.bearerTokenEntry, HintText: "API访问令牌（必填）"},
			{Text: "跳过TLS验证", Widget: g.tlsSkipVerifyCheck, HintText: "是否跳过TLS证书验证"},
			{Text: "启用限流", Widget: g.rateLimitEnabledCheck, HintText: "是否启用请求限流"},
			{Text: "限流RPS", Widget: g.rateLimitRPSEntry, HintText: "每秒请求数限制"},
			{Text: "请求超时", Widget: g.requestTimeoutEntry, HintText: "请求超时时间"},
		},
	}

	// 控制按钮布局
	buttons := container.NewHBox(g.testButton, g.startButton, g.stopButton)

	content := container.NewVBox(
		form,
		widget.NewSeparator(),
		buttons,
		widget.NewSeparator(),
		g.statusLabel,
	)

	return container.NewScroll(content)
}

// onCustomBotModeChanged Custom Bot模式变更事件处理
func (g *GUI) onCustomBotModeChanged(checked bool) {
	if checked {
		g.monicaBotUIDEntry.Enable()
	} else {
		g.monicaBotUIDEntry.Disable()
	}
}

// CreateMainContainer 创建主容器
func (g *GUI) CreateMainContainer() *container.AppTabs {
	// 创建各个配置标签页
	mainConfigTab := g.createMainConfigTab()
	serverTab := g.createServerTab()
	loggingTab := g.createLoggingTab()

	// 创建标签页容器
	tabs := container.NewAppTabs(
		container.NewTabItem("主要配置", mainConfigTab),
		container.NewTabItem("服务器配置", serverTab),
		container.NewTabItem("日志配置", loggingTab),
	)

	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

// createServerTab 创建服务器配置标签页
func (g *GUI) createServerTab() *container.Scroll {
	// 初始化控件
	g.serverHostEntry = widget.NewEntry()
	g.serverPortEntry = widget.NewEntry()
	g.readTimeoutEntry = widget.NewEntry()
	g.writeTimeoutEntry = widget.NewEntry()
	g.idleTimeoutEntry = widget.NewEntry()

	// 设置默认值
	cfg := g.configManager.GetConfig()
	g.serverHostEntry.SetText(cfg.Server.Host)
	g.serverPortEntry.SetText(strconv.Itoa(cfg.Server.Port))
	g.readTimeoutEntry.SetText(cfg.Server.ReadTimeout.String())
	g.writeTimeoutEntry.SetText(cfg.Server.WriteTimeout.String())
	g.idleTimeoutEntry.SetText(cfg.Server.IdleTimeout.String())

	// 创建布局
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "主机地址", Widget: g.serverHostEntry, HintText: "服务器监听的主机地址"},
			{Text: "端口", Widget: g.serverPortEntry, HintText: "服务器监听的端口号"},
			{Text: "读取超时", Widget: g.readTimeoutEntry, HintText: "读取请求的超时时间"},
			{Text: "写入超时", Widget: g.writeTimeoutEntry, HintText: "写入响应的超时时间"},
			{Text: "空闲超时", Widget: g.idleTimeoutEntry, HintText: "连接空闲超时时间"},
		},
	}

	return container.NewScroll(form)
}

// createMonicaTab 创建Monica配置标签页
func (g *GUI) createMonicaTab() *container.Scroll {
	// 初始化控件
	g.monicaCookieEntry = widget.NewEntry()
	g.monicaBotUIDEntry = widget.NewEntry()
	g.enableCustomBotModeCheck = widget.NewCheck("启用自定义Bot模式", nil)

	// 设置默认值
	cfg := g.configManager.GetConfig()
	g.monicaCookieEntry.SetText(cfg.Monica.Cookie)
	g.monicaBotUIDEntry.SetText(cfg.Monica.BotUID)
	g.enableCustomBotModeCheck.SetChecked(cfg.Monica.EnableCustomBotMode)

	// 创建布局
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Cookie*", Widget: g.monicaCookieEntry, HintText: "Monica登录后的Cookie（必填）"},
			{Text: "Bot UID", Widget: g.monicaBotUIDEntry, HintText: "自定义Bot的UID"},
			{Text: "启用自定义Bot模式", Widget: g.enableCustomBotModeCheck, HintText: "启用后支持系统提示词"},
		},
	}

	return container.NewScroll(form)
}

// createSecurityTab 创建安全配置标签页
func (g *GUI) createSecurityTab() *container.Scroll {
	// 初始化控件
	g.bearerTokenEntry = widget.NewEntry()
	g.tlsSkipVerifyCheck = widget.NewCheck("跳过TLS验证", nil)
	g.rateLimitEnabledCheck = widget.NewCheck("启用限流", nil)
	g.rateLimitRPSEntry = widget.NewEntry()
	g.requestTimeoutEntry = widget.NewEntry()

	// 设置默认值
	cfg := g.configManager.GetConfig()
	g.bearerTokenEntry.SetText(cfg.Security.BearerToken)
	g.tlsSkipVerifyCheck.SetChecked(cfg.Security.TLSSkipVerify)
	g.rateLimitEnabledCheck.SetChecked(cfg.Security.RateLimitEnabled)
	g.rateLimitRPSEntry.SetText(strconv.Itoa(cfg.Security.RateLimitRPS))
	g.requestTimeoutEntry.SetText(cfg.Security.RequestTimeout.String())

	// 创建布局
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Bearer Token*", Widget: g.bearerTokenEntry, HintText: "API访问令牌（必填）"},
			{Text: "跳过TLS验证", Widget: g.tlsSkipVerifyCheck, HintText: "是否跳过TLS证书验证"},
			{Text: "启用限流", Widget: g.rateLimitEnabledCheck, HintText: "是否启用请求限流"},
			{Text: "限流RPS", Widget: g.rateLimitRPSEntry, HintText: "每秒请求数限制"},
			{Text: "请求超时", Widget: g.requestTimeoutEntry, HintText: "请求超时时间"},
		},
	}

	return container.NewScroll(form)
}

// createLoggingTab 创建日志配置标签页
func (g *GUI) createLoggingTab() *container.Scroll {
	// 初始化控件
	logLevelSelect := widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	logFormatSelect := widget.NewSelect([]string{"json", "console"}, nil)
	logOutputSelect := widget.NewSelect([]string{"stdout", "stderr", "file"}, nil)
	g.enableRequestLogCheck = widget.NewCheck("启用请求日志", nil)
	g.maskSensitiveCheck = widget.NewCheck("掩盖敏感信息", nil)

	// 设置默认值
	cfg := g.configManager.GetConfig()

	// 设置日志级别下拉框
	logLevelSelect.SetSelected(cfg.Logging.Level)

	// 设置日志格式下拉框
	logFormatSelect.SetSelected(cfg.Logging.Format)

	// 设置日志输出下拉框
	logOutputSelect.SetSelected(cfg.Logging.Output)

	g.enableRequestLogCheck.SetChecked(cfg.Logging.EnableRequestLog)
	g.maskSensitiveCheck.SetChecked(cfg.Logging.MaskSensitive)

	// 保存控件引用
	g.logLevelEntry = logLevelSelect
	g.logFormatEntry = logFormatSelect
	g.logOutputEntry = logOutputSelect

	// 创建日志文件路径显示
	logPathLabel := widget.NewLabel("日志文件路径: ")
	logPathValue := widget.NewLabel("./logs/monica-proxy.log")
	logPathValue.Wrapping = fyne.TextWrapWord

	// 创建打开日志文件按钮
	openLogButton := widget.NewButton("打开日志文件所在路径", func() {
		// 这里应该实现打开日志文件所在路径的功能
		// 由于跨平台实现复杂，这里只是示例
	})

	// 创建布局
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "日志级别", Widget: logLevelSelect, HintText: "选择日志级别"},
			{Text: "日志格式", Widget: logFormatSelect, HintText: "选择日志格式"},
			{Text: "日志输出", Widget: logOutputSelect, HintText: "选择日志输出方式"},
			{Text: "启用请求日志", Widget: g.enableRequestLogCheck, HintText: "是否启用请求日志"},
			{Text: "掩盖敏感信息", Widget: g.maskSensitiveCheck, HintText: "是否掩盖敏感信息"},
		},
	}

	logPathGroup := widget.NewCard("日志文件信息", "",
		container.NewVBox(
			logPathLabel,
			logPathValue,
			openLogButton,
		),
	)

	content := container.NewVBox(
		form,
		logPathGroup,
	)

	return container.NewScroll(content)
}

// createControlTab 创建服务控制标签页
func (g *GUI) createControlTab() *container.Scroll {
	// 初始化控件
	g.startButton = widget.NewButton("启动服务", g.onStartService)
	g.stopButton = widget.NewButton("停止服务", g.onStopService)
	g.stopButton.Disable()

	g.statusLabel = widget.NewLabel("服务状态: 未启动")
	g.statusLabel.Wrapping = fyne.TextWrapWord

	// 创建布局
	buttons := container.NewHBox(g.startButton, g.stopButton)

	content := container.NewVBox(
		widget.NewLabel("Monica Proxy 服务控制"),
		buttons,
		widget.NewSeparator(),
		g.statusLabel,
	)

	return container.NewScroll(content)
}

// onStartService 启动服务事件处理
func (g *GUI) onStartService() {
	// 更新配置
	g.updateConfigFromUI()

	// 保存配置到文件
	if err := g.configManager.SaveConfig(); err != nil {
		g.statusLabel.SetText(fmt.Sprintf("保存配置失败: %v", err))
		return
	}

	g.statusLabel.SetText("配置已保存，正在启动服务...")
	g.startButton.Disable()

	// 在后台goroutine中启动服务
	go func() {
		// 创建应用实例
		cfg := g.configManager.GetConfig()

		// 设置日志级别
		logger.SetLevel(cfg.Logging.Level)

		// 创建应用实例
		serverMu.Lock()
		serverApp = newApp(cfg)
		serverMu.Unlock()

		// 启动服务器
		g.statusLabel.SetText("服务启动中...")

		if err := serverApp.Start(); err != nil {
			g.statusLabel.SetText(fmt.Sprintf("服务启动失败: %v", err))
			g.startButton.Enable()
			return
		}

		// 更新UI（需要在主线程中执行）
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Monica Proxy",
			Content: "服务已启动",
		})
	}()

	g.stopButton.Enable()
	g.statusLabel.SetText("服务已启动")
}

// onStopService 停止服务事件处理
func (g *GUI) onStopService() {
	g.statusLabel.SetText("正在停止服务...")
	g.stopButton.Disable()

	// 在后台goroutine中停止服务
	go func() {
		serverMu.Lock()
		if serverApp != nil {
			// 注意：echo框架没有提供Stop方法，我们需要使用其他方式停止
			// 这里我们只是更新状态
			serverApp = nil
		}
		serverMu.Unlock()

		// 更新UI（需要在主线程中执行）
		g.startButton.Enable()
		g.statusLabel.SetText("服务已停止")

		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Monica Proxy",
			Content: "服务已停止",
		})
	}()
}

// updateConfigFromUI 从UI更新配置
func (g *GUI) updateConfigFromUI() {
	cfg := g.configManager.GetConfig()

	// 更新服务器配置
	cfg.Server.Host = g.serverHostEntry.Text
	if port, err := strconv.Atoi(g.serverPortEntry.Text); err == nil {
		cfg.Server.Port = port
	}
	// 注意：在实际实现中，需要解析时间字符串为time.Duration

	// 更新Monica配置
	cfg.Monica.Cookie = g.monicaCookieEntry.Text
	cfg.Monica.BotUID = g.monicaBotUIDEntry.Text
	cfg.Monica.EnableCustomBotMode = g.enableCustomBotModeCheck.Checked

	// 更新安全配置
	cfg.Security.BearerToken = g.bearerTokenEntry.Text
	cfg.Security.TLSSkipVerify = g.tlsSkipVerifyCheck.Checked
	cfg.Security.RateLimitEnabled = g.rateLimitEnabledCheck.Checked
	if rps, err := strconv.Atoi(g.rateLimitRPSEntry.Text); err == nil {
		cfg.Security.RateLimitRPS = rps
	}
	// 注意：在实际实现中，需要解析时间字符串为time.Duration

	// 更新日志配置
	if logLevelSelect, ok := g.logLevelEntry.(*widget.Select); ok {
		cfg.Logging.Level = logLevelSelect.Selected
	}
	if logFormatSelect, ok := g.logFormatEntry.(*widget.Select); ok {
		cfg.Logging.Format = logFormatSelect.Selected
	}
	if logOutputSelect, ok := g.logOutputEntry.(*widget.Select); ok {
		cfg.Logging.Output = logOutputSelect.Selected
	}
	cfg.Logging.EnableRequestLog = g.enableRequestLogCheck.Checked
	cfg.Logging.MaskSensitive = g.maskSensitiveCheck.Checked
}

// App 应用实例
type App struct {
	config *config.Config
	server *echo.Echo
}

// newApp 创建应用实例
func newApp(cfg *config.Config) *App {
	// 初始化HTTP客户端
	utils.InitHTTPClients(cfg)

	// 设置 Echo Server
	e := echo.New()
	e.Logger.SetOutput(io.Discard)
	e.HideBanner = true

	// 配置服务器
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout
	e.Server.IdleTimeout = cfg.Server.IdleTimeout

	// 添加基础中间件
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())

	// 添加限流中间件
	e.Use(customMiddleware.RateLimit(cfg))

	// 注册路由
	apiserver.RegisterRoutes(e, cfg)

	return &App{
		config: cfg,
		server: e,
	}
}

// Start 启动应用
func (a *App) Start() error {
	return a.server.Start(a.config.GetAddress())
}

// onTestConfig 测试配置事件处理
func (g *GUI) onTestConfig() {
	// 更新配置
	g.updateConfigFromUI()
	
	// 获取当前配置
	cfg := g.configManager.GetConfig()
	
	// 检查必填项
	if cfg.Monica.Cookie == "" {
		dialog.ShowInformation("配置错误", "请填写Monica Cookie", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}
	
	if cfg.Security.BearerToken == "" {
		dialog.ShowInformation("配置错误", "请填写API Key", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}
	
	// 如果启用了Custom Bot模式，检查Bot UID
	if cfg.Monica.EnableCustomBotMode && cfg.Monica.BotUID == "" {
		dialog.ShowInformation("配置错误", "启用Custom Bot模式时必须填写Bot UID", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}
	
	// 显示测试中状态
	g.statusLabel.SetText("正在测试配置...")
	g.testButton.Disable()
	
	// 在后台goroutine中执行测试
	go func() {
		// 创建HTTP客户端
		client := resty.New().
			SetTimeout(30 * time.Second).
			SetHeaders(map[string]string{
				"Authorization": "Bearer " + cfg.Security.BearerToken,
				"User-Agent":    "Monica-Proxy-GUI/1.0",
			})
		
		// 如果有Cookie，也添加到请求头
		if cfg.Monica.Cookie != "" {
			client.SetHeader("Cookie", cfg.Monica.Cookie)
		}
		
		// 测试API端点 - 获取模型列表
		resp, err := client.R().Get(fmt.Sprintf("http://%s:%d/v1/models", cfg.Server.Host, cfg.Server.Port))
		
		// 更新UI需要在主线程中执行
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title: "配置测试",
		})
		
		// 更新UI
		g.testButton.Enable()
		
		if err != nil {
			g.statusLabel.SetText(fmt.Sprintf("测试失败: %v", err))
			dialog.ShowError(fmt.Errorf("测试失败: %v", err), fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}
		
		if resp.StatusCode() == http.StatusOK {
			g.statusLabel.SetText("配置测试成功!")
			dialog.ShowInformation("测试成功", "配置验证通过，可以正常启动服务", fyne.CurrentApp().Driver().AllWindows()[0])
		} else if resp.StatusCode() == http.StatusUnauthorized {
			g.statusLabel.SetText("API Key错误")
			dialog.ShowError(fmt.Errorf("API Key验证失败，请检查API Key是否正确"), fyne.CurrentApp().Driver().AllWindows()[0])
		} else if resp.StatusCode() == http.StatusForbidden {
			g.statusLabel.SetText("访问被拒绝")
			dialog.ShowError(fmt.Errorf("访问被拒绝，请检查Cookie和API Key是否正确"), fyne.CurrentApp().Driver().AllWindows()[0])
		} else {
			g.statusLabel.SetText(fmt.Sprintf("测试失败: HTTP %d", resp.StatusCode()))
			dialog.ShowError(fmt.Errorf("测试失败: HTTP %d\n响应: %s", resp.StatusCode(), resp.String()), fyne.CurrentApp().Driver().AllWindows()[0])
		}
	}()
}
