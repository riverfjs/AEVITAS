package heartbeat

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	workspace   string
	onHeartbeat func(prompt string) (string, error)
	interval    time.Duration
}

func New(workspace string, onHB func(string) (string, error), interval time.Duration) *Service {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &Service{
		workspace:   workspace,
		onHeartbeat: onHB,
		interval:    interval,
	}
}

func (s *Service) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("[heartbeat] started, interval=%s", s.interval)

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-ctx.Done():
			log.Printf("[heartbeat] stopped")
			return nil
		}
	}
}

func (s *Service) tick() {
	hbPath := filepath.Join(s.workspace, "HEARTBEAT.md")
	data, err := os.ReadFile(hbPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[heartbeat] read error: %v", err)
		}
		return
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return
	}

	log.Printf("[heartbeat] triggering with prompt (%d chars)", len(content))

	if s.onHeartbeat == nil {
		log.Printf("[heartbeat] no handler set")
		return
	}

	result, err := s.onHeartbeat(content)
	if err != nil {
		log.Printf("[heartbeat] error: %v", err)
		return
	}

	if strings.Contains(result, "HEARTBEAT_OK") {
		log.Printf("[heartbeat] nothing to do")
	} else {
		log.Printf("[heartbeat] result: %s", truncate(result, 200))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
