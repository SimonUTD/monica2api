package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"monica-proxy/internal/apiserver"
	"monica-proxy/internal/config"
	"monica-proxy/internal/logger"
	customMiddleware "monica-proxy/internal/middleware"
	utils "monica-proxy/internal/utils"

	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"gopkg.in/yaml.v3"
)

//go:embed all:frontend/dist
var assets embed.FS

// 全局变量
var (
	wailsServerApp *WailsBackendApp
	wailsServerMu  sync.Mutex
)

// WailsBackendApp 结构体包含后端应用程序的状态和配置
type WailsBackendApp struct {
	config *config.Config
	server *echo.Echo
}

// WailsApp Wails应用程序结构
type WailsApp struct {
	configManager *WailsConfigManager
	ctx           context.Context
}

// WailsConfigManager 配置管理器
type WailsConfigManager struct {
	config *config.Config
}

// NewWailsConfigManager 创建新的配置管理器
func NewWailsConfigManager() *WailsConfigManager {
	// 尝试从用户配置目录加载配置
	var cfg *config.Config
	var err error

	// 首先尝试从用户配置目录加载
	if userHome, homeErr := os.UserHomeDir(); homeErr == nil {
		userConfigPath := filepath.Join(userHome, ".monica-proxy", "config.yaml")
		if _, statErr := os.Stat(userConfigPath); statErr == nil {
			// 临时设置环境变量让config.Load()能找到用户配置文件
			oldConfigFile := os.Getenv("CONFIG_FILE")
			os.Setenv("CONFIG_FILE", userConfigPath)
			cfg, err = config.Load()
			// 恢复原来的环境变量
			if oldConfigFile != "" {
				os.Setenv("CONFIG_FILE", oldConfigFile)
			} else {
				os.Unsetenv("CONFIG_FILE")
			}
		}
	}

	// 如果用户配置文件不存在或加载失败，使用默认加载方式
	if cfg == nil {
		cfg, err = config.Load()
		if err != nil {
			cfg = config.GetDefaultConfig()
		}
	}

	return &WailsConfigManager{config: cfg}
}

// SaveConfig 保存配置
func (cm *WailsConfigManager) SaveConfig() error {
	// 将配置保存到config.yaml文件
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		return err
	}

	// 获取用户配置目录
	configPath := "config.yaml"
	if userHome, err := os.UserHomeDir(); err == nil {
		configPath = filepath.Join(userHome, ".monica-proxy", "config.yaml")
		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			// 如果创建用户目录失败，回退到当前目录
			configPath = "config.yaml"
		}
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetConfig 获取配置
func (cm *WailsConfigManager) GetConfig() *config.Config {
	return cm.config
}

// WailsTestResult 测试结果
type WailsTestResult struct {
	Endpoint     string `json:"endpoint"`
	URL          string `json:"url"`
	RequestData  string `json:"requestData"`
	ResponseData string `json:"responseData"`
	StatusCode   int    `json:"statusCode"`
	Error        string `json:"error,omitempty"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	IsRunning bool   `json:"isRunning"`
	Message   string `json:"message"`
	Address   string `json:"address,omitempty"`
	APIKey    string `json:"apiKey,omitempty"`
}

// QuotaInfo 额度信息
type QuotaInfo struct {
	GeniusBot int    `json:"geniusBot"`
	Credits   int    `json:"credits"`
	Error     string `json:"error,omitempty"`
}

// NewWailsApp 创建Wails应用
func NewWailsApp() *WailsApp {
	return &WailsApp{
		configManager: NewWailsConfigManager(),
	}
}

// Startup 在应用启动时调用
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx
}

// DomReady 在DOM准备就绪时调用
func (a *WailsApp) DomReady(ctx context.Context) {
	// 可以在这里执行前端相关的初始化
}

// Shutdown 在应用关闭时调用
func (a *WailsApp) Shutdown(ctx context.Context) {
	// 停止服务
	wailsServerMu.Lock()
	if wailsServerApp != nil && wailsServerApp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		wailsServerApp.server.Shutdown(ctx)
		wailsServerApp = nil
	}
	wailsServerMu.Unlock()
}

// GetConfig 获取当前配置
func (a *WailsApp) GetConfig() map[string]interface{} {
	cfg := a.configManager.GetConfig()
	return map[string]interface{}{
		"server": map[string]interface{}{
			"host":         cfg.Server.Host,
			"port":         cfg.Server.Port,
			"readTimeout":  cfg.Server.ReadTimeout.String(),
			"writeTimeout": cfg.Server.WriteTimeout.String(),
			"idleTimeout":  cfg.Server.IdleTimeout.String(),
		},
		"proxy": map[string]interface{}{
			"httpProxy":  cfg.Proxy.HTTPProxy,
			"httpsProxy": cfg.Proxy.HTTPSProxy,
			"noProxy":    cfg.Proxy.NoProxy,
		},
		"monica": map[string]interface{}{
			"cookie":              cfg.Monica.Cookie,
			"botUID":              cfg.Monica.BotUID,
			"enableCustomBotMode": cfg.Monica.EnableCustomBotMode,
		},
		"security": map[string]interface{}{
			"bearerToken":      cfg.Security.BearerToken,
			"tlsSkipVerify":    cfg.Security.TLSSkipVerify,
			"rateLimitEnabled": cfg.Security.RateLimitEnabled,
			"rateLimitRPS":     cfg.Security.RateLimitRPS,
			"requestTimeout":   cfg.Security.RequestTimeout.String(),
		},
		"logging": map[string]interface{}{
			"level":            cfg.Logging.Level,
			"format":           cfg.Logging.Format,
			"output":           cfg.Logging.Output,
			"enableRequestLog": cfg.Logging.EnableRequestLog,
			"maskSensitive":    cfg.Logging.MaskSensitive,
		},
	}
}

// UpdateConfig 更新配置
func (a *WailsApp) UpdateConfig(configData map[string]interface{}) error {
	cfg := a.configManager.GetConfig()

	// 更新服务器配置
	if server, ok := configData["server"].(map[string]interface{}); ok {
		if host, ok := server["host"].(string); ok {
			cfg.Server.Host = host
		}
		if port, ok := server["port"].(float64); ok {
			cfg.Server.Port = int(port)
		}
	}

	// 更新代理配置
	if proxy, ok := configData["proxy"].(map[string]interface{}); ok {
		if httpProxy, ok := proxy["httpProxy"].(string); ok {
			cfg.Proxy.HTTPProxy = httpProxy
		}
		if httpsProxy, ok := proxy["httpsProxy"].(string); ok {
			cfg.Proxy.HTTPSProxy = httpsProxy
		}
		if noProxy, ok := proxy["noProxy"].(string); ok {
			cfg.Proxy.NoProxy = noProxy
		}
	}

	// 更新Monica配置
	if monica, ok := configData["monica"].(map[string]interface{}); ok {
		if cookie, ok := monica["cookie"].(string); ok {
			cfg.Monica.Cookie = cookie
		}
		if botUID, ok := monica["botUID"].(string); ok {
			cfg.Monica.BotUID = botUID
		}
		if enableCustomBotMode, ok := monica["enableCustomBotMode"].(bool); ok {
			cfg.Monica.EnableCustomBotMode = enableCustomBotMode
		}
	}

	// 更新安全配置
	if security, ok := configData["security"].(map[string]interface{}); ok {
		if bearerToken, ok := security["bearerToken"].(string); ok {
			cfg.Security.BearerToken = bearerToken
		}
		if tlsSkipVerify, ok := security["tlsSkipVerify"].(bool); ok {
			cfg.Security.TLSSkipVerify = tlsSkipVerify
		}
		if rateLimitEnabled, ok := security["rateLimitEnabled"].(bool); ok {
			cfg.Security.RateLimitEnabled = rateLimitEnabled
		}
		if rateLimitRPS, ok := security["rateLimitRPS"].(float64); ok {
			cfg.Security.RateLimitRPS = int(rateLimitRPS)
		}
	}

	// 更新日志配置
	if logging, ok := configData["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			cfg.Logging.Level = level
		}
		if format, ok := logging["format"].(string); ok {
			cfg.Logging.Format = format
		}
		if output, ok := logging["output"].(string); ok {
			cfg.Logging.Output = output
		}
		if enableRequestLog, ok := logging["enableRequestLog"].(bool); ok {
			cfg.Logging.EnableRequestLog = enableRequestLog
		}
		if maskSensitive, ok := logging["maskSensitive"].(bool); ok {
			cfg.Logging.MaskSensitive = maskSensitive
		}
	}

	return a.configManager.SaveConfig()
}

// StartService 启动服务
func (a *WailsApp) StartService() error {
	// 保存配置
	if err := a.configManager.SaveConfig(); err != nil {
		return fmt.Errorf("保存配置失败: %v", err)
	}

	cfg := a.configManager.GetConfig()

	// 设置日志配置
	logOutput := cfg.Logging.Output
	if logOutput == "file" {
		// 如果是文件模式，使用用户目录下的日志路径
		if userHome, err := os.UserHomeDir(); err == nil {
			logOutput = filepath.Join(userHome, ".monica-proxy", "logs", "monica-proxy.log")
		} else {
			logOutput = "./logs/monica-proxy.log"
		}
	}
	logger.UpdateConfig(cfg.Logging.Level, cfg.Logging.Format, logOutput, cfg.Logging.MaskSensitive)

	// 创建应用实例
	utils.InitHTTPClients(cfg)

	wailsServerMu.Lock()
	defer wailsServerMu.Unlock()

	// 如果服务已在运行，先停止
	if wailsServerApp != nil {
		return fmt.Errorf("服务已在运行")
	}

	// 创建Echo服务器
	e := echo.New()
	e.Logger.SetOutput(os.Stderr)
	e.HideBanner = true

	// 配置服务器
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout
	e.Server.IdleTimeout = cfg.Server.IdleTimeout

	// 添加中间件
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())
	e.Use(customMiddleware.RateLimit(cfg))

	// 注册路由
	apiserver.RegisterRoutes(e, cfg)

	wailsServerApp = &WailsBackendApp{
		config: cfg,
		server: e,
	}

	// 启动服务器
	go func() {
		if err := wailsServerApp.Start(); err != nil {
			log.Printf("服务器启动失败: %v", err)
			wailsServerMu.Lock()
			wailsServerApp = nil
			wailsServerMu.Unlock()
		}
	}()

	return nil
}

// StopService 停止服务
func (a *WailsApp) StopService() error {
	wailsServerMu.Lock()
	defer wailsServerMu.Unlock()

	if wailsServerApp == nil || wailsServerApp.server == nil {
		return fmt.Errorf("服务未运行")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := wailsServerApp.server.Shutdown(ctx); err != nil {
		wailsServerApp.server.Close()
	}

	wailsServerApp = nil
	return nil
}

// GetServiceStatus 获取服务状态
func (a *WailsApp) GetServiceStatus() ServiceStatus {
	wailsServerMu.Lock()
	defer wailsServerMu.Unlock()

	status := ServiceStatus{
		IsRunning: false,
		Message:   "服务未启动",
	}

	if wailsServerApp != nil && wailsServerApp.server != nil {
		status.IsRunning = true
		status.Message = "服务正在运行"

		cfg := a.configManager.GetConfig()
		testHost := cfg.Server.Host
		if testHost == "0.0.0.0" {
			testHost = "localhost"
		}
		status.Address = fmt.Sprintf("http://%s:%d", testHost, cfg.Server.Port)

		if cfg.Security.BearerToken != "" {
			apiKey := cfg.Security.BearerToken
			if len(apiKey) > 8 {
				apiKey = apiKey[:8] + "..."
			}
			status.APIKey = apiKey
		}
	}

	return status
}

// TestConfig 测试配置
func (a *WailsApp) TestConfig() ([]WailsTestResult, error) {
	cfg := a.configManager.GetConfig()

	// 检查必填项
	if cfg.Monica.Cookie == "" {
		return nil, fmt.Errorf("请填写Monica Cookie")
	}

	if cfg.Security.BearerToken == "" {
		return nil, fmt.Errorf("请填写API Key")
	}

	if cfg.Monica.EnableCustomBotMode && cfg.Monica.BotUID == "" {
		return nil, fmt.Errorf("启用Custom Bot模式时必须填写Bot UID")
	}

	// 检查服务状态
	wailsServerMu.Lock()
	if wailsServerApp == nil {
		wailsServerMu.Unlock()
		return nil, fmt.Errorf("请先启动服务再进行测试")
	}
	wailsServerMu.Unlock()

	// 创建HTTP客户端
	client := resty.New()
	transport := &http.Transport{Proxy: nil}
	client.SetTransport(transport)

	client.SetTimeout(30 * time.Second).
		SetHeaders(map[string]string{
			"Authorization": "Bearer " + cfg.Security.BearerToken,
			"User-Agent":    "Monica-Proxy-Wails/1.0",
			"Content-Type":  "application/json",
		})

	if cfg.Monica.Cookie != "" {
		client.SetHeader("Cookie", cfg.Monica.Cookie)
	}

	// 测试结果
	var results []WailsTestResult
	testHost := cfg.Server.Host
	if testHost == "0.0.0.0" {
		testHost = "localhost"
	}
	baseURL := fmt.Sprintf("http://%s:%d", testHost, cfg.Server.Port)

	// 测试1: 获取模型列表
	resp, err := client.R().Get(baseURL + "/v1/models")
	result := WailsTestResult{
		Endpoint:    "/v1/models",
		URL:         baseURL + "/v1/models",
		RequestData: "",
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.StatusCode = resp.StatusCode()
		result.ResponseData = resp.String()
	}
	results = append(results, result)

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
	resp, err = client.R().SetBody(chatData).Post(baseURL + "/v1/chat/completions")
	result = WailsTestResult{
		Endpoint:    "/v1/chat/completions",
		URL:         baseURL + "/v1/chat/completions",
		RequestData: chatData,
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.StatusCode = resp.StatusCode()
		result.ResponseData = resp.String()
	}
	results = append(results, result)

	// 测试3: 图片生成接口
	imageData := `{
  "prompt": "a white siamese cat",
  "n": 1,
  "size": "512x512"
}`
	resp, err = client.R().SetBody(imageData).Post(baseURL + "/v1/images/generations")
	result = WailsTestResult{
		Endpoint:    "/v1/images/generations",
		URL:         baseURL + "/v1/images/generations",
		RequestData: imageData,
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.StatusCode = resp.StatusCode()
		result.ResponseData = resp.String()
	}
	results = append(results, result)

	return results, nil
}

// GetQuota 获取Monica额度信息
func (a *WailsApp) GetQuota() QuotaInfo {
	cfg := a.configManager.GetConfig()

	if cfg.Monica.Cookie == "" {
		return QuotaInfo{Error: "请先填写Monica Cookie"}
	}

	quotaResp, err := utils.GetMonicaQuota(cfg)
	if err != nil {
		return QuotaInfo{Error: err.Error()}
	}

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

	return QuotaInfo{
		GeniusBot: geniusBotQuota,
		Credits:   creditsQuota,
	}
}

// OpenLogDirectory 打开日志文件所在目录
func (a *WailsApp) OpenLogDirectory() error {
	// 获取用户日志目录
	var logDir string
	if userHome, err := os.UserHomeDir(); err == nil {
		logDir = filepath.Join(userHome, ".monica-proxy", "logs")
	} else {
		// 回退到当前目录
		logDir = "logs"
	}

	// 如果日志目录不存在，创建它
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %v", err)
		}
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(logDir)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	// 根据不同操作系统打开文件管理器
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", absPath)
	case "windows":
		cmd = exec.Command("explorer", absPath)
	case "linux":
		cmd = exec.Command("xdg-open", absPath)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// GetLogFilePath 获取日志文件路径
func (a *WailsApp) GetLogFilePath() string {
	// 优先从日志配置获取路径
	if logPath := logger.GetLogFilePath(); logPath != "" {
		return logPath
	}
	
	// 如果配置中不是文件路径，使用默认路径
	var logPath string
	if userHome, err := os.UserHomeDir(); err == nil {
		logPath = filepath.Join(userHome, ".monica-proxy", "logs", "monica-proxy.log")
	} else {
		// 回退到当前目录
		logPath = "./logs/monica-proxy.log"
	}

	return logPath
}

// newWailsBackendApp 创建Wails后端应用实例
func newWailsBackendApp(cfg *config.Config) *WailsBackendApp {
	// 初始化HTTP客户端
	utils.InitHTTPClients(cfg)

	// 设置 Echo Server
	e := echo.New()
	e.Logger.SetOutput(os.Stderr)
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

	return &WailsBackendApp{
		config: cfg,
		server: e,
	}
}

// Start 启动应用
func (a *WailsBackendApp) Start() error {
	return a.server.Start(a.config.GetAddress())
}

func main() {
	// 创建Wails应用
	app := NewWailsApp()

	// Wails应用配置
	err := wails.Run(&options.App{
		Title:         "Monica Proxy Wails - 基于界面的Monica（Web）转换成 API 的工具",
		Width:         1200,
		Height:        1000,
		MinWidth:      1200,
		MinHeight:     1000,
		MaxWidth:      1200,
		MaxHeight:     1000,
		DisableResize: true,
		AssetServer:   &assetserver.Options{Assets: assets},
		OnStartup:     app.Startup,
		OnDomReady:    app.DomReady,
		OnShutdown:    app.Shutdown,
		Bind:          []interface{}{app},
		Frameless:     false,
	})

	if err != nil {
		log.Fatal(err)
	}
}
