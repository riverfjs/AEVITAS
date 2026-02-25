package main

// cron.go — CronJob types and CronManager.
// All cron operations go through the myclaw gateway WS RPC (see gateway.go).

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ── Types (mirror myclaw internal/cron/types.go) ──────────────────────────────

type CronSchedule struct {
	Kind    string `json:"kind"`              // "every" | "at" | "cron"
	Expr    string `json:"expr,omitempty"`    // cron expression (kind=cron)
	EveryMs int64  `json:"everyMs,omitempty"` // interval ms (kind=every)
	AtMs    int64  `json:"atMs,omitempty"`    // unix ms (kind=at)
}

type CronPayload struct {
	Kind    string `json:"kind"`              // "agentTurn" | "command"
	Message string `json:"message,omitempty"` // agentTurn
	Command string `json:"command,omitempty"` // command
}

type CronDelivery struct {
	Mode    string `json:"mode"`              // "announce" | "none"
	Channel string `json:"channel,omitempty"` // e.g. "telegram"
	To      string `json:"to,omitempty"`      // chat id
}

type CronJobState struct {
	NextRunAtMs int64  `json:"nextRunAtMs"`
	LastRunAtMs int64  `json:"lastRunAtMs"`
	LastStatus  string `json:"lastStatus"` // "ok" | "error"
	LastError   string `json:"lastError"`
}

type CronJob struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	Schedule       CronSchedule  `json:"schedule"`
	SessionTarget  string        `json:"sessionTarget,omitempty"`
	Payload        CronPayload   `json:"payload"`
	Delivery       *CronDelivery `json:"delivery,omitempty"`
	State          CronJobState  `json:"state"`
	DeleteAfterRun bool          `json:"deleteAfterRun"`
}

// ── CronManager ───────────────────────────────────────────────────────────────

type CronManager struct {
	chatID  string
	channel string
}

func NewCronManager(chatID, channel string) *CronManager {
	return &CronManager{chatID: chatID, channel: channel}
}

func (cm *CronManager) delivery() *CronDelivery {
	if cm.chatID == "" {
		return nil
	}
	ch := cm.channel
	if ch == "" {
		ch = "telegram"
	}
	return &CronDelivery{Mode: "announce", Channel: ch, To: cm.chatID}
}

// AddReminderJob schedules a one-shot reminder 2h before dueDate.
func (cm *CronManager) AddReminderJob(taskID int, description string, dueDate time.Time) error {
	reminderTime := dueDate.Add(-2 * time.Hour)
	if reminderTime.Before(time.Now()) {
		return nil
	}
	params := map[string]interface{}{
		"name":           fmt.Sprintf("todo-reminder-task-%d", taskID),
		"deleteAfterRun": true,
		"schedule":       CronSchedule{Kind: "at", AtMs: reminderTime.UnixMilli()},
		"payload": CronPayload{
			Kind:    "agentTurn",
			Message: fmt.Sprintf("⏰ 提醒：任务 #%d '%s' 将在2小时后到期\n截止时间: %s", taskID, description, dueDate.Format("2006-01-02 15:04")),
		},
		"delivery": cm.delivery(),
	}
	_, err := callGateway("cron.add", params)
	return err
}

// RemoveReminderJob removes a task reminder; tolerates "not found".
func (cm *CronManager) RemoveReminderJob(taskID int) error {
	_, err := callGateway("cron.remove", map[string]string{
		"id": fmt.Sprintf("todo-%d-reminder", taskID),
	})
	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil
	}
	return err
}

// AddRecurringJob adds an interval-based cron job that runs a shell command.
func (cm *CronManager) AddRecurringJob(name, command string, intervalMs int64) error {
	params := map[string]interface{}{
		"name":     name,
		"schedule": CronSchedule{Kind: "every", EveryMs: intervalMs},
		"payload":  CronPayload{Kind: "command", Command: command},
		"delivery": cm.delivery(),
	}
	_, err := callGateway("cron.add", params)
	return err
}

// ListJobs returns all cron jobs from the gateway.
func (cm *CronManager) ListJobs() ([]CronJob, error) {
	payload, err := callGateway("cron.list", map[string]bool{"includeDisabled": true})
	if err != nil {
		return nil, err
	}
	var result struct {
		Jobs []CronJob `json:"jobs"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("parse cron.list response: %w", err)
	}
	return result.Jobs, nil
}

// DeleteJob removes a cron job by ID.
func (cm *CronManager) DeleteJob(jobID string) error {
	_, err := callGateway("cron.remove", map[string]string{"id": jobID})
	return err
}

// RunJob immediately triggers a cron job by ID.
func (cm *CronManager) RunJob(jobID string) error {
	_, err := callGateway("cron.run", map[string]string{"id": jobID})
	return err
}
