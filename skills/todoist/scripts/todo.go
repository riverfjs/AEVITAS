package main

// todo.go — Task and TodoList types with persistence.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	DueDate     time.Time `json:"due_date,omitempty"`
	HasReminder bool      `json:"has_reminder"`
}

type TodoList struct {
	Tasks    []Task `json:"tasks"`
	NextID   int    `json:"next_id"`
	FilePath string `json:"-"`
}

func NewTodoList(filePath string) *TodoList {
	return &TodoList{Tasks: []Task{}, NextID: 1, FilePath: filePath}
}

func (tl *TodoList) Load() error {
	dir := filepath.Dir(tl.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(tl.FilePath); os.IsNotExist(err) {
		return tl.Save()
	}
	data, err := os.ReadFile(tl.FilePath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, tl)
}

func (tl *TodoList) Save() error {
	if err := os.MkdirAll(filepath.Dir(tl.FilePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tl.FilePath, data, 0644)
}

func (tl *TodoList) AddTask(description string, dueDate time.Time, cron *CronManager) (Task, error) {
	task := Task{
		ID:          tl.NextID,
		Description: description,
		Completed:   false,
		CreatedAt:   time.Now(),
		DueDate:     dueDate,
		HasReminder: !dueDate.IsZero(),
	}
	tl.Tasks = append(tl.Tasks, task)
	tl.NextID++
	if err := tl.Save(); err != nil {
		return Task{}, err
	}
	if task.HasReminder && cron != nil {
		if err := cron.AddReminderJob(task.ID, task.Description, task.DueDate); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create reminder: %v\n", err)
		}
	}
	return task, nil
}

func (tl *TodoList) CompleteTask(id int, cron *CronManager) error {
	for i := range tl.Tasks {
		if tl.Tasks[i].ID == id {
			tl.Tasks[i].Completed = true
			tl.Tasks[i].CompletedAt = time.Now()
			if cron != nil {
				_ = cron.RemoveReminderJob(id)
			}
			return tl.Save()
		}
	}
	return fmt.Errorf("task %d not found", id)
}

func (tl *TodoList) DeleteTask(id int, cron *CronManager) error {
	for i, t := range tl.Tasks {
		if t.ID == id {
			tl.Tasks = append(tl.Tasks[:i], tl.Tasks[i+1:]...)
			if cron != nil {
				_ = cron.RemoveReminderJob(id)
			}
			return tl.Save()
		}
	}
	return fmt.Errorf("task %d not found", id)
}

func (tl *TodoList) OverdueTasks() []Task {
	now := time.Now()
	var out []Task
	for _, t := range tl.Tasks {
		if !t.Completed && t.HasReminder && !t.DueDate.IsZero() && t.DueDate.Before(now) {
			out = append(out, t)
		}
	}
	return out
}
