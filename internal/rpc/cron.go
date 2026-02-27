package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/riverfjs/aevitas/internal/cron"
)

// RegisterCronHandlers registers all cron.* RPC methods on s.
// Mirrors openclaw's server-methods/cron.ts handler set.
func RegisterCronHandlers(s *Server, svc *cron.Service) {
	// cron.list → ListJobs()
	// params: { includeDisabled?: bool }
	s.Register("cron.list", func(params json.RawMessage, respond RespondFn) {
		jobs := svc.ListJobs()
		respond(true, map[string]interface{}{"jobs": jobs}, "")
	})

	// cron.run → RunJob(id)
	// params: { id: string }
	s.Register("cron.run", func(params json.RawMessage, respond RespondFn) {
		var p struct {
			ID    string `json:"id"`
			JobID string `json:"jobId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			respond(false, nil, fmt.Sprintf("invalid params: %v", err))
			return
		}
		id := p.ID
		if id == "" {
			id = p.JobID
		}
		if id == "" {
			respond(false, nil, "missing id")
			return
		}
		if err := svc.RunJob(id); err != nil {
			respond(false, nil, err.Error())
			return
		}
		respond(true, map[string]interface{}{"ok": true, "id": id}, "")
	})

	// cron.add → AddJob(name, schedule, payload)
	// params: { name, schedule, payload, sessionTarget?, delivery?, deleteAfterRun? }
	s.Register("cron.add", func(params json.RawMessage, respond RespondFn) {
		var p struct {
			Name           string            `json:"name"`
			Schedule       cron.Schedule     `json:"schedule"`
			Payload        cron.Payload      `json:"payload"`
			SessionTarget  cron.SessionTarget `json:"sessionTarget"`
			Delivery       *cron.Delivery    `json:"delivery"`
			DeleteAfterRun bool              `json:"deleteAfterRun"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			respond(false, nil, fmt.Sprintf("invalid params: %v", err))
			return
		}
		if p.Name == "" {
			respond(false, nil, "missing name")
			return
		}
		job, err := svc.AddJobWithOptions(p.Name, p.Schedule, p.Payload, cron.AddJobOptions{
			SessionTarget:  p.SessionTarget,
			Delivery:       p.Delivery,
			DeleteAfterRun: p.DeleteAfterRun,
		})
		if err != nil {
			respond(false, nil, err.Error())
			return
		}
		respond(true, job, "")
	})

	// cron.remove → RemoveJob(id)
	// params: { id: string }
	s.Register("cron.remove", func(params json.RawMessage, respond RespondFn) {
		var p struct {
			ID    string `json:"id"`
			JobID string `json:"jobId"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			respond(false, nil, fmt.Sprintf("invalid params: %v", err))
			return
		}
		id := p.ID
		if id == "" {
			id = p.JobID
		}
		if id == "" {
			respond(false, nil, "missing id")
			return
		}
		ok := svc.RemoveJob(id)
		if !ok {
			respond(false, nil, fmt.Sprintf("job %s not found", id))
			return
		}
		respond(true, map[string]interface{}{"ok": true, "id": id}, "")
	})

	// cron.enable → EnableJob(id, enabled)
	// params: { id: string, enabled: bool }
	s.Register("cron.enable", func(params json.RawMessage, respond RespondFn) {
		var p struct {
			ID      string `json:"id"`
			JobID   string `json:"jobId"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			respond(false, nil, fmt.Sprintf("invalid params: %v", err))
			return
		}
		id := p.ID
		if id == "" {
			id = p.JobID
		}
		if id == "" {
			respond(false, nil, "missing id")
			return
		}
		job, err := svc.EnableJob(id, p.Enabled)
		if err != nil {
			respond(false, nil, err.Error())
			return
		}
		respond(true, job, "")
	})
}
