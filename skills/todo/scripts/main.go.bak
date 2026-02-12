package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Task represents a todo item
type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	DueDate     time.Time `json:"due_date,omitempty"`
	HasReminder bool      `json:"has_reminder"`
}

// TodoList represents a collection of tasks
type TodoList struct {
	Tasks    []Task `json:"tasks"`
	NextID   int    `json:"next_id"`
	FilePath string `json:"-"`
}

// NewTodoList creates a new todo list
func NewTodoList(filePath string) *TodoList {
	return &TodoList{
		Tasks:    []Task{},
		NextID:   1,
		FilePath: filePath,
	}
}

// Load loads tasks from the JSON file
func (tl *TodoList) Load() error {
	// Create directory if it does not exist
	dir := filepath.Dir(tl.FilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Check if file exists
	if _, err := os.Stat(tl.FilePath); os.IsNotExist(err) {
		return tl.Save() // Create empty file
	}

	// Read file
	data, err := os.ReadFile(tl.FilePath)
	if err != nil {
		return err
	}

	// Empty file case
	if len(data) == 0 {
		return nil
	}

	// Parse JSON
	return json.Unmarshal(data, tl)
}

// Save saves tasks to the JSON file
func (tl *TodoList) Save() error {
	// Create directory if it does not exist
	dir := filepath.Dir(tl.FilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Marshal JSON
	data, err := json.MarshalIndent(tl, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(tl.FilePath, data, 0644)
}

// AddTask adds a new task
func (tl *TodoList) AddTask(description string, dueDate time.Time) (Task, error) {
	hasReminder := !dueDate.IsZero()
	
	task := Task{
		ID:          tl.NextID,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
		DueDate:     dueDate,
		HasReminder: hasReminder,
	}

	tl.Tasks = append(tl.Tasks, task)
	tl.NextID++

	if err := tl.Save(); err != nil {
		return Task{}, err
	}

	return task, nil
}

// CompleteTask marks a task as completed
func (tl *TodoList) CompleteTask(id int) error {
	for i := range tl.Tasks {
		if tl.Tasks[i].ID == id {
			tl.Tasks[i].Completed = true
			tl.Tasks[i].CompletedAt = time.Now()
			return tl.Save()
		}
	}
	return fmt.Errorf("task with ID %d not found", id)
}

// DeleteTask removes a task
func (tl *TodoList) DeleteTask(id int) error {
	for i, task := range tl.Tasks {
		if task.ID == id {
			// Remove the task
			tl.Tasks = append(tl.Tasks[:i], tl.Tasks[i+1:]...)
			return tl.Save()
		}
	}
	return fmt.Errorf("task with ID %d not found", id)
}

// CheckReminders checks for due tasks and returns those that are due
func (tl *TodoList) CheckReminders() []Task {
	now := time.Now()
	var dueTasks []Task

	for _, task := range tl.Tasks {
		// Skip completed tasks or tasks without reminders
		if task.Completed || !task.HasReminder {
			continue
		}

		// Check if the task is due (past due date)
		if !task.DueDate.IsZero() && task.DueDate.Before(now) {
			dueTasks = append(dueTasks, task)
		}
	}

	return dueTasks
}

func main() {
	fmt.Println("Todo Management Tool")
	
	// Set up the data file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not determine home directory: %v\n", err)
		os.Exit(1)
	}
	
	dataPath := filepath.Join(homeDir, ".myclaw/workspace/.claude/skills/todo/data/tasks.json")
	todoList := NewTodoList(dataPath)
	
	// Load existing tasks
	if err := todoList.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	// Check for command line arguments
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	
	switch command {
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing task description")
			return
		}
		
		description := os.Args[2]
		
		// Check for due date parameter
		var dueDate time.Time
		if len(os.Args) >= 5 && os.Args[3] == "--due" {
			// Parse the due date (format: YYYY-MM-DD)
			date, err := time.Parse("2006-01-02", os.Args[4])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Invalid date format. Use YYYY-MM-DD\n")
				return
			}
			dueDate = date
		}
		
		task, err := todoList.AddTask(description, dueDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding task: %v\n", err)
			return
		}
		
		fmt.Printf("Added task #%d: %s\n", task.ID, task.Description)
		if !task.DueDate.IsZero() {
			fmt.Printf("  Due date: %s\n", task.DueDate.Format("2006-01-02"))
		}
		
	case "list":
		if len(todoList.Tasks) == 0 {
			fmt.Println("No tasks found.")
			return
		}
		
		fmt.Println("Tasks:")
		for _, task := range todoList.Tasks {
			status := "[ ]"
			if task.Completed {
				status = "[✓]"
			}
			
			fmt.Printf("%d. %s %s", task.ID, status, task.Description)
			
			if !task.DueDate.IsZero() {
				dueStr := task.DueDate.Format("2006-01-02")
				if time.Now().After(task.DueDate) && !task.Completed {
					fmt.Printf(" (OVERDUE: %s)", dueStr)
				} else {
					fmt.Printf(" (Due: %s)", dueStr)
				}
			}
			
			fmt.Println()
		}
		
	case "complete":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing task ID")
			return
		}
		
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid task ID: %v\n", err)
			return
		}
		
		if err := todoList.CompleteTask(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		
		fmt.Printf("Marked task #%d as completed\n", id)
		
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing task ID")
			return
		}
		
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid task ID: %v\n", err)
			return
		}
		
		if err := todoList.DeleteTask(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		
		fmt.Printf("Deleted task #%d\n", id)
		
	case "reminders":
		dueTasks := todoList.CheckReminders()
		if len(dueTasks) == 0 {
			fmt.Println("No overdue tasks.")
			return
		}
		
		fmt.Println("Overdue tasks:")
		for _, task := range dueTasks {
			fmt.Printf("%d. %s (Due: %s)\n", 
				task.ID, 
				task.Description, 
				task.DueDate.Format("2006-01-02"))
		}
		
	case "help":
		printUsage()
		
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  todo add <description> [--due YYYY-MM-DD]  - Add a new task with optional due date")
	fmt.Println("  todo list                                  - List all tasks")
	fmt.Println("  todo complete <id>                         - Mark a task as completed")
	fmt.Println("  todo delete <id>                           - Delete a task")
	fmt.Println("  todo reminders                             - Show overdue tasks")
	fmt.Println("  todo help                                  - Show this help message")
}

// CheckUpcomingReminders returns tasks due within specific time windows
func (tl *TodoList) CheckUpcomingReminders() (urgentTasks []Task, regularTasks []Task) {
	now := time.Now()
	
	for _, task := range tl.Tasks {
		// Skip completed tasks or tasks without reminders
		if task.Completed || !task.HasReminder || task.DueDate.IsZero() {
			continue
		}

		// Skip already overdue tasks as they will be handled by CheckReminders()
		if task.DueDate.Before(now) {
			continue
		}
		
		// Calculate time until due
		timeUntilDue := task.DueDate.Sub(now)
		
		// Urgent reminder (due within 12 hours)
		if timeUntilDue <= 12*time.Hour {
			urgentTasks = append(urgentTasks, task)
		} else if timeUntilDue <= 24*time.Hour {
			// Regular reminder (due within 24 hours, but more than 12 hours)
			regularTasks = append(regularTasks, task)
		}
	}
	
	return urgentTasks, regularTasks
}

// FormatDuration formats a duration to a human-readable string
func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
