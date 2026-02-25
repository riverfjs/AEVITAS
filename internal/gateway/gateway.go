package gateway

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
	"github.com/stellarlinkco/myclaw/internal/rpc"
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
type RuntimeFactory func(cfg *config.Config, sysPrompt string, realtimeCallback func(api.RealtimeEvent)) (Runtime, error)

// Options for creating a Gateway
type Options struct {
	RuntimeFactory RuntimeFactory
	SignalChan     chan os.Signal // for testing signal handling
}

// DefaultRuntimeFactory creates the default agentsdk-go runtime
func DefaultRuntimeFactory(cfg *config.Config, sysPrompt string, realtimeCallback func(api.RealtimeEvent)) (Runtime, error) {
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
		ProjectRoot:           cfg.Agent.Workspace,
		ModelFactory:          provider,
		SystemPrompt:          sysPrompt,
		MaxIterations:         cfg.Agent.MaxToolIterations,
		Logger:                sdkLog,
		HistoryLimit:          cfg.Agent.HistoryLimit,
		RealtimeEventCallback: realtimeCallback,
		ProgressInterval:      cfg.Agent.ToolLog.Interval,
		AutoRecall:            cfg.Agent.AutoRecall,
		AutoRecallMaxResults:  cfg.Agent.AutoRecallMaxResults,
		AutoCompact: api.CompactConfig{
			Enabled:   cfg.Agent.Compaction.Enabled,
			Threshold: cfg.Agent.Compaction.Threshold,
		},
		ContextWindowTokens:        cfg.Agent.ContextWindow.Tokens,
		ContextWindowWarnRatio:     cfg.Agent.ContextWindow.WarnRatio,
		ContextWindowHardMinTokens: cfg.Agent.ContextWindow.HardMinTokens,
		MemoryFlush: api.MemoryFlushConfig{
			Enabled:             cfg.Agent.MemoryFlush.Enabled,
			ReserveTokensFloor:  cfg.Agent.MemoryFlush.ReserveTokensFloor,
			SoftThresholdTokens: cfg.Agent.MemoryFlush.SoftThresholdTokens,
			Prompt:              cfg.Agent.MemoryFlush.Prompt,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create runtime: %w", err)
	}
	return &runtimeAdapter{rt: rt}, nil
}

type Gateway struct {
	cfg            *config.Config
	bus            *bus.MessageBus
	runtime        Runtime
	runtimeFactory RuntimeFactory // Factory to recreate runtime on restart
	channels       *channel.ChannelManager
	cron           *cron.Service
	hb             *heartbeat.Service
	cmdHandler     *channel.CommandHandler
	signalChan     chan os.Signal // for testing
	logger         sdklogger.Logger

	// Current execution context (for realtime callbacks)
	currentChannelID string
	currentChatID    string
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

	// Build system prompt
	sysPrompt := g.buildSystemPrompt()

	// Build real-time event callback.
	// Progress updates require toolLog.enabled; context window warnings always fire.
	realtimeCallback := func(event api.RealtimeEvent) {
		g.logger.Infof("[gateway] Realtime event: type=%s, count=%d, tool=%s", event.Type, event.Count, event.LastTool)
		if g.currentChannelID == "" || g.currentChatID == "" {
			return
		}

		var msg string
		switch event.Type {
		case api.RealtimeEventContextWindowWarn:
			// Always forward context window warnings — the user must know.
			msg = event.Message

		case api.RealtimeEventProgressUpdate:
			// Only forward progress updates when toolLog is enabled.
			if !cfg.Agent.ToolLog.Enabled {
				return
			}
			msg = fmt.Sprintf("⏳ %s", event.LastTool)
			if len(event.RecentCalls) > 0 {
				params := event.RecentCalls[0].Params
				if params != "" && params != "{}" {
					msg += fmt.Sprintf(": %s", params)
				}
			}

		default:
			return
		}

		if msg == "" {
			return
		}
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: g.currentChannelID,
			ChatID:  g.currentChatID,
			Content: msg,
		}
		g.logger.Debugf("[gateway] Sent %s event to %s/%s", event.Type, g.currentChannelID, g.currentChatID)
	}

	// Create runtime using factory (allows injection for testing)
	factory := opts.RuntimeFactory
	if factory == nil {
		factory = DefaultRuntimeFactory
	}
	g.runtimeFactory = factory // Save factory for restart
	rt, err := factory(cfg, sysPrompt, realtimeCallback)
	if err != nil {
		return nil, err
	}
	g.runtime = rt

	// Signal channel for testing
	g.signalChan = opts.SignalChan

	// Cron
	cronStorePath := filepath.Join(config.ConfigDir(), "data", "cron", "jobs.json")
	g.cron = cron.NewService(cronStorePath, g.logger)
	g.cron.OnJob = func(job cron.CronJob) (string, error) {
		var result string
		var err error

		sessionID := "system"
		if job.SessionTarget == cron.SessionIsolated {
			sessionID = fmt.Sprintf("cron-isolated-%s", job.ID)
		}

		switch job.Payload.Kind {
		case "command":
			// Direct exec: bypass agent entirely, stdout is the result
			out, execErr := exec.Command("bash", "-c", job.Payload.Command).Output()
			if execErr != nil {
				result = fmt.Sprintf("command error: %v\n%s", execErr, string(out))
			} else {
				result = string(out)
			}

		case "systemEvent":
			// Inject text as system event — no agent turn, result is the text itself
			result = job.Payload.Text

		default:
			// "agentTurn" or legacy (empty kind with message field)
			msg := job.Payload.Message
			if msg == "" {
				msg = job.Payload.Text
			}
			result, err = g.runAgent(context.Background(), msg, sessionID)
			if err != nil {
				return "", err
			}
		}

		// Deliver result via Delivery config (new style)
		if d := job.Delivery; d != nil && d.Mode == "announce" && d.Channel != "" {
			g.bus.Outbound <- bus.OutboundMessage{
				Channel: d.Channel,
				ChatID:  d.To,
				Content: result,
			}
		}
		return result, nil
	}

	// Heartbeat
	g.hb = heartbeat.New(cfg.Agent.Workspace, func(prompt string) (string, error) {
		return g.runAgent(context.Background(), prompt, "system")
	}, g.heartbeatNotify, 0, g.logger)

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

// resetSession clears the session history for the given sessionID.
func (g *Gateway) resetSession(sessionID string) error {
	if err := g.runtime.ClearSession(sessionID); err != nil {
		return fmt.Errorf("failed to clear session: %w", err)
	}
	return nil
}

// runAgent runs the agent with the given prompt and sessionID, returning the text output.
func (g *Gateway) runAgent(ctx context.Context, prompt, sessionID string) (string, error) {
	resp, err := g.runtime.Run(ctx, api.Request{
		Prompt:    prompt,
		SessionID: sessionID,
	})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Result == nil {
		return "", nil
	}
	return resp.Result.Output, nil
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

	// Start WebSocket RPC server (same protocol as openclaw)
	rpcAddr := fmt.Sprintf("%s:%d", g.cfg.Gateway.Host, g.cfg.Gateway.Port)
	rpcSrv := rpc.NewServer(g.logger)
	rpc.RegisterCronHandlers(rpcSrv, g.cron)
	if err := rpcSrv.Start(ctx, rpcAddr); err != nil {
		return fmt.Errorf("rpc server: %w", err)
	}

	g.logger.Infof("[gateway] running on ws://%s", rpcAddr)

	// Send startup notification
	g.sendStartupNotification()
	
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
			var cmdResult channel.CommandResult
			if g.cmdHandler != nil {
				cmdResult = g.cmdHandler.HandleCommand(msg)
			}
			if cmdResult.Handled {
				g.logger.Infof("[gateway] command handled: %s", truncate(msg.Content, 40))
				
				// Stop typing indicator for commands
				if stopTyping, ok := msg.Metadata["stop_typing"].(chan struct{}); ok {
					close(stopTyping)
				}
				
				if cmdResult.Response != "" || len(cmdResult.Files) > 0 {
					g.bus.Outbound <- bus.OutboundMessage{
						Channel: msg.Channel,
						ChatID:  msg.ChatID,
						Content: cmdResult.Response,
						Media:   cmdResult.Files,
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
	// Set current execution context for realtime callbacks
	g.currentChannelID = msg.Channel
	g.currentChatID = msg.ChatID
	defer func() {
		g.currentChannelID = ""
		g.currentChatID = ""
	}()
	
	// Stop typing indicator when processing completes (deferred)
	if stopTyping, ok := msg.Metadata["stop_typing"].(chan struct{}); ok {
		defer close(stopTyping)
	}
	
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

	// 3 & 4. Process all hook events in one pass
	hookResult := g.processHookEvents(resp)

	// Send files
	for _, filePath := range hookResult.sendFiles {
		g.logger.Infof("[gateway] SendFile detected: %s", filePath)
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Media:   []string{filePath},
		}
	}

	// AskUserQuestion takes priority and short-circuits
	if hookResult.askQuestion != "" {
		g.logger.Infof("[gateway] AskUserQuestion: %s", truncate(hookResult.askQuestion, 60))
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: hookResult.askQuestion,
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
	
	result := strings.TrimSpace(content.String())
	
	// Send message (without auto-extracting attachments)
	if result != "" {
		g.logger.Infof("[gateway] outbound to %s/%s: %s", msg.Channel, msg.ChatID, truncate(result, 80))
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: result,
		}
	} else if len(hookResult.sendFiles) == 0 {
		g.logger.Warnf("[gateway] no response generated for %s/%s", msg.Channel, msg.SenderID)
	}

	// Notify user about memory writes
	if hookResult.memoryNotice != "" {
		g.bus.Outbound <- bus.OutboundMessage{
			Channel: msg.Channel,
			ChatID:  msg.ChatID,
			Content: hookResult.memoryNotice,
		}
	}
}

// hookEventResult holds all data extracted from a single pass over resp.HookEvents.
type hookEventResult struct {
	sendFiles    []string
	askQuestion  string
	memoryNotice string
}

// processHookEvents iterates resp.HookEvents once and extracts all relevant data.
func (g *Gateway) processHookEvents(resp *api.Response) hookEventResult {
	if resp == nil {
		return hookEventResult{}
	}

	type memWrite struct {
		path  string
		bytes int
	}
	var (
		res       hookEventResult
		toolNames []string
		memWrites []memWrite
	)

	for _, event := range resp.HookEvents {
		switch event.Type {
		case events.PostToolUse:
			payload, ok := event.Payload.(events.ToolResultPayload)
			if !ok {
				continue
			}
			toolNames = append(toolNames, payload.Name)

			switch payload.Name {
			case "AskUserQuestion", "ask_user_question":
				if output, ok := payload.Result.(string); ok && output != "" {
					res.askQuestion = output
				}

			case "memory_write":
				if payload.Err != nil {
					continue
				}
				path, _ := payload.Params["path"].(string)
				if path == "" {
					path = "memory"
				}
				n := 0
				if output, ok := payload.Result.(string); ok {
					fmt.Sscanf(output, "Appended %d", &n)
					if n == 0 {
						fmt.Sscanf(output, "Written %d", &n)
					}
				}
				memWrites = append(memWrites, memWrite{path: path, bytes: n})
			}

		case events.FileAttachment:
			payload, ok := event.Payload.(events.FileAttachmentPayload)
			if !ok {
				continue
			}
			g.logger.Debugf("[gateway] FileAttachment: path=%s tool=%s", payload.FilePath, payload.ToolName)
			if payload.ToolName == "SendFile" && payload.FilePath != "" {
				res.sendFiles = append(res.sendFiles, payload.FilePath)
			}
		}
	}

	if len(toolNames) > 0 {
		g.logger.Debugf("[gateway] PostToolUse: used %d tool(s): %v", len(toolNames), toolNames)
	}
	if len(res.sendFiles) > 0 {
		g.logger.Infof("[gateway] Extracted %d file(s) from SendFile tool", len(res.sendFiles))
	}

	// Build memory notice
	if len(memWrites) > 0 {
		parts := make([]string, 0, len(memWrites))
		for _, w := range memWrites {
			if w.bytes > 0 {
				parts = append(parts, fmt.Sprintf("📝 %s (+%d bytes)", w.path, w.bytes))
			} else {
				parts = append(parts, fmt.Sprintf("📝 %s", w.path))
			}
		}
		res.memoryNotice = strings.Join(parts, "\n")
	}

	return res
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

// sendStartupNotification sends a startup message to the user who triggered restart
func (g *Gateway) sendStartupNotification() {
	// Check if there's a restart trigger file
	restartTriggerFile := filepath.Join(os.Getenv("HOME"), ".myclaw", "restart_trigger.txt")
	data, err := os.ReadFile(restartTriggerFile)
	if err != nil {
		// No trigger file, skip notification
		g.logger.Debug("[gateway] no restart trigger file found, skipping startup notification")
		return
	}
	
	// Parse channel:chatID format
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	if len(parts) != 2 {
		g.logger.Warnf("[gateway] invalid restart trigger format: %s", string(data))
		return
	}
	
	channelName := parts[0]
	chatID := parts[1]
	
	// Get PID for status
	pid := os.Getpid()
	
	startupMsg := fmt.Sprintf("✅ **Gateway Restarted Successfully**\n\nPID: %d\nTime: %s",
		pid, time.Now().Format("2006-01-02 15:04:05"))
	
	g.logger.Infof("[gateway] sending restart notification to %s/%s", channelName, chatID)
	
	// Send notification
	g.bus.Outbound <- bus.OutboundMessage{
		Channel: channelName,
		ChatID:  chatID,
		Content: startupMsg,
	}
	
	// Clean up trigger file
	os.Remove(restartTriggerFile)
}

// heartbeatNotify delivers a heartbeat agent response to the user.
// It sends to the last active session, falling back to the first configured
// Telegram allowFrom user if no session is currently active.
func (g *Gateway) heartbeatNotify(result string) {
	channelID := g.currentChannelID
	chatID := g.currentChatID

	// Fallback: use first Telegram allowFrom
	if channelID == "" || chatID == "" {
		if len(g.cfg.Channels.Telegram.AllowFrom) > 0 {
			channelID = "telegram"
			chatID = g.cfg.Channels.Telegram.AllowFrom[0]
		}
	}

	if channelID == "" || chatID == "" {
		g.logger.Warnf("[heartbeat] cannot notify: no active session and no allowFrom configured")
		return
	}

	g.logger.Infof("[heartbeat] notifying user channel=%s chatID=%s", channelID, chatID)
	g.bus.Outbound <- bus.OutboundMessage{
		Channel: channelID,
		ChatID:  chatID,
		Content: result,
	}
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


