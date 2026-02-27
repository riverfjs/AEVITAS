package main

// main.go — CLI entry point for the todoist skill.
// Delegates to todo.go (tasks), cron.go (cron jobs), gateway.go (RPC client).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
}

func loadConfig(path string) (*Config, error) {
	cfg := &Config{Channel: "telegram"}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, json.Unmarshal(data, cfg)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fatalf("could not determine home directory: %v", err)
	}

	skillRoot := filepath.Join(homeDir, ".aevitas/workspace/.claude/skills/todoist")
	dataPath := filepath.Join(skillRoot, "data/tasks.json")
	cfgPath := filepath.Join(skillRoot, "config.json")

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &Config{Channel: "telegram"}
	}

	todos := NewTodoList(dataPath)
	if err := todos.Load(); err != nil {
		fatalf("failed to load tasks: %v", err)
	}

	var cron *CronManager
	if cfg.ChatID != "" {
		cron = NewCronManager(cfg.ChatID, cfg.Channel)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	// ── Task commands ──────────────────────────────────────────────────────────
	case "add":
		runAdd(todos, cron, args)
	case "list":
		runList(todos)
	case "complete":
		runComplete(todos, cron, args)
	case "delete":
		runDelete(todos, cron, args)
	case "reminders":
		runReminders(todos)

	// ── Cron commands (via gateway WS RPC) ────────────────────────────────────
	case "cron-add":
		runCronAdd(cron, cfg, args)
	case "cron-list":
		runCronList(cron, cfg)
	case "cron-delete":
		runCronDelete(cron, cfg, args)
	case "cron-run":
		runCronRun(cron, cfg, args)

	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// ── Task handlers ──────────────────────────────────────────────────────────────

func runAdd(todos *TodoList, cron *CronManager, args []string) {
	if len(args) < 1 {
		fatalf("Usage: todoist add <description> [--due YYYY-MM-DD]")
	}
	description := args[0]
	var dueDate time.Time
	if len(args) >= 3 && args[1] == "--due" {
		d, err := time.Parse("2006-01-02", args[2])
		if err != nil {
			fatalf("invalid date format, use YYYY-MM-DD")
		}
		dueDate = d
	}
	task, err := todos.AddTask(description, dueDate, cron)
	if err != nil {
		fatalf("add task: %v", err)
	}
	fmt.Printf("Added task #%d: %s\n", task.ID, task.Description)
	if !task.DueDate.IsZero() {
		fmt.Printf("  Due: %s\n", task.DueDate.Format("2006-01-02"))
	}
}

func runList(todos *TodoList) {
	if len(todos.Tasks) == 0 {
		fmt.Println("No tasks.")
		return
	}
	for _, t := range todos.Tasks {
		status := "[ ]"
		if t.Completed {
			status = "[✓]"
		}
		fmt.Printf("%d. %s %s", t.ID, status, t.Description)
		if !t.DueDate.IsZero() {
			due := t.DueDate.Format("2006-01-02")
			if !t.Completed && time.Now().After(t.DueDate) {
				fmt.Printf(" (OVERDUE: %s)", due)
			} else {
				fmt.Printf(" (Due: %s)", due)
			}
		}
		fmt.Println()
	}
}

func runComplete(todos *TodoList, cron *CronManager, args []string) {
	id := requireID(args, "complete")
	if err := todos.CompleteTask(id, cron); err != nil {
		fatalf("%v", err)
	}
	fmt.Printf("Completed task #%d\n", id)
}

func runDelete(todos *TodoList, cron *CronManager, args []string) {
	id := requireID(args, "delete")
	if err := todos.DeleteTask(id, cron); err != nil {
		fatalf("%v", err)
	}
	fmt.Printf("Deleted task #%d\n", id)
}

func runReminders(todos *TodoList) {
	tasks := todos.OverdueTasks()
	if len(tasks) == 0 {
		fmt.Println("No overdue tasks.")
		return
	}
	fmt.Println("Overdue tasks:")
	for _, t := range tasks {
		fmt.Printf("  %d. %s (Due: %s)\n", t.ID, t.Description, t.DueDate.Format("2006-01-02"))
	}
}

// ── Cron handlers ──────────────────────────────────────────────────────────────

func ensureCron(cron *CronManager, cfg *Config) *CronManager {
	if cron != nil {
		return cron
	}
	return NewCronManager(cfg.ChatID, cfg.Channel)
}

func runCronAdd(cron *CronManager, cfg *Config, args []string) {
	if len(args) < 3 {
		fatalf("Usage: todoist cron-add <name> <command> <interval-ms>")
	}
	name := args[0]
	command := args[1]
	intervalMs, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		fatalf("invalid interval: %v", err)
	}
	if err := ensureCron(cron, cfg).AddRecurringJob(name, command, intervalMs); err != nil {
		fatalf("cron-add: %v", err)
	}
	fmt.Printf("Added cron job: %s\n", name)
	fmt.Printf("  Command:  %s\n", command)
	fmt.Printf("  Interval: %dms (%.1fh)\n", intervalMs, float64(intervalMs)/3_600_000)
}

func runCronList(cron *CronManager, cfg *Config) {
	jobs, err := ensureCron(cron, cfg).ListJobs()
	if err != nil {
		fatalf("cron-list: %v", err)
	}
	if len(jobs) == 0 {
		fmt.Println("No cron jobs.")
		return
	}
	for i, j := range jobs {
		fmt.Printf("\n[%d] %s  (id: %s)\n", i+1, j.Name, j.ID)
		fmt.Printf("    enabled: %v  schedule: %s", j.Enabled, j.Schedule.Kind)
		switch j.Schedule.Kind {
		case "every":
			fmt.Printf(" (every %.1fh)", float64(j.Schedule.EveryMs)/3_600_000)
		case "at":
			fmt.Printf(" (%s)", time.UnixMilli(j.Schedule.AtMs).Format("2006-01-02 15:04"))
		case "cron":
			fmt.Printf(" (%s)", j.Schedule.Expr)
		}
		fmt.Println()
		if j.State.LastRunAtMs > 0 {
			fmt.Printf("    last run: %s  status: %s\n",
				time.UnixMilli(j.State.LastRunAtMs).Format("2006-01-02 15:04:05"),
				j.State.LastStatus)
		}
	}
}

func runCronDelete(cron *CronManager, cfg *Config, args []string) {
	if len(args) < 1 {
		fatalf("Usage: todoist cron-delete <job-id>")
	}
	if err := ensureCron(cron, cfg).DeleteJob(args[0]); err != nil {
		fatalf("cron-delete: %v", err)
	}
	fmt.Printf("Deleted cron job: %s\n", args[0])
}

func runCronRun(cron *CronManager, cfg *Config, args []string) {
	if len(args) < 1 {
		fatalf("Usage: todoist cron-run <job-id>")
	}
	if err := ensureCron(cron, cfg).RunJob(args[0]); err != nil {
		fatalf("cron-run: %v", err)
	}
	fmt.Printf("Triggered job: %s\n", args[0])
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func requireID(args []string, cmd string) int {
	if len(args) < 1 {
		fatalf("Usage: todoist %s <id>", cmd)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fatalf("invalid id: %v", err)
	}
	return id
}

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
	os.Exit(1)
}

func printUsage() {
	fmt.Print(`Todoist — task management + cron scheduling via aevitas gateway

Task commands:
  todoist add <description> [--due YYYY-MM-DD]   Add a task
  todoist list                                    List all tasks
  todoist complete <id>                           Mark task complete
  todoist delete <id>                             Delete a task
  todoist reminders                               Show overdue tasks

Cron commands (WebSocket RPC → aevitas gateway ws://127.0.0.1:18790):
  todoist cron-add <name> <command> <ms>          Add recurring job
  todoist cron-list                               List all cron jobs
  todoist cron-delete <id>                        Delete a cron job
  todoist cron-run <id>                           Trigger job immediately

Intervals:
  1h  = 3600000    6h = 21600000    24h = 86400000

Examples:
  todoist add "Book flights" --due 2026-04-01
  todoist cron-add "Flight Monitor" "bash ~/.aevitas/.../monitor.sh check-all" 21600000
  todoist cron-run flight-monitor-auto
  todoist cron-list
`)
}
