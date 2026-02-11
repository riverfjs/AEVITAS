package gateway

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/cexll/agentsdk-go/pkg/api"
	"github.com/cexll/agentsdk-go/pkg/core/events"
	sdklogger "github.com/cexll/agentsdk-go/pkg/logger"
	"github.com/cexll/agentsdk-go/pkg/model"
	"github.com/stellarlinkco/myclaw/internal/bus"
	"github.com/stellarlinkco/myclaw/internal/channel"
	"github.com/stellarlinkco/myclaw/internal/config"
	"github.com/stellarlinkco/myclaw/internal/cron"
	"github.com/stellarlinkco/myclaw/internal/heartbeat"
	"github.com/stellarlinkco/myclaw/internal/logger"
	"github.com/stellarlinkco/myclaw/internal/memory"
)

// Runtime interface for agent runtime (allows mocking in tests)
type Runtime interface {
	Run(ctx context.Context, req api.Request) (*api.Response, error)
	ClearSession(sessionID string) error
	Close()
}

// runtimeAdapter wraps api.Runtime to implement Runtime interface
type runtimeAdapter struct {
	rt *api.Runtime
}

func (r *runtimeAdapter) Run(ctx context.Context, req api.Request) (*api.Response, error) {
	return r.rt.Run(ctx, req)
}

func (r *runtimeAdapter) ClearSession(sessionID string) error {
	return r.rt.ClearSession(sessionID)
}

func (r *runtimeAdapter) Close() {
	r.rt.Close()
}

// RuntimeFactory creates a Runtime instance
type RuntimeFactory func(cfg *config.Config, sysPrompt string) (Runtime, error)

// Options for creating a Gateway
type Options struct {
	RuntimeFactory RuntimeFactory
	SignalChan     chan os.Signal // for testing signal handling
}

// DefaultRuntimeFactory creates the default agentsdk-go runtime
func DefaultRuntimeFactory(cfg *config.Config, sysPrompt string) (Runtime, error) {
	// 初始化 logger - 默认启用 debug 日志
	debug := true // 始终启用详细日志
	zapLogger, err := logger.InitLogger(cfg.Agent.Workspace, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	sdkLog := sdklogger.NewZapLogger(zapLogger)

	var provider api.ModelFactory
	switch cfg.Provider.Type {
	case "openai":
		provider = &model.OpenAIProvider{
			APIKey:    cfg.Provider.APIKey,
			BaseURL:   cfg.Provider.BaseURL,
			ModelName: cfg.Agent.Model,
			MaxTokens: cfg.Agent.MaxTokens,
		}
	default: // "anthropic" or empty
		provider = &model.AnthropicProvider{
			APIKey:    cfg.Provider.APIKey,
			BaseURL:   cfg.Provider.BaseURL,
			ModelName: cfg.Agent.Model,
			MaxTokens: cfg.Agent.MaxTokens,
		}
	}

	rt, err := api.New(context.Background(), api.Options{
		ProjectRoot:   cfg.Agent.Workspace,
		ModelFactory:  provider,
		SystemPrompt:  sysPrompt,
		MaxIterations: cfg.Agent.MaxToolIterations,
		Logger:        sdkLog,
	})
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}
	return &runtimeAdapter{rt: rt}, nil
}

type Gateway struct {
	cfg           *config.Config
	bus           *bus.MessageBus
	runtime       Runtime
	runtimeFactory RuntimeFactory // Factory to recreate runtime on restart
	channels      *channel.ChannelManager
	cron          *cron.Service
	hb            *heartbeat.Service
	mem           *memory.MemoryStore
	cmdHandler    *channel.CommandHandler
	signalChan    chan os.Signal // for testing
	logger        sdklogger.Logger
}

// New creates a Gateway with default options
func New(cfg *config.Config) (*Gateway, error) {
	return NewWithOptions(cfg, Options{})
}

// NewWithOptions creates a Gateway with custom options for testing
func NewWithOptions(cfg *config.Config, opts Options) (*Gateway, error) {
	// 初始化 logger
	debug := true // 始终启用详细日志
	zapLogger, err := logger.InitLogger(cfg.Agent.Workspace, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	g := &Gateway{
		cfg:    cfg,
		logger: sdklogger.NewZapLogger(zapLogger),
	}

	// Message bus
	g.bus = bus.NewMessageBus(config.DefaultBufSize)

	// Memory
	g.mem = memory.NewMemoryStore(cfg.Agent.Workspace)

	// Build system prompt
	sysPrompt := g.buildSystemPrompt()

	// Create runtime using factory (allows injection for testing)
	factory := opts.RuntimeFactory
	if factory == nil {
		factory = DefaultRuntimeFactory
	}
	g.runtimeFactory = factory // Save factory for restart
	rt, err := factory(cfg, sysPrompt)
	if err != nil {
		return nil, err
	}
	g.runtime = rt

	// Signal channel for testing
	g.signalChan = opts.SignalChan

	// runAgent helper for cron/heartbeat
	runAgent := func(prompt string) (string, error) {
		resp, err := g.runtime.Run(context.Background(), api.Request{
			Prompt:    prompt,
			SessionID: "system",
		})
		if err != nil {
			return "", err
		}
		if resp == nil || resp.Result == nil {
			return "", nil
		}
		return resp.Result.Output, nil
	}

	// Cron
	cronStorePath := filepath.Join(config.ConfigDir(), "data", "cron", "jobs.json")
	g.cron = cron.NewService(cronStorePath, g.logger)
	g.cron.OnJob = func(job cron.CronJob) (string, error) {
		result, err := runAgent(job.Payload.Message)
		if err != nil {
			return "", err
		}
		if job.Payload.Deliver && job.Payload.Channel != "" {
			g.bus.Outbound <- bus.OutboundMessage{
				Channel: job.Payload.Channel,
				ChatID:  job.Payload.To,
				Content: result,
			}
		}
		return result, nil
	}

	// Heartbeat
	g.hb = heartbeat.New(cfg.Agent.Workspace, runAgent, 0, g.logger)

	// Command handler
	g.cmdHandler = channel.NewCommandHandler(g.runtime, cfg.Agent.Workspace)

	// Channels
	chMgr, err := channel.NewChannelManager(cfg.Channels, g.bus, g.logger)
	if err != nil {
		return nil, fmt.Errorf("create channel manager: %w", err)
	}
	g.channels = chMgr

	return g, nil
}

func (g *Gateway) buildSystemPrompt() string {
	var sb strings.Builder

	if data, err := os.ReadFile(filepath.Join(g.cfg.Agent.Workspace, "AGENTS.md")); err == nil {
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	if data, err := os.ReadFile(filepath.Join(g.cfg.Agent.Workspace, "SOUL.md")); err == nil {
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	if memCtx := g.mem.GetMemoryContext(); memCtx != "" {
		sb.WriteString(memCtx)
	}

	return sb.String()
}

// start initializes and starts all gateway services
func (g *Gateway) start(ctx context.Context) error {
	go g.bus.DispatchOutbound(ctx)

	if err := g.channels.StartAll(ctx); err != nil {
		return fmt.Errorf("start channels: %w", err)
	}
	g.logger.Infof("[gateway] channels started: %v", g.channels.EnabledChannels())

	if err := g.cron.Start(ctx); err != nil {
		g.logger.Warnf("[gateway] cron start warning: %v", err)
	}

	go func() {
		if err := g.hb.Start(ctx); err != nil {
			g.logger.Errorf("[gateway] heartbeat error: %v", err)
		}
	}()

	go g.processLoop(ctx)

	g.logger.Infof("[gateway] running on %s:%d", g.cfg.Gateway.Host, g.cfg.Gateway.Port)
	return nil
}

func (g *Gateway) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initial start
	if err := g.start(ctx); err != nil {
		return err
	}

	// Use injected signal channel for testing, or create default
	sigCh := g.signalChan
	if sigCh == nil {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	}
	sig := <-sigCh
	g.logger.Infof("[gateway] shutdown signal received: %v", sig)
	cancel() // Cancel context to stop all goroutines
	g.logger.Infof("[gateway] shutting down...")
	return g.Shutdown()
}

func (g *Gateway) processLoop(ctx context.Context) {
	for {
		select {
		case msg := <-g.bus.Inbound:
			g.logger.Infof("[gateway] inbound from %s/%s: %s", msg.Channel, msg.SenderID, truncate(msg.Content, 80))

			// Check if this is a special command
			cmdResult := g.cmdHandler.HandleCommand(msg)
			if cmdResult.Handled {
				g.logger.Infof("[gateway] command handled: %s", truncate(msg.Content, 40))
				if cmdResult.Response != "" {
					g.bus.Outbound <- bus.OutboundMessage{
						Channel: msg.Channel,
						ChatID:  msg.ChatID,
						Content: cmdResult.Response,
					}
				}
				continue
			}

			// 异步处理 agent
			go g.processAgent(ctx, msg)
		case <-ctx.Done():
			return
		}
	}
}

func (g *Gateway) processAgent(ctx context.Context, msg bus.InboundMessage) {
	// Build attachments from media
	var attachments []api.Attachment
	for _, mediaPath := range msg.Media {
		// Detect mime type from file extension
		mimeType := detectMimeType(mediaPath)
		attachments = append(attachments, api.Attachment{
			Type:     "image",
			FilePath: mediaPath,
			MimeType: mimeType,
		})
	}
	
	if len(attachments) > 0 {
		g.logger.Infof("[gateway] processing %d image(s)", len(attachments))
	}

	resp, err := g.runtime.Run(ctx, api.Request{
		Prompt:      msg.Content,
		SessionID:   msg.SessionKey(),
		Attachments: attachments,
	})
	
	if err != nil {
		g.logger.Errorf("[gateway] agent error: %v", err)
		
		// Provide user-friendly error messages
		var errorMsg string
		if strings.Contains(err.Error(), "max iterations reached") {
			errorMsg = "抱歉，这个任务太复杂了，我尝试了太多次工具调用。请简化您的请求或分步骤提问。"
		} else if strings.Contains(err.Error(), "context deadline exceeded") {
			errorMsg = "抱歉，处理超时了。请稍后再试或简化您的请求。"
		} else {
			errorMsg = "抱歉，处理您的消息时遇到了错误。"
		}
		
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: errorMsg,
		}
		return
	}
	
	// Extract screenshot paths from tool results
	screenshotPaths := g.extractScreenshotPaths(resp)
	
	// 构建响应内容（优先级：Commands > Skills > AskUserQuestion > Subagent > Result.Output）
	var content strings.Builder
	
	// 1. Command results（如 /help）
	for _, cmdRes := range resp.CommandResults {
		if output, ok := cmdRes.Result.Output.(string); ok && output != "" {
			content.WriteString(output)
			content.WriteString("\n\n")
		}
	}
	
	// 2. Skill results
	for _, skillRes := range resp.SkillResults {
		if output, ok := skillRes.Result.Output.(string); ok && output != "" {
			content.WriteString(output)
			content.WriteString("\n\n")
		}
	}
	
	// 3. AskUserQuestion（通过 HookEvents）
	if question := g.extractAskUserQuestion(resp); question != "" {
		g.logger.Infof("[gateway] AskUserQuestion: %s", truncate(question, 60))
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: question,
		}
		return
	}
	
	// 4. Subagent result
	if resp.Subagent != nil {
		if output, ok := resp.Subagent.Output.(string); ok && output != "" {
			content.WriteString(output)
			content.WriteString("\n\n")
		}
	}
	
	// 5. Main agent output
	if resp.Result != nil && resp.Result.Output != "" {
		content.WriteString(resp.Result.Output)
	}
	
	// 6. Append screenshot paths if any
	if len(screenshotPaths) > 0 {
		for _, path := range screenshotPaths {
			content.WriteString("\n")
			content.WriteString(path)
		}
	}
	
	result := strings.TrimSpace(content.String())
	if result != "" {
		g.logger.Infof("[gateway] outbound to %s/%s: %s", msg.Channel, msg.ChatID, truncate(result, 80))
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: result,
		}
	} else {
		g.logger.Warnf("[gateway] no response generated for %s/%s", msg.Channel, msg.SenderID)
	}
}

// extractAskUserQuestion extracts formatted question from AskUserQuestion tool execution
func (g *Gateway) extractAskUserQuestion(resp *api.Response) string {
	if resp == nil {
		return ""
	}
	
	// Collect tool names for summary logging
	var toolNames []string
	var askUserResult string
	
	for _, event := range resp.HookEvents {
		if event.Type != events.PostToolUse {
			continue
		}
		
		payload, ok := event.Payload.(events.ToolResultPayload)
		if !ok {
			continue
		}
		
		toolNames = append(toolNames, payload.Name)
		
		if payload.Name != "AskUserQuestion" && payload.Name != "ask_user_question" {
			continue
		}
		
		// payload.Result is the formatted question string
		if output, ok := payload.Result.(string); ok && output != "" {
			askUserResult = output
		}
	}
	
	// Log summary if tools were used
	if len(toolNames) > 0 {
		g.logger.Debugf("[gateway] PostToolUse: used %d tool(s): %v", len(toolNames), toolNames)
	}
	
	return askUserResult
}

// extractScreenshotPaths extracts screenshot file paths from tool execution results
func (g *Gateway) extractScreenshotPaths(resp *api.Response) []string {
	if resp == nil {
		return nil
	}
	
	var paths []string
	
	for _, event := range resp.HookEvents {
		if event.Type != events.PostToolUse {
			continue
		}
		
		payload, ok := event.Payload.(events.ToolResultPayload)
		if !ok {
			continue
		}
		
		// Check if it's a Bash tool call
		if payload.Name != "Bash" {
			continue
		}
		
		// Try to parse Result as a map
		if resultMap, ok := payload.Result.(map[string]interface{}); ok {
			// Check if it contains a "path" field with screenshot
			if pathStr, ok := resultMap["path"].(string); ok {
				if strings.Contains(pathStr, "screenshot-") && strings.HasSuffix(pathStr, ".png") {
					paths = append(paths, pathStr)
					g.logger.Debugf("[gateway] Found screenshot: %s", pathStr)
				}
			}
		}
		
		// Also try string result (in case it's JSON string)
		if resultStr, ok := payload.Result.(string); ok {
			// Simple regex to extract screenshot paths
			re := regexp.MustCompile(`(/[^\s"]+/screenshot-[0-9]+\.png)`)
			matches := re.FindAllString(resultStr, -1)
			for _, match := range matches {
				paths = append(paths, match)
				g.logger.Debugf("[gateway] Found screenshot: %s", match)
			}
		}
	}
	
	return paths
}

func (g *Gateway) Shutdown() error {
	g.cron.Stop()
	_ = g.channels.StopAll()
	if g.runtime != nil {
		g.runtime.Close()
	}
	g.logger.Infof("[gateway] shutdown complete")
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// detectMimeType detects MIME type from file extension
func detectMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg" // default
	}
}


