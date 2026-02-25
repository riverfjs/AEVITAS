package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	sdklogger "github.com/cexll/agentsdk-go/pkg/logger"
)

const dedupWindow = 24 * time.Hour

type Service struct {
	workspace   string
	onHeartbeat func(prompt string) (string, error)
	notifyFn    func(result string) // called when agent has something to say (not HEARTBEAT_OK)
	interval    time.Duration
	logger      sdklogger.Logger

	// deduplication: don't notify user with identical text within dedupWindow
	lastNotifiedText string
	lastNotifiedAt   time.Time
}

func New(
	workspace string,
	onHB func(string) (string, error),
	notifyFn func(string),
	interval time.Duration,
	logger sdklogger.Logger,
) *Service {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &Service{
		workspace:   workspace,
		onHeartbeat: onHB,
		notifyFn:    notifyFn,
		interval:    interval,
		logger:      logger,
	}
}

func (s *Service) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logf("[heartbeat] started, interval=%s", s.interval)

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-ctx.Done():
			s.logf("[heartbeat] stopped")
			return nil
		}
	}
}

func (s *Service) tick() {
	hbPath := filepath.Join(s.workspace, "HEARTBEAT.md")
	data, err := os.ReadFile(hbPath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logger.Errorf("[heartbeat] read error: %v", err)
		}
		return
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return
	}

	s.logf("[heartbeat] triggering with prompt (%d chars)", len(content))

	if s.onHeartbeat == nil {
		s.logf("[heartbeat] no handler set")
		return
	}

	result, err := s.onHeartbeat(content)
	if err != nil {
		s.logf("[heartbeat] error: %v", err)
		return
	}

	if strings.Contains(result, "HEARTBEAT_OK") {
		s.logf("[heartbeat] nothing to do")
		return
	}

	result = strings.TrimSpace(result)
	if result == "" {
		return
	}

	s.logf("[heartbeat] result: %s", truncate(result, 200))

	if s.notifyFn == nil {
		return
	}

	// Deduplication: skip if same text was sent within dedupWindow
	if result == s.lastNotifiedText && time.Since(s.lastNotifiedAt) < dedupWindow {
		s.logf("[heartbeat] skipping duplicate notification")
		return
	}

	s.notifyFn(result)
	s.lastNotifiedText = result
	s.lastNotifiedAt = time.Now()
}

func (s *Service) logf(format string, args ...any) {
	if s.logger != nil {
		s.logger.Infof(format, args...)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
