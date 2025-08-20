package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `yaml:"server" json:"server"`

	// Monica API 配置
	Monica MonicaConfig `yaml:"monica" json:"monica"`

	// 安全配置
	Security SecurityConfig `yaml:"security" json:"security"`

	// HTTP 客户端配置
	HTTPClient HTTPClientConfig `yaml:"http_client" json:"http_client"`

	// 日志配置
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// 代理配置
	Proxy ProxyConfig `yaml:"proxy" json:"proxy"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

// MonicaConfig Monica API 配置
type MonicaConfig struct {
	Cookie              string `yaml:"cookie" json:"cookie"`
	BotUID              string `yaml:"bot_uid" json:"bot_uid"`
	EnableCustomBotMode bool   `yaml:"enable_custom_bot_mode" json:"enable_custom_bot_mode"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	BearerToken      string        `yaml:"bearer_token" json:"bearer_token"`
	TLSSkipVerify    bool          `yaml:"tls_skip_verify" json:"tls_skip_verify"`
	RateLimitEnabled bool          `yaml:"rate_limit_enabled" json:"rate_limit_enabled"`
	RateLimitRPS     int           `yaml:"rate_limit_rps" json:"rate_limit_rps"`
	RequestTimeout   time.Duration `yaml:"request_timeout" json:"request_timeout"`
}

// HTTPClientConfig HTTP 客户端配置
type HTTPClientConfig struct {
	Timeout             time.Duration `yaml:"timeout" json:"timeout"`
	MaxIdleConns        int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`
	MaxConnsPerHost     int           `yaml:"max_conns_per_host" json:"max_conns_per_host"`
	RetryCount          int           `yaml:"retry_count" json:"retry_count"`
	RetryWaitTime       time.Duration `yaml:"retry_wait_time" json:"retry_wait_time"`
	RetryMaxWaitTime    time.Duration `yaml:"retry_max_wait_time" json:"retry_max_wait_time"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level            string `yaml:"level" json:"level"`
	Format           string `yaml:"format" json:"format"`
	Output           string `yaml:"output" json:"output"`
	EnableRequestLog bool   `yaml:"enable_request_log" json:"enable_request_log"`
	MaskSensitive    bool   `yaml:"mask_sensitive" json:"mask_sensitive"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	HTTPProxy  string `yaml:"http_proxy" json:"http_proxy"`
	HTTPSProxy string `yaml:"https_proxy" json:"https_proxy"`
	NoProxy    string `yaml:"no_proxy" json:"no_proxy"`
}

// Load 加载配置，优先级：配置文件 > 环境变量 > 默认值
func Load() (*Config, error) {
	// 1. 设置默认配置
	config := getDefaultConfig()

	// 2. 尝试加载 .env 文件
	_ = godotenv.Load()

	// 3. 尝试加载配置文件
	if err := loadConfigFile(config); err != nil {
		// 配置文件加载失败不是致命错误，继续使用环境变量和默认值
		fmt.Printf("Warning: Failed to load config file: %v\n", err)
	}

	// 4. 环境变量覆盖
	overrideWithEnv(config)

	// 5. 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  5 * time.Minute,
			WriteTimeout: 5 * time.Minute,
			IdleTimeout:  60 * time.Second,
		},
		Monica: MonicaConfig{
			Cookie:              "",
			BotUID:              "",
			EnableCustomBotMode: false,
		},
		Security: SecurityConfig{
			TLSSkipVerify:    true,
			RateLimitEnabled: false, // 默认禁用，需要明确配置
			RateLimitRPS:     0,     // 默认0，禁用限流
			RequestTimeout:   30 * time.Second,
		},
		HTTPClient: HTTPClientConfig{
			Timeout:             3 * time.Minute,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     50,
			RetryCount:          3,
			RetryWaitTime:       1 * time.Second,
			RetryMaxWaitTime:    10 * time.Second,
		},
		Logging: LoggingConfig{
			Level:            "info",
			Format:           "json",
			Output:           "stdout",
			EnableRequestLog: true,
			MaskSensitive:    true,
		},
		Proxy: ProxyConfig{
			HTTPProxy:  "",
			HTTPSProxy: "",
			NoProxy:    "",
		},
	}
}

// getDefaultConfig 获取默认配置 (保持向后兼容)
func getDefaultConfig() *Config {
	return GetDefaultConfig()
}

// loadConfigFile 加载配置文件
func loadConfigFile(config *Config) error {
	configPaths := []string{
		"config.yaml",
		"config.yml",
		"config.json",
		"./configs/config.yaml",
		"./configs/config.yml",
		"./configs/config.json",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return loadFromFile(path, config)
		}
	}

	// 检查环境变量指定的配置文件
	if configPath := os.Getenv("CONFIG_FILE"); configPath != "" {
		return loadFromFile(configPath, config)
	}

	return fmt.Errorf("no config file found")
}

// loadFromFile 从文件加载配置
func loadFromFile(path string, config *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, config)
	case ".json":
		return json.Unmarshal(data, config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}
}

// overrideWithEnv 用环境变量覆盖配置
func overrideWithEnv(config *Config) {
	// 服务器配置
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	// 保持向后兼容，支持原来的PORT环境变量
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if timeout := os.Getenv("SERVER_READ_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			config.Server.ReadTimeout = t
		}
	}

	// Monica 配置
	if cookie := os.Getenv("MONICA_COOKIE"); cookie != "" {
		config.Monica.Cookie = cookie
	}
	if botUID := os.Getenv("BOT_UID"); botUID != "" {
		config.Monica.BotUID = botUID
	}
	if enableCustomBot := os.Getenv("ENABLE_CUSTOM_BOT_MODE"); enableCustomBot != "" {
		if enabled, err := strconv.ParseBool(enableCustomBot); err == nil {
			config.Monica.EnableCustomBotMode = enabled
		}
	}

	// 安全配置
	if token := os.Getenv("BEARER_TOKEN"); token != "" {
		config.Security.BearerToken = token
	}
	if skipVerify := os.Getenv("TLS_SKIP_VERIFY"); skipVerify != "" {
		if skip, err := strconv.ParseBool(skipVerify); err == nil {
			config.Security.TLSSkipVerify = skip
		}
	}
	if rateLimitEnabled := os.Getenv("RATE_LIMIT_ENABLED"); rateLimitEnabled != "" {
		if enabled, err := strconv.ParseBool(rateLimitEnabled); err == nil {
			config.Security.RateLimitEnabled = enabled
		}
	}
	if rateLimitRPS := os.Getenv("RATE_LIMIT_RPS"); rateLimitRPS != "" {
		if rps, err := strconv.Atoi(rateLimitRPS); err == nil {
			config.Security.RateLimitRPS = rps
		}
	}

	// 日志配置
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}
	
	// 代理配置
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		config.Proxy.HTTPProxy = httpProxy
	}
	if httpsProxy := os.Getenv("HTTPS_PROXY"); httpsProxy != "" {
		config.Proxy.HTTPSProxy = httpsProxy
	}
	if noProxy := os.Getenv("NO_PROXY"); noProxy != "" {
		config.Proxy.NoProxy = noProxy
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	var errors []string

	// 验证必要配置
	if c.Monica.Cookie == "" {
		errors = append(errors, "MONICA_COOKIE is required")
	}
	if c.Security.BearerToken == "" {
		errors = append(errors, "BEARER_TOKEN is required")
	}

	// 如果启用了 Custom Bot 模式，必须设置 BOT_UID
	if c.Monica.EnableCustomBotMode && c.Monica.BotUID == "" {
		errors = append(errors, "BOT_UID is required when ENABLE_CUSTOM_BOT_MODE is true")
	}

	// 验证端口范围
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errors = append(errors, "SERVER_PORT must be between 1 and 65535")
	}

	// 验证超时配置
	if c.Server.ReadTimeout < 0 {
		errors = append(errors, "SERVER_READ_TIMEOUT must be positive")
	}
	if c.HTTPClient.Timeout < 0 {
		errors = append(errors, "HTTP_CLIENT_TIMEOUT must be positive")
	}

	// 验证限流配置
	if c.Security.RateLimitRPS <= 0 {
		// 如果RPS<=0，自动禁用限流
		c.Security.RateLimitEnabled = false
	}
	if c.Security.RateLimitEnabled && c.Security.RateLimitRPS <= 0 {
		errors = append(errors, "RATE_LIMIT_RPS must be positive when rate limiting is enabled")
	}
	if c.Security.RateLimitRPS > 10000 {
		errors = append(errors, "RATE_LIMIT_RPS should not exceed 10000 for performance reasons")
	}

	// 验证日志级别
	validLevels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}
	if !contains(validLevels, c.Logging.Level) {
		errors = append(errors, fmt.Sprintf("LOG_LEVEL must be one of: %s", strings.Join(validLevels, ", ")))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// GetAddress 获取服务器监听地址
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
