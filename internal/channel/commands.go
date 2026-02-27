package channel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/riverfjs/aevitas/internal/bus"
	"github.com/riverfjs/aevitas/pkg/utils"
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
	Handled  bool     // Whether the command was handled
	Response string   // Response message to send back
	Files    []string // File paths to send (e.g., log files)
}

// HandleCommand processes special commands and returns whether it was handled.
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
			Handled:  true,
			Response: h.handleReset(msg.SessionKey()),
		}
	case "/restart":
		// Save chat info for restart notification
		restartInfo := fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)
		restartTriggerFile := filepath.Join(os.Getenv("HOME"), ".aevitas", "restart_trigger.txt")
		os.WriteFile(restartTriggerFile, []byte(restartInfo), 0644)
		
		return CommandResult{
			Handled: true,
			Response: h.handleRestart(),
		}
	case "/logs":
		// Parse argument: number or "all"
		arg := "100" // default 100 lines
		if len(parts) > 1 {
			arg = strings.ToLower(parts[1])
		}
		return h.handleLogs(arg)
	case "/status":
		return CommandResult{
			Handled: true,
			Response: h.handleStatus(),
		}
	case "/chatid":
		return CommandResult{
			Handled:  true,
			Response: fmt.Sprintf("💬 **Your Chat Information**\n\nChannel: %s\nChat ID: `%s`\nSender ID: `%s`", msg.Channel, msg.ChatID, msg.SenderID),
		}
	case "/cleanup":
		// Check if user is confirming previous cleanup request
		if len(parts) > 1 && (strings.ToLower(parts[1]) == "confirm" || strings.ToLower(parts[1]) == "yes") {
			return h.handleCleanupConfirm(msg.ChatID)
		}
		// Initial cleanup request - scan and show stats
		return h.handleCleanupScan(msg.ChatID)
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
• /restart - Restart gateway (production only)
• /logs [lines|all] - Show logs (default 100 lines, max 1000, or "all" for full file)
• /status - Show gateway status
• /chatid - Show your chat ID
• /cleanup - Clean temporary screenshot files (requires confirmation)

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
	return "✅ **Session Reset**\n\nLet's start fresh!"
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

func (h *CommandHandler) handleRestart() string {
	scriptPath := filepath.Join(os.Getenv("HOME"), ".aevitas", "bin", "..", "..", "Documents", "chatbot", "aevitas", "scripts", "restart.sh")
	
	// Check if restart script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "⚠️ Restart script not found. This command only works in production mode.\n\nUse `make prod` to install and `scripts/start.sh` to run in background."
	}
	
	// Execute restart script in background
	cmd := exec.Command("/bin/bash", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("❌ Failed to restart: %v", err)
	}
	
	return "🔄 **Restarting Gateway**\n\nThe gateway will restart in a few seconds. You'll receive a notification when it's back online."
}

func (h *CommandHandler) handleLogs(arg string) CommandResult {
	// Always use home directory for logs in production
	logFile := filepath.Join(os.Getenv("HOME"), ".aevitas", "workspace", "logs", "aevitas.log")
	
	// Check if log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return CommandResult{
			Handled:  true,
			Response: "⚠️ Log file not found at: " + logFile,
		}
	}
	
	// Check if user wants full file
	if arg == "all" {
		return CommandResult{
			Handled:  true,
			Response: "📄 **Gateway Logs (Full File)**\n\nSending complete log file...",
			Files:    []string{logFile},
		}
	}
	
	// Parse number of lines (default 100)
	lines := 100
	if n, err := strconv.Atoi(arg); err == nil && n > 0 {
		if n > 1000 {
			lines = 1000 // Max 1000 lines
		} else {
			lines = n
		}
	}
	
	// Read last N lines
	content, err := readLastLines(logFile, lines)
	if err != nil {
		return CommandResult{
			Handled:  true,
			Response: fmt.Sprintf("❌ Failed to read log file: %v", err),
		}
	}
	
	response := fmt.Sprintf("📄 **Gateway Logs (Last %d lines)**\n\n```\n%s\n```", lines, content)
	
	return CommandResult{
		Handled:  true,
		Response: response,
	}
}

// readLastLines reads the last n lines from a file
func readLastLines(filePath string, n int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return "", err
	}
	fileSize := stat.Size()
	
	// Read file in chunks from the end
	const bufSize = 4096
	var lines []string
	var buffer []byte
	
	for offset := fileSize; offset > 0 && len(lines) < n; {
		// Calculate chunk size
		chunkSize := int64(bufSize)
		if offset < chunkSize {
			chunkSize = offset
		}
		
		// Seek to position
		offset -= chunkSize
		_, err := file.Seek(offset, 0)
		if err != nil {
			return "", err
		}
		
		// Read chunk
		chunk := make([]byte, chunkSize)
		_, err = file.Read(chunk)
		if err != nil {
			return "", err
		}
		
		// Prepend to buffer
		buffer = append(chunk, buffer...)
		
		// Split into lines
		text := string(buffer)
		allLines := strings.Split(text, "\n")
		
		// If we're not at the beginning of the file, keep incomplete first line in buffer
		if offset > 0 && len(allLines) > 0 {
			buffer = []byte(allLines[0])
			allLines = allLines[1:]
		} else {
			buffer = nil
		}
		
		// Prepend lines
		lines = append(allLines, lines...)
	}
	
	// Get last n lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	
	// Remove empty trailing line
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	
	return strings.Join(lines, "\n"), nil
}

func (h *CommandHandler) handleStatus() string {
	pidFile := filepath.Join(os.Getenv("HOME"), ".aevitas", "aevitas.pid")
	
	// Check if PID file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		// Maybe running in foreground, show current process PID
		currentPID := os.Getpid()
		return fmt.Sprintf("🟡 **Gateway Status: Running (Foreground)**\n\nCurrent PID: %d\n\nNo PID file found - gateway may be running in foreground mode.", currentPID)
	}
	
	// Read PID
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Sprintf("⚠️ Failed to read PID file: %v", err)
	}
	
	pid := strings.TrimSpace(string(pidBytes))
	
	// Check if process is running
	cmd := exec.Command("ps", "-p", pid, "-o", "pid,etime,command")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("🔴 **Gateway Status: Not Running**\n\nStale PID file found (PID: %s)\n\nUse `make start` or `/restart` to start.", pid)
	}
	
	// Parse ps output to get uptime
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var uptime string
	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 2 {
			uptime = fields[1] // ELAPSED time
		}
	}
	
	statusMsg := fmt.Sprintf("🟢 **Gateway Status: Running (Background)**\n\nPID: %s", pid)
	if uptime != "" {
		statusMsg += fmt.Sprintf("\nUptime: %s", uptime)
	}
	
	return statusMsg
}

// handleCleanupScan scans for temporary screenshot files and shows statistics
func (h *CommandHandler) handleCleanupScan(chatID string) CommandResult {
	// Scan for screenshot files in common temp directories
	var tempFiles []string
	var totalSize int64
	
	// Check /tmp and OS temp dir
	tempDirs := []string{
		os.TempDir(),
		"/tmp",
	}
	
	for _, dir := range tempDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		
		// Find screenshot-*.png files
		pattern := filepath.Join(dir, "screenshot-*.png")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		
		for _, file := range matches {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			tempFiles = append(tempFiles, file)
			totalSize += info.Size()
		}
	}
	
	if len(tempFiles) == 0 {
		return CommandResult{
			Handled:  true,
			Response: "✨ **No Temporary Files Found**\n\nYour system is clean!",
		}
	}
	
	// Get oldest and newest file times
	var oldestTime, newestTime int64
	for _, file := range tempFiles {
		info, _ := os.Stat(file)
		modTime := info.ModTime().Unix()
		if oldestTime == 0 || modTime < oldestTime {
			oldestTime = modTime
		}
		if modTime > newestTime {
			newestTime = modTime
		}
	}
	
	// Save pending cleanup list to temp file
	cleanupFile := filepath.Join(os.TempDir(), fmt.Sprintf("cleanup_%s.txt", chatID))
	data := strings.Join(tempFiles, "\n")
	if err := os.WriteFile(cleanupFile, []byte(data), 0644); err != nil {
		return CommandResult{
			Handled:  true,
			Response: fmt.Sprintf("❌ Failed to save cleanup list: %v", err),
		}
	}
	
	// Format response
	response := "🗑️ **Temporary Files Found**\n\n"
	response += "📊 Statistics:\n"
	response += fmt.Sprintf("• Files: %d screenshot(s)\n", len(tempFiles))
	response += fmt.Sprintf("• Total Size: %.2f MB\n", float64(totalSize)/(1024*1024))
	response += fmt.Sprintf("• Oldest: %s\n", utils.FormatRelativeTime(oldestTime))
	response += fmt.Sprintf("• Newest: %s\n\n", utils.FormatRelativeTime(newestTime))
	response += "⚠️ **Warning**: This action cannot be undone!\n\n"
	response += "Reply with `/cleanup confirm` or `/cleanup yes` to delete these files."
	
	return CommandResult{
		Handled:  true,
		Response: response,
	}
}

// handleCleanupConfirm deletes the pending cleanup files
func (h *CommandHandler) handleCleanupConfirm(chatID string) CommandResult {
	// Load pending cleanup list
	cleanupFile := filepath.Join(os.TempDir(), fmt.Sprintf("cleanup_%s.txt", chatID))
	data, err := os.ReadFile(cleanupFile)
	if err != nil {
		return CommandResult{
			Handled:  true,
			Response: "⚠️ No pending cleanup request found or it has expired.\n\nUse `/cleanup` to scan for temporary files first.",
		}
	}
	
	tempFiles := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(tempFiles) == 0 {
		return CommandResult{
			Handled:  true,
			Response: "⚠️ No files to clean.",
		}
	}
	
	// Delete files
	deletedCount := 0
	var failedFiles []string
	
	for _, file := range tempFiles {
		if file == "" {
			continue
		}
		if err := os.Remove(file); err != nil {
			failedFiles = append(failedFiles, filepath.Base(file))
		} else {
			deletedCount++
		}
	}
	
	// Remove cleanup list file
	os.Remove(cleanupFile)
	
	// Format response
	response := fmt.Sprintf("✅ **Cleanup Complete**\n\nDeleted %d file(s)", deletedCount)
	if len(failedFiles) > 0 {
		response += fmt.Sprintf("\n\n⚠️ Failed to delete %d file(s):\n%s", len(failedFiles), strings.Join(failedFiles, ", "))
	}
	
	return CommandResult{
		Handled:  true,
		Response: response,
	}
}

