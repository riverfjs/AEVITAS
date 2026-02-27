package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultModel             = "claude-sonnet-4-5-20250929"
	DefaultMaxTokens         = 8192
	DefaultTemperature       = 0.7
	DefaultMaxToolIterations = 20
	DefaultExecTimeout       = 60
	DefaultHost              = "0.0.0.0"
	DefaultPort              = 18790
	DefaultBufSize           = 100
)

type Config struct {
	Agent    AgentConfig    `json:"agent"`
	Channels ChannelsConfig `json:"channels"`
	Provider ProviderConfig `json:"provider"`
	Tools    ToolsConfig    `json:"tools"`
	Gateway  GatewayConfig  `json:"gateway"`
}

type AgentConfig struct {
	Workspace         string        `json:"workspace"`
	Model             string        `json:"model"`
	MaxTokens         int           `json:"maxTokens"`
	Temperature       float64       `json:"temperature"`
	MaxToolIterations int           `json:"maxToolIterations"`
	// HistoryLimit caps the number of user turns loaded from disk into each
	// session context. 0 = no limit (all history). Default: 30.
	HistoryLimit      int           `json:"historyLimit,omitempty"`
	// ToolLog controls real-time progress log messages sent during agent execution.
	ToolLog           ToolLogConfig `json:"toolLog,omitempty"`
	// AutoRecall enables automatic memory injection before each agent turn.
	// When true, MEMORY.md is searched with the user's prompt and top results
	// are prepended to the message as context. Default: true.
	AutoRecall           bool             `json:"autoRecall,omitempty"`
	AutoRecallMaxResults int              `json:"autoRecallMaxResults,omitempty"`
	// Compaction controls automatic context-window compaction.
	Compaction CompactionConfig `json:"compaction,omitempty"`
	// ContextWindow configures the Context Window Guard.
	// When Tokens > 0, the SDK warns when the estimated history token count
	// approaches the limit and rejects requests when too full.
	ContextWindow ContextWindowConfig `json:"contextWindow,omitempty"`
	// MemoryFlush controls automatic pre-compaction memory flush.
	// When enabled, a hidden agent turn runs to write memories to disk when
	// input tokens approach the context window limit.
	MemoryFlush MemoryFlushConfig `json:"memoryFlush,omitempty"`
}

// ToolLogConfig controls periodic tool-call progress messages sent to the chat.
type ToolLogConfig struct {
	Enabled  bool `json:"enabled"`           // Whether to send progress messages (default: false)
	Interval int  `json:"interval,omitempty"` // Send a message every N tool calls (default: 5)
}

// ContextWindowConfig configures the lightweight Context Window Guard.
type ContextWindowConfig struct {
	// Tokens is the total context window size in tokens. 0 = disabled.
	// Set this to your model's context window (e.g. 200000 for Claude 3.5 Sonnet).
	Tokens int `json:"tokens,omitempty"`
	// WarnRatio is the usage fraction (0–1) that triggers a warning message.
	// Default: 0.8 (warn when estimated usage exceeds 80% of Tokens).
	WarnRatio float64 `json:"warnRatio,omitempty"`
	// HardMinTokens is the minimum estimated remaining tokens below which the
	// request is rejected with a message advising /reset. Default: 2000.
	HardMinTokens int `json:"hardMinTokens,omitempty"`
}

// MemoryFlushConfig controls automatic memory flush before context window exhaustion.
// Flush fires when inputTokens >= ContextWindow.Tokens - ReserveTokensFloor - SoftThresholdTokens.
type MemoryFlushConfig struct {
	// Enabled turns memory flush on or off. Default: true.
	Enabled bool `json:"enabled"`
	// ReserveTokensFloor is tokens reserved for model output during the flush turn.
	// Default: 20000 (same as openclaw). Separate from ContextWindow.HardMinTokens.
	ReserveTokensFloor int `json:"reserveTokensFloor,omitempty"`
	// SoftThresholdTokens is the additional early-trigger buffer. Default: 4000 (same as openclaw).
	// With 200k context and defaults, flush triggers at 176k tokens (88% usage).
	SoftThresholdTokens int `json:"softThresholdTokens,omitempty"`
	// Prompt is the synthetic user message for the flush turn. Uses default if empty.
	Prompt string `json:"prompt,omitempty"`
}

// CompactionConfig controls automatic session history compaction.
// When enabled, the SDK monitors token usage and summarises older messages
// via LLM when usage exceeds Threshold * context-window, keeping the
// conversation going indefinitely without hitting the model's context limit.
type CompactionConfig struct {
	// Enabled turns on automatic compaction. Default: false.
	Enabled bool `json:"enabled"`
	// Threshold is the fraction of the context window (0–1) at which compaction
	// triggers. Default: 0.8 (80%).
	Threshold float64 `json:"threshold,omitempty"`
}

type ProviderConfig struct {
	Type    string `json:"type,omitempty"` // "anthropic" (default) or "openai"
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseUrl,omitempty"`
}

type ChannelsConfig struct {
	Telegram TelegramConfig `json:"telegram"`
	Feishu   FeishuConfig   `json:"feishu"`
	WeCom    WeComConfig    `json:"wecom"`
}

type TelegramConfig struct {
	Enabled   bool     `json:"enabled"`
	Token     string   `json:"token"`
	AllowFrom []string `json:"allowFrom"`
	Proxy     string   `json:"proxy,omitempty"`
}

type FeishuConfig struct {
	Enabled           bool     `json:"enabled"`
	AppID             string   `json:"appId"`
	AppSecret         string   `json:"appSecret"`
	VerificationToken string   `json:"verificationToken"`
	EncryptKey        string   `json:"encryptKey,omitempty"`
	Port              int      `json:"port,omitempty"`
	AllowFrom         []string `json:"allowFrom"`
}

type WeComConfig struct {
	Enabled        bool     `json:"enabled"`
	Token          string   `json:"token"`
	EncodingAESKey string   `json:"encodingAESKey"`
	ReceiveID      string   `json:"receiveId,omitempty"`
	Port           int      `json:"port,omitempty"`
	AllowFrom      []string `json:"allowFrom"`
}

type ToolsConfig struct {
	BraveAPIKey         string `json:"braveApiKey,omitempty"`
	ExecTimeout         int    `json:"execTimeout"`
	RestrictToWorkspace bool   `json:"restrictToWorkspace"`
}

type GatewayConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Agent: AgentConfig{
			Workspace:         filepath.Join(home, ".aevitas", "workspace"),
			Model:             DefaultModel,
			MaxTokens:         DefaultMaxTokens,
			Temperature:       DefaultTemperature,
			MaxToolIterations: DefaultMaxToolIterations,
			HistoryLimit:      30,
			ToolLog:           ToolLogConfig{Enabled: false, Interval: 5},
			AutoRecall:        true,
			MemoryFlush: MemoryFlushConfig{Enabled: true, ReserveTokensFloor: 20000, SoftThresholdTokens: 4000},
		},
		Provider: ProviderConfig{},
		Channels: ChannelsConfig{},
		Tools: ToolsConfig{
			ExecTimeout:         DefaultExecTimeout,
			RestrictToWorkspace: true,
		},
		Gateway: GatewayConfig{
			Host: DefaultHost,
			Port: DefaultPort,
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aevitas")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	// Environment variable overrides
	if key := os.Getenv("AEVITAS_API_KEY"); key != "" {
		cfg.Provider.APIKey = key
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" && cfg.Provider.APIKey == "" {
		cfg.Provider.APIKey = key
	}
	if key := os.Getenv("ANTHROPIC_AUTH_TOKEN"); key != "" && cfg.Provider.APIKey == "" {
		cfg.Provider.APIKey = key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" && cfg.Provider.APIKey == "" {
		cfg.Provider.APIKey = key
		if cfg.Provider.Type == "" {
			cfg.Provider.Type = "openai"
		}
	}
	if url := os.Getenv("AEVITAS_BASE_URL"); url != "" {
		cfg.Provider.BaseURL = url
	}
	if url := os.Getenv("ANTHROPIC_BASE_URL"); url != "" && cfg.Provider.BaseURL == "" {
		cfg.Provider.BaseURL = url
	}
	if token := os.Getenv("AEVITAS_TELEGRAM_TOKEN"); token != "" {
		cfg.Channels.Telegram.Token = token
	}
	if appID := os.Getenv("AEVITAS_FEISHU_APP_ID"); appID != "" {
		cfg.Channels.Feishu.AppID = appID
	}
	if appSecret := os.Getenv("AEVITAS_FEISHU_APP_SECRET"); appSecret != "" {
		cfg.Channels.Feishu.AppSecret = appSecret
	}
	if token := os.Getenv("AEVITAS_WECOM_TOKEN"); token != "" {
		cfg.Channels.WeCom.Token = token
	}
	if aesKey := os.Getenv("AEVITAS_WECOM_ENCODING_AES_KEY"); aesKey != "" {
		cfg.Channels.WeCom.EncodingAESKey = aesKey
	}
	if receiveID := os.Getenv("AEVITAS_WECOM_RECEIVE_ID"); receiveID != "" {
		cfg.Channels.WeCom.ReceiveID = receiveID
	}

	if cfg.Agent.Workspace == "" {
		cfg.Agent.Workspace = DefaultConfig().Agent.Workspace
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}
