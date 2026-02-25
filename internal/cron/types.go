package cron

import (
	"crypto/rand"
	"fmt"
)

// Schedule ─────────────────────────────────────────────────────────────────

type Schedule struct {
	Kind     string `json:"kind"`              // "cron" | "every" | "at"
	Expr     string `json:"expr,omitempty"`    // cron expression (kind=cron)
	EveryMs  int64  `json:"everyMs,omitempty"` // interval ms (kind=every)
	AnchorMs int64  `json:"anchorMs,omitempty"`
	AtMs     int64  `json:"atMs,omitempty"` // one-shot unix ms (kind=at)
}

// Payload ──────────────────────────────────────────────────────────────────
// kind = "systemEvent" : inject Text as system message into session (no agent turn)
// kind = "agentTurn"   : run agent with Message; result can be delivered
// kind = "command"     : exec Command via bash, stdout is delivered directly

type Payload struct {
	Kind    string `json:"kind"`              // "systemEvent" | "agentTurn" | "command"
	Text    string `json:"text,omitempty"`    // systemEvent
	Message string `json:"message,omitempty"` // agentTurn (legacy field also accepted)
	Command string `json:"command,omitempty"` // command
}

// SessionTarget ────────────────────────────────────────────────────────────

type SessionTarget string

const (
	SessionMain     SessionTarget = "main"
	SessionIsolated SessionTarget = "isolated"
)

// Delivery ─────────────────────────────────────────────────────────────────
// mode = "announce" : send result to Channel/To
// mode = "none"     : discard result

type Delivery struct {
	Mode    string `json:"mode"`              // "announce" | "none"
	Channel string `json:"channel,omitempty"` // e.g. "telegram"
	To      string `json:"to,omitempty"`      // chat id
}

// JobState / CronJob ───────────────────────────────────────────────────────

type JobState struct {
	NextRunAtMs int64  `json:"nextRunAtMs"`
	LastRunAtMs int64  `json:"lastRunAtMs"`
	LastStatus  string `json:"lastStatus"` // "ok" | "error"
	LastError   string `json:"lastError"`
}

type CronJob struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Enabled        bool          `json:"enabled"`
	Schedule       Schedule      `json:"schedule"`
	SessionTarget  SessionTarget `json:"sessionTarget,omitempty"` // default: "main"
	Payload        Payload       `json:"payload"`
	Delivery       *Delivery     `json:"delivery,omitempty"`
	State          JobState      `json:"state"`
	DeleteAfterRun bool          `json:"deleteAfterRun"`
}

// effectiveDelivery resolves delivery config, supporting the legacy flat fields
// (payload.deliver / payload.channel / payload.to) for backward compatibility.
func (j CronJob) effectiveDelivery() *Delivery {
	if j.Delivery != nil {
		return j.Delivery
	}
	return nil
}

func NewCronJob(name string, schedule Schedule, payload Payload) CronJob {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return CronJob{
		ID:            fmt.Sprintf("%x", b),
		Name:          name,
		Enabled:       true,
		SessionTarget: SessionMain,
		Schedule:      schedule,
		Payload:       payload,
	}
}
