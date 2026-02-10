package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	sdklogger "github.com/cexll/agentsdk-go/pkg/logger"
)

type Service struct {
	workspace   string
	onHeartbeat func(prompt string) (string, error)
	interval    time.Duration
	logger      sdklogger.Logger
}

func New(workspace string, onHB func(string) (string, error), interval time.Duration, logger sdklogger.Logger) *Service {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &Service{
		workspace:   workspace,
		onHeartbeat: onHB,
		interval:    interval,
		logger:      logger,
	}
}

func (s *Service) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Infof("[heartbeat] started, interval=%s", s.interval)

	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-ctx.Done():
			s.logger.Infof("[heartbeat] stopped")
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

	s.logger.Infof("[heartbeat] triggering with prompt (%d chars)", len(content))

	if s.onHeartbeat == nil {
		s.logger.Warnf("[heartbeat] no handler set")
		return
	}

	result, err := s.onHeartbeat(content)
	if err != nil {
		s.logger.Errorf("[heartbeat] error: %v", err)
		return
	}

	if strings.Contains(result, "HEARTBEAT_OK") {
		s.logger.Infof("[heartbeat] nothing to do")
	} else {
		s.logger.Infof("[heartbeat] result: %s", truncate(result, 200))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
