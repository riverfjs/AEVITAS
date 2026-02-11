package channel

import (
	"fmt"
	"strings"

	"github.com/stellarlinkco/myclaw/internal/bus"
	"github.com/stellarlinkco/myclaw/pkg/utils"
)

// SessionResetter is an interface for clearing sessions
type SessionResetter interface {
	ClearSession(sessionID string) error
}

// CommandHandler handles special commands before they reach the agent
type CommandHandler struct {
	runtime   SessionResetter // Runtime for session management
	workspace string          // Workspace path for listing skills
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(runtime SessionResetter, workspace string) *CommandHandler {
	return &CommandHandler{
		runtime:   runtime,
		workspace: workspace,
	}
}

// CommandResult represents the result of command processing
type CommandResult struct {
	Handled  bool   // Whether the command was handled
	Response string // Response message to send back
}

// HandleCommand processes special commands and returns whether it was handled
func (h *CommandHandler) HandleCommand(msg bus.InboundMessage) CommandResult {
	content := strings.TrimSpace(msg.Content)
	
	// Check if it's a command (starts with /)
	if !strings.HasPrefix(content, "/") {
		return CommandResult{Handled: false}
	}

	// Extract command and args
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return CommandResult{Handled: false}
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/start":
		return CommandResult{
			Handled: true,
			Response: h.handleStart(),
		}
	case "/help":
		return CommandResult{
			Handled: true,
			Response: h.handleHelp(),
		}
	case "/reset":
		return CommandResult{
			Handled: true,
			Response: h.handleReset(msg.SessionKey()),
		}
	// /restart command removed - Telegram polling mode has connection conflicts on restart
	// Use process manager (systemd/launchd) or external restart script instead
	case "/skill":
		// Handle /skill list
		if len(parts) > 1 && strings.ToLower(parts[1]) == "list" {
			return CommandResult{
				Handled: true,
				Response: h.handleSkillList(),
			}
		}
		// Unknown /skill subcommand
		return CommandResult{
			Handled: true,
			Response: "❓ Unknown command. Use `/skill list` to see available skills.",
		}
	default:
		// Unknown command - return friendly message instead of passing to agent
		return CommandResult{
			Handled: true,
			Response: fmt.Sprintf("❓ Unknown command: %s\n\nUse /help to see available commands.", command),
		}
	}
}

func (h *CommandHandler) handleStart() string {
	return `🚀 **Aevitas Activated**

Advanced Evolutionary Virtual Intelligence with Temporal Awareness System online and operational.

Type /help to see what I can do.`
}

func (h *CommandHandler) handleHelp() string {
	return `📚 **Aevitas Capabilities**

**Core Tools:**
• File Operations: Read, Write, Edit files
• Web: Search (Brave API), Fetch web pages  
• Code: Execute commands, Search text, Find files
• Vision: Analyze images (multimodal support)

**Skills:**
Use /skill list to see installed skills

**Commands:**
• /start - Welcome message
• /help - Show this help  
• /skill list - List installed skills
• /reset - Clear conversation history

**Multimodal:**
Send images with text - I can analyze photos, diagrams, screenshots, etc.

Just send a message or image to get started!`
}

func (h *CommandHandler) handleReset(sessionKey string) string {
	if h.runtime == nil {
		return "⚠️ Session reset is not available"
	}

	if err := h.runtime.ClearSession(sessionKey); err != nil {
		return fmt.Sprintf("❌ Failed to reset session: %v", err)
	}

	return "✅ **Session Reset Successfully**\n\nLet's start fresh!"
}


func (h *CommandHandler) handleSkillList() string {
	if h.workspace == "" {
		return "⚠️ Skill listing is not available (workspace not configured)"
	}

	skills, err := utils.ListInstalledSkills(h.workspace)
	if err != nil {
		return fmt.Sprintf("❌ Failed to list skills: %v", err)
	}

	if len(skills) == 0 {
		return "📦 **Installed Skills**\n\nNo skills are currently installed.\n\nYou can install skills using the `Skill` tool or ask me to help you create new ones!"
	}

	var sb strings.Builder
	sb.WriteString("📦 **Installed Skills**\n\n")
	for i, skill := range skills {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, skill))
	}
	sb.WriteString("\nUse the `Skill` tool to load a skill's documentation, or ask me about what a specific skill can do!")

	return sb.String()
}

