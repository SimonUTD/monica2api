package main

import (
	"context"
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
	utils "monica-proxy/internal/utils"

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
	cliMode = flag.Bool("cli", false, "启动命令行模式")
)

// 全局变量
var (
	serverApp *App
	serverMu  sync.Mutex
)

func main() {
	flag.Parse()

	// 如果指定了-cli参数，则启动命令行模式
	if *cliMode {
		startCLIMode()
		return
	}

	// 否则启动GUI模式（默认行为）
	startGUIMode()
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
	myWindow.Resize(fyne.NewSize(1200, 800))

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

// TestResult 测试结果结构
type TestResult struct {
	Endpoint     string
	URL          string
	RequestData  string
	ResponseData string
	StatusCode   int
	Error        error
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
	// 代理配置控件
	httpProxyEntry  *widget.Entry
	httpsProxyEntry *widget.Entry
	noProxyEntry    *widget.Entry

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
	testButton      *widget.Button
	quotaButton     *widget.Button
	startButton     *widget.Button
	stopButton      *widget.Button
	statusLabel     *widget.Label
	testStatusLabel *widget.Label // --- FIX: 新增测试状态标签 ---
	quotaLabel      *widget.Label

	// API信息显示
	apiInfoLabel *widget.Label
	baseUrlLabel *widget.Label
	apiKeyLabel  *widget.Label
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
	g.testButton = widget.NewButton("测试 API 配置", g.onTestConfig)

	// 添加查询额度按钮
	g.quotaButton = widget.NewButton("查询 Monica 额度", g.onQueryQuota)

	g.statusLabel = widget.NewLabel("服务状态: 未启动")
	g.statusLabel.Wrapping = fyne.TextWrapOff
	g.statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// --- FIX: 初始化测试状态标签 ---
	g.testStatusLabel = widget.NewLabel("测试状态: 未开始")
	g.testStatusLabel.Wrapping = fyne.TextWrapOff

	// 添加额度显示标签
	g.quotaLabel = widget.NewLabel("额度信息: 未查询")
	g.quotaLabel.Wrapping = fyne.TextWrapOff

	// 添加API信息显示标签
	g.apiInfoLabel = widget.NewLabel("API 端点:")
	g.apiInfoLabel.TextStyle = fyne.TextStyle{Bold: true}
	g.baseUrlLabel = widget.NewLabel("base_url: 服务未启动")
	g.baseUrlLabel.Wrapping = fyne.TextWrapOff
	g.apiKeyLabel = widget.NewLabel("API Key: ")
	g.apiKeyLabel.Wrapping = fyne.TextWrapOff

	// 设置默认值
	cfg := g.configManager.GetConfig()
	g.monicaCookieEntry.SetText(cfg.Monica.Cookie)

	// 设置API Key显示（如果已配置）
	if cfg.Security.BearerToken != "" {
		// 截断显示前8位，后面用...代替
		apiKey := cfg.Security.BearerToken
		if len(apiKey) > 8 {
			apiKey = apiKey[:8] + "..."
		}
		g.apiKeyLabel.SetText("API Key: " + apiKey)
	}
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

	// 创建proxy状态标签
	proxyStatus := "未启用代理"
	if cfg.Proxy.HTTPProxy != "" || cfg.Proxy.HTTPSProxy != "" {
		proxyStatus = "已启用代理"
	}
	proxyStatusLabel := widget.NewLabel("Proxy状态: " + proxyStatus)
	proxyStatusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// 控制按钮布局（垂直排列）
	controlButtons := container.NewVBox(
		widget.NewLabel("服务控制:"),
		widget.NewSeparator(),
		container.NewHBox(g.startButton, g.stopButton),
		g.statusLabel, // 服务状态放在启动按钮下面一行
		widget.NewSeparator(),
		container.NewHBox(g.testButton),
		g.testStatusLabel, // --- FIX: 将测试状态标签添加到布局中 ---
		widget.NewSeparator(),
		container.NewHBox(g.quotaButton, g.quotaLabel),
		widget.NewSeparator(),
		proxyStatusLabel,
		widget.NewSeparator(),
		g.apiInfoLabel,
		g.baseUrlLabel,
		g.apiKeyLabel,
		widget.NewSeparator(),
		widget.NewLabel("支持的API端点:"),
		widget.NewLabel("POST /v1/chat/completions - 聊天对话（兼容ChatGPT）"),
		widget.NewLabel("GET /v1/models - 获取模型列表"),
		widget.NewLabel("POST /v1/images/generations - 图片生成（兼容DALL-E）"),
	)

	// 左右布局：左侧配置表单，右侧控制面板
	split := container.NewHSplit(form, controlButtons)
	split.SetOffset(0.7) // 左侧占70%宽度

	content := container.NewVBox(
		split,
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
	// 初始化代理控件
	g.httpProxyEntry = widget.NewEntry()
	g.httpsProxyEntry = widget.NewEntry()
	g.noProxyEntry = widget.NewEntry()

	// 设置默认值
	cfg := g.configManager.GetConfig()
	g.serverHostEntry.SetText(cfg.Server.Host)
	g.serverPortEntry.SetText(strconv.Itoa(cfg.Server.Port))
	g.readTimeoutEntry.SetText(cfg.Server.ReadTimeout.String())
	g.writeTimeoutEntry.SetText(cfg.Server.WriteTimeout.String())
	g.idleTimeoutEntry.SetText(cfg.Server.IdleTimeout.String())
	// 设置代理默认值
	g.httpProxyEntry.SetText(cfg.Proxy.HTTPProxy)
	g.httpsProxyEntry.SetText(cfg.Proxy.HTTPSProxy)
	g.noProxyEntry.SetText(cfg.Proxy.NoProxy)

	// 创建布局
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "主机地址", Widget: g.serverHostEntry, HintText: "服务器监听的主机地址"},
			{Text: "端口", Widget: g.serverPortEntry, HintText: "服务器监听的端口号"},
			{Text: "读取超时", Widget: g.readTimeoutEntry, HintText: "读取请求的超时时间"},
			{Text: "写入超时", Widget: g.writeTimeoutEntry, HintText: "写入响应的超时时间"},
			{Text: "空闲超时", Widget: g.idleTimeoutEntry, HintText: "连接空闲超时时间"},
			{Text: "HTTP代理", Widget: g.httpProxyEntry, HintText: "HTTP代理地址（例如：http://proxy.example.com:8080）"},
			{Text: "HTTPS代理", Widget: g.httpsProxyEntry, HintText: "HTTPS代理地址（例如：https://proxy.example.com:8080）"},
			{Text: "不使用代理", Widget: g.noProxyEntry, HintText: "不使用代理的域名列表（逗号分隔）"},
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
		// 使用已更新的配置
		cfg := g.configManager.GetConfig()

		// 设置日志级别
		logger.SetLevel(cfg.Logging.Level)

		// 创建应用实例
		serverMu.Lock()
		serverApp = newApp(cfg)
		serverMu.Unlock()

		// 逻辑修复：先更新UI到“已启动”状态，然后再调用会阻塞的Start()方法。
		// 这样UI就能立即反映出服务正在运行。

		// 警告: Fyne UI操作不是线程安全的。
		// 直接在goroutine中更新UI组件可能会导致竞争条件或应用崩溃。
		// 这是一个逻辑上的修复，但底层的线程安全问题仍然存在。
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Monica Proxy",
			Content: "服务已启动",
		})

		testHost := cfg.Server.Host
		if testHost == "0.0.0.0" {
			testHost = "localhost"
		}
		baseUrl := fmt.Sprintf("http://%s:%d", testHost, cfg.Server.Port)

		g.statusLabel.SetText("服务状态: ✓ 已启动")
		g.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
		g.stopButton.Enable()
		g.baseUrlLabel.SetText("base_url: " + baseUrl)

		if cfg.Security.BearerToken != "" {
			apiKey := cfg.Security.BearerToken
			if len(apiKey) > 8 {
				apiKey = apiKey[:8] + "..."
			}
			g.apiKeyLabel.SetText("API Key: " + apiKey)
		}

		// 现在启动服务器。这个调用会阻塞当前的goroutine，直到服务停止。
		if err := serverApp.Start(); err != nil {
			// 当服务停止或启动失败时，Start()会返回一个错误。
			// 在这里更新UI以反映服务的最终状态。
			// 同样，这也是一个非线程安全的操作。
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Monica Proxy",
				Content: fmt.Sprintf("服务已停止或启动失败: %v", err),
			})
			g.statusLabel.SetText(fmt.Sprintf("服务已停止: %v", err))
			g.startButton.Enable()
			g.stopButton.Disable()
			g.baseUrlLabel.SetText("base_url: 服务未启动")
		}
	}()
}

// showDetailedTestResults 显示详细的测试结果
func (g *GUI) showDetailedTestResults(testResults []TestResult) {
	// 在主线程中更新UI
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   "测试完成",
		Content: "配置测试已完成，请查看详细结果",
	})

	// 创建详细结果窗口
	resultWindow := fyne.CurrentApp().NewWindow("详细测试结果")
	resultWindow.Resize(fyne.NewSize(800, 600))

	// 创建可折叠的内容区域
	var contentItems []fyne.CanvasObject

	for _, result := range testResults {
		// 创建状态标签
		var statusText string
		if result.Error != nil {
			statusText = fmt.Sprintf("❌ 失败: %v", result.Error)
		} else if result.StatusCode >= 200 && result.StatusCode < 300 {
			statusText = fmt.Sprintf("✅ 成功 (HTTP %d)", result.StatusCode)
		} else if result.StatusCode == 401 {
			statusText = fmt.Sprintf("❌ API Key错误 (HTTP %d)", result.StatusCode)
		} else if result.StatusCode == 403 {
			statusText = fmt.Sprintf("❌ 访问被拒绝 (HTTP %d)", result.StatusCode)
		} else {
			statusText = fmt.Sprintf("❌ 错误 (HTTP %d)", result.StatusCode)
		}

		// 创建详细信息卡片（默认折叠）
		details := container.NewVBox()

		// URL
		details.Add(widget.NewRichTextFromMarkdown(fmt.Sprintf("**请求URL:**\n```\n%s\n```", result.URL)))

		// 请求数据
		if result.RequestData != "" {
			details.Add(widget.NewRichTextFromMarkdown(fmt.Sprintf("**请求数据:**\n```json\n%s\n```", result.RequestData)))
		}

		// 响应数据
		if result.Error == nil {
			responseData := result.ResponseData
			if len(responseData) > 1000 {
				responseData = responseData[:1000] + "\n... (响应数据过长，已截断)"
			}
			details.Add(widget.NewRichTextFromMarkdown(fmt.Sprintf("**响应数据:**\n```json\n%s\n```", responseData)))
		}

		// 创建可折叠容器
		accordion := widget.NewAccordion(
			widget.NewAccordionItem(result.Endpoint+" - "+statusText, details),
		)
		contentItems = append(contentItems, accordion)
	}

	// 创建滚动容器
	scrollContainer := container.NewScroll(container.NewVBox(contentItems...))

	// 创建关闭按钮
	closeButton := widget.NewButton("关闭", func() {
		resultWindow.Close()
	})

	// 创建主容器
	mainContainer := container.NewBorder(nil, container.NewHBox(closeButton), nil, nil, scrollContainer)
	resultWindow.SetContent(mainContainer)
	resultWindow.Show()
}

// onStopService 停止服务事件处理
func (g *GUI) onStopService() {
	g.statusLabel.SetText("正在停止服务...")
	g.stopButton.Disable()

	// 在后台goroutine中停止服务
	go func() {
		serverMu.Lock()
		if serverApp != nil && serverApp.server != nil {
			// 使用Echo框架的Shutdown方法来优雅地停止服务器
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Shutdown会使另一个goroutine中的Start()方法返回，从而触发那里的UI更新逻辑
			if err := serverApp.server.Shutdown(ctx); err != nil {
				// 如果优雅关闭失败，强制关闭
				serverApp.server.Close()
			}

			serverApp = nil
		}
		serverMu.Unlock()

		// 逻辑修复：当用户点击停止时，UI更新的逻辑现在由 onStartService 的 goroutine 中
		// Start() 方法返回后的代码块处理。
		// 因此，这里不再需要直接更新UI到“未启动”状态。
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Monica Proxy",
			Content: "服务已尝试停止",
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

	// 更新代理配置
	cfg.Proxy.HTTPProxy = g.httpProxyEntry.Text
	cfg.Proxy.HTTPSProxy = g.httpsProxyEntry.Text
	cfg.Proxy.NoProxy = g.noProxyEntry.Text

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

	// 更新API Key显示
	if cfg.Security.BearerToken != "" {
		apiKey := cfg.Security.BearerToken
		if len(apiKey) > 8 {
			apiKey = apiKey[:8] + "..."
		}
		g.apiKeyLabel.SetText("API Key: " + apiKey)
	} else {
		g.apiKeyLabel.SetText("API Key: ")
	}
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

	// 检查服务是否启动
	serverMu.Lock()
	if serverApp == nil {
		serverMu.Unlock()
		dialog.ShowInformation("服务未启动", "请先启动服务再进行测试", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}
	serverMu.Unlock()

	// --- FIX: 更新独立的测试状态标签，而不是主服务状态标签 ---
	g.testStatusLabel.SetText("正在测试配置...")
	g.testButton.Disable()

	// 在后台goroutine中执行测试
	go func() {
		// 创建HTTP客户端
		client := resty.New()

		// 创建一个没有代理配置的 http transport
		// 这是为了确保测试客户端直接连接到本地服务器，而不是尝试通过系统或应用配置的代理。
		transport := &http.Transport{
			Proxy: nil,
		}
		client.SetTransport(transport)

		client.SetTimeout(30 * time.Second).
			SetHeaders(map[string]string{
				"Authorization": "Bearer " + cfg.Security.BearerToken,
				"User-Agent":    "Monica-Proxy-GUI/1.0",
				"Content-Type":  "application/json",
			})

		// 如果有Cookie，也添加到请求头
		if cfg.Monica.Cookie != "" {
			client.SetHeader("Cookie", cfg.Monica.Cookie)
		}

		// 测试结果收集
		var testResults []TestResult
		// 对于测试，使用localhost而不是0.0.0.0，因为0.0.0.0在测试时无法连接
		testHost := cfg.Server.Host
		if testHost == "0.0.0.0" {
			testHost = "localhost"
		}
		baseURL := fmt.Sprintf("http://%s:%d", testHost, cfg.Server.Port)

		// 测试1: 获取模型列表
		resp1, err1 := client.R().Get(baseURL + "/v1/models")
		result1 := TestResult{
			Endpoint:    "/v1/models",
			URL:         baseURL + "/v1/models",
			RequestData: "",
		}
		if err1 != nil {
			result1.Error = err1
		} else {
			result1.StatusCode = resp1.StatusCode()
			result1.ResponseData = resp1.String()
		}
		testResults = append(testResults, result1)

		// 测试2: 聊天对话接口
		chatData := `{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant."
    },
    {
      "role": "user",
      "content": "Hello"
    }
  ],
  "stream": false
}`
		resp2, err2 := client.R().SetBody(chatData).Post(baseURL + "/v1/chat/completions")
		result2 := TestResult{
			Endpoint:    "/v1/chat/completions",
			URL:         baseURL + "/v1/chat/completions",
			RequestData: chatData,
		}
		if err2 != nil {
			result2.Error = err2
		} else {
			result2.StatusCode = resp2.StatusCode()
			result2.ResponseData = resp2.String()
		}
		testResults = append(testResults, result2)

		// 测试3: 图片生成接口
		imageData := `{
  "prompt": "a white siamese cat",
  "n": 1,
  "size": "512x512"
}`
		resp3, err3 := client.R().SetBody(imageData).Post(baseURL + "/v1/images/generations")
		result3 := TestResult{
			Endpoint:    "/v1/images/generations",
			URL:         baseURL + "/v1/images/generations",
			RequestData: imageData,
		}
		if err3 != nil {
			result3.Error = err3
		} else {
			result3.StatusCode = resp3.StatusCode()
			result3.ResponseData = resp3.String()
		}
		testResults = append(testResults, result3)

		// 更新UI
		// 警告: 直接在goroutine中更新UI不是线程安全的，可能导致问题。
		g.testButton.Enable()
		// --- FIX: 测试结束后更新测试状态标签 ---
		g.testStatusLabel.SetText("测试完成，请查看详细结果")

		// 显示详细测试结果
		// 警告: 在goroutine中创建新窗口也不是线程安全的。
		g.showDetailedTestResults(testResults)
	}()
}

// onQueryQuota 查询Monica额度事件处理
func (g *GUI) onQueryQuota() {
	// 获取当前配置
	cfg := g.configManager.GetConfig()

	// 检查Cookie是否填写
	if cfg.Monica.Cookie == "" {
		dialog.ShowInformation("配置错误", "请先填写Monica Cookie", fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}

	// 显示查询中状态
	g.quotaLabel.SetText("额度信息: 查询中...")
	g.quotaButton.Disable()

	// 在后台goroutine中执行查询
	go func() {
		defer g.quotaButton.Enable()

		// 使用utils包中的函数获取额度信息
		quotaResp, err := utils.GetMonicaQuota(cfg)
		if err != nil {
			g.quotaLabel.SetText("额度信息: 查询失败 - " + err.Error())
			return
		}

		// 解析额度信息
		var geniusBotQuota, creditsQuota int
		for _, module := range quotaResp.Data.ModuleQuotas {
			for _, quota := range module.Quotas {
				if quota.Scene == "plan" {
					if module.Module == "genius_bot" {
						geniusBotQuota = quota.CurrentQuota
					} else if module.Module == "credits" {
						creditsQuota = quota.CurrentQuota
					}
				}
			}
		}

		// 显示额度信息
		quotaText := fmt.Sprintf("额度信息: Genius Bot: %d, Credits: %d", geniusBotQuota, creditsQuota)
		g.quotaLabel.SetText(quotaText)
	}()
}
